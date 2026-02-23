package forwarder

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	agt "github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/manager/database"
	"github.com/jackadi-io/jackadi/internal/proto"
)

// GRPCForwarder simply forwards tasks received from one component to another component.
//
// The main usage is for the Manager to forward CLI requests to the agents.
type GRPCForwarder struct {
	proto.UnimplementedForwarderServer
	taskDispatcher Dispatcher[*proto.TaskRequest, *proto.TaskResponse]
	db             *badger.DB
}

func New(taskDispatcher Dispatcher[*proto.TaskRequest, *proto.TaskResponse], db *badger.DB) GRPCForwarder {
	return GRPCForwarder{
		taskDispatcher: taskDispatcher,
		db:             db,
	}
}

func (f *GRPCForwarder) storeRequest(req *proto.TaskRequest, targetsStatus map[string]bool) {
	dbReq := database.Request{Task: req.GetTask()}
	for target, connected := range targetsStatus {
		if connected {
			dbReq.ConnectedTarget = append(dbReq.ConnectedTarget, target)
		} else {
			dbReq.DisconnectedTarget = append(dbReq.DisconnectedTarget, target)
		}
	}

	data, err := database.MarshalRequest(&dbReq)
	if err != nil {
		slog.Error("unable to record result", "error", "marshal error")
		return
	}

	dbDerr := f.db.Update(func(txn *badger.Txn) error {
		var key []byte
		if req.GetGroupID() > 0 {
			key = database.GenerateRequestKey(req.GetGroupID())
		} else {
			key = database.GenerateRequestKey(req.GetId())
		}

		singleEntry := badger.NewEntry(key, data).WithTTL(config.DBTaskRequestTTL)
		if err := txn.SetEntry(singleEntry); err != nil {
			slog.Error("unable to record result", "error", err)
			return err
		}

		return nil
	})
	if dbDerr != nil {
		slog.Warn("failed to store task request", "error", dbDerr)
	}
}

// ExecTask gets the request from the requester (e.g. the CLI), and send it to the manager's stream.
//
// The manager's stream is linked to a single agent.
func (f *GRPCForwarder) ExecTask(ctx context.Context, req *proto.TaskRequest) (*proto.FwdResponse, error) {
	targetsStatus, err := f.taskDispatcher.TargetedAgents(req.GetTarget(), req.GetTargetMode())
	if err != nil {
		return nil, err
	}

	// in theory this lock is useless as we are not supposed to receive multiple responses
	// from the same agent for a same request. Better safe than sorry.
	lock := sync.Mutex{}
	results := make(map[string]*proto.TaskResponse, len(targetsStatus))

	// the group ID enables to get all responses when the request is targeting multiple agents
	groupID := time.Now().UnixNano()
	req.GroupID = &groupID

	f.storeRequest(req, targetsStatus)
	wg := sync.WaitGroup{}
	for agent, connected := range targetsStatus {
		if !connected {
			slog.Debug("targeted agent disconnected", "agent", agent)
			lock.Lock()
			results[agent] = &proto.TaskResponse{
				GroupID:       req.GroupID,
				InternalError: proto.InternalError_DISCONNECTED,
			}
			lock.Unlock()
			continue
		}

		wg.Go(func() {
			resp := make(chan *proto.TaskResponse)
			task := Task[*proto.TaskRequest, *proto.TaskResponse]{
				Request:    req,
				ResponseCh: resp,
			}
			timeout := time.Duration(req.GetTimeout()) * time.Second
			if err := f.taskDispatcher.Send(agt.ID(agent), task, timeout); err != nil {
				internalError := proto.InternalError_UNKNOWN_ERROR
				switch {
				case errors.Is(err, ErrAgentNotFound):
					internalError = proto.InternalError_DISCONNECTING
				case errors.Is(err, ErrClosedTaskChannel):
					internalError = proto.InternalError_DISCONNECTING
				case errors.Is(err, ErrTimeout):
					internalError = proto.InternalError_TIMEOUT
				}

				lock.Lock()
				defer lock.Unlock()
				results[agent] = &proto.TaskResponse{
					GroupID:       req.GroupID,
					InternalError: internalError,
				}
				return
			}

			select {
			case r := <-resp:
				lock.Lock()
				defer lock.Unlock()
				results[agent] = r
			case <-time.After(timeout):
				lock.Lock()
				defer lock.Unlock()
				results[agent] = &proto.TaskResponse{
					GroupID:       req.GroupID,
					InternalError: proto.InternalError_TIMEOUT,
				}
			}
		})
	}
	wg.Wait()

	return &proto.FwdResponse{Responses: results}, nil
}
