package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/helper"
	"github.com/jackadi-io/jackadi/internal/manager/database"
	"github.com/jackadi-io/jackadi/internal/manager/forwarder"
	"github.com/jackadi-io/jackadi/internal/manager/inventory"
	"github.com/jackadi-io/jackadi/internal/proto"
	"github.com/jackadi-io/jackadi/internal/serializer"
	"google.golang.org/protobuf/types/known/structpb"
)

type ServerConfig struct {
	AutoAccept  bool
	MTLSEnabled bool
	ConfigDir   string
}

type Server struct {
	proto.UnimplementedCommServer
	config          ServerConfig
	Inventory       *inventory.Agents
	taskDispatcher  forwarder.Dispatcher[*proto.TaskRequest, *proto.TaskResponse]
	db              *badger.DB
	dbMutex         *sync.Mutex
	shutdownRequest map[agent.ID]chan struct{}
	pluginList      pluginList
}

type pluginList struct {
	cache      map[string][]string
	lastUpdate time.Time
	lock       *sync.Mutex
}

func New(config ServerConfig, agentsInventory *inventory.Agents, taskDispatcher forwarder.Dispatcher[*proto.TaskRequest, *proto.TaskResponse], jobDatabase *badger.DB) Server {
	return Server{
		config:          config,
		Inventory:       agentsInventory,
		taskDispatcher:  taskDispatcher,
		db:              jobDatabase,
		dbMutex:         &sync.Mutex{},
		shutdownRequest: make(map[agent.ID]chan struct{}),
		pluginList:      pluginList{lock: &sync.Mutex{}},
	}
}

func (s *Server) RequestShutdown(agentID agent.ID) error {
	ch, ok := s.shutdownRequest[agentID]
	if !ok {
		return errors.New("shutdownRequest channel not found")
	}
	if ch == nil {
		return errors.New("shutdownRequest channel not initialized")
	}

	close(s.shutdownRequest[agentID])
	s.shutdownRequest[agentID] = nil
	if err := s.taskDispatcher.Forget(agentID); err != nil {
		slog.Error("fail to forget an agent", "error", err)
	}
	return nil
}

// GetInventory returns the server's inventory.
func (s *Server) GetInventory() *inventory.Agents {
	return s.Inventory
}

// storeResult records task responses in a local KV store. The KV store is an embedded Badger instance.
//
// It stores the result itself by task ID. It also stores a mapping between a group ID and task IDs.
// Group ID are grouping tasks response from a same request, i.e. when the request was targeting multiple agents.
func (s *Server) storeResult(agentID agent.ID, msg *proto.TaskResponse) {
	s.dbMutex.Lock()
	defer s.dbMutex.Unlock()

	dbDerr := s.db.Update(func(txn *badger.Txn) error {
		data, err := database.MarshalTask(agentID, msg)
		if err != nil {
			slog.Error("unable to record result", "error", "marshal error")
			return err
		}

		id := strconv.FormatInt(msg.GetId(), 10)
		if msg.GetId() == 0 {
			return nil
		}

		key := database.GenerateResultKey(id)
		singleEntry := badger.NewEntry(key, data).WithTTL(config.DBTaskResultTTL)
		if err := txn.SetEntry(singleEntry); err != nil {
			slog.Error("unable to record result", "error", err)
			return err
		}

		// map the result to the matching groupID
		if msg.GetGroupID() == 0 {
			slog.Debug("no need to group the result", "msg", "no group ID defined")
			return nil
		}
		groupID := strconv.FormatInt(msg.GetGroupID(), 10)
		groupKey := database.GenerateResultKey(groupID)
		item, err := txn.Get(groupKey)
		if err != nil {
			if !errors.Is(err, badger.ErrKeyNotFound) {
				slog.Error("unable to group the result", "error", "marshal error")
				return err
			}
			groupEntry := badger.NewEntry(groupKey, []byte("grouped:"+id)).WithTTL(config.DBTaskResultTTL)
			if err := txn.SetEntry(groupEntry); err != nil {
				slog.Error("unable to record the new group", "error", err)
				return err
			}
			return nil
		}
		groupEntry, err := item.ValueCopy(nil)
		if err != nil {
			slog.Error("unable to get value of existing group", "error", err)
			return err
		}

		groupEntry = append(groupEntry, []byte(","+id)...)
		if err := txn.Set(groupKey, groupEntry); err != nil {
			slog.Error("unable to get update the existing group", "error", err)
			return err
		}

		slog.Debug("updating group", "value", string(groupEntry))
		return nil
	})
	if dbDerr != nil {
		slog.Warn("failed to store task result", "error", dbDerr)
	}
}

// dispatchRequestsToAgent waits for requests and send them to the linked agent.
func (s *Server) dispatchRequestsToAgent(agentID agent.ID, stream proto.Comm_ExecTaskServer, responsesCh map[int64]chan *proto.TaskResponse, responsesChLock *sync.Mutex) error {
	tasksCh, err := s.taskDispatcher.GetTasksChannel(agentID)
	if err != nil {
		return err
	}

	for d := range tasksCh {
		ID := time.Now().UnixNano()
		err := stream.Send(
			&proto.TaskRequest{
				Id:      ID,
				GroupID: d.Request.GroupID,
				Task:    d.Request.Task,
				Input:   d.Request.GetInput(),
				Timeout: d.Request.Timeout,
			},
		)
		if err != nil {
			slog.Error("failed to send task", "err", err, "agent", agentID)
			return err
		}
		responsesChLock.Lock()
		responsesCh[ID] = d.ResponseCh
		responsesChLock.Unlock()
		d := d
		go func() {
			// ensure cleaning to avoid memory leak when responses never received
			time.Sleep(time.Duration(d.Request.Timeout) * time.Second)
			responsesChLock.Lock()
			delete(responsesCh, ID)
			responsesChLock.Unlock()
		}()
	}
	return nil
}

// dispatchAgentResponse waits for agent's response and send it back to the requester.
//
// It stores all received responses to job database.
func (s *Server) dispatchAgentResponse(stream proto.Comm_ExecTaskServer, agentID agent.ID, responsesCh map[int64]chan *proto.TaskResponse, responsesChLock *sync.Mutex) error {
	ctx := stream.Context()

	for {
		msg, err := stream.Recv()
		select {
		case <-ctx.Done():
			slog.Debug("stream context cancelled", "agent", agentID, "error", ctx.Err())
			return ctx.Err()
		case <-s.shutdownRequest[agentID]:
			slog.Warn("shutdown request received", "agent", agentID)
			slog.Debug("ignored response because received after shutdown request", "agent", agentID)
			return errors.New("shutdown requested")
		default:
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			slog.Error("connection with agent stopped", "agent", agentID, "error", err)
			return err
		}

		slog.Debug("received task response", "id", msg.GetId(), "agent", agentID, "group", msg.GetGroupID())
		if msg.GetInternalError() != proto.InternalError_STARTED_TIMEOUT {
			// we don't store the message if the task has started to avoid duplicate entries if the task finishes after the timeout
			s.storeResult(agentID, msg)
		}

		if msg.GetInternalError() == proto.InternalError_OK {
			s.Inventory.MarkAgentActive(agentID)
		}

		responsesChLock.Lock()
		ch, ok := responsesCh[msg.GetId()]
		responsesChLock.Unlock()
		if ok {
			select {
			case ch <- msg:
			case <-time.After(config.ResponseChannelTimeout):
			}
			responsesChLock.Lock()
			delete(responsesCh, msg.GetId())
			responsesChLock.Unlock()
		} else {
			slog.Info("response not sent to the caller", "err", "response channel not found", "agent", agentID, "id", msg.GetId())
		}

		if msg.GetInternalError() > 0 {
			slog.Error("the agent rejected the request", "error", msg.GetInternalError(), "message", msg.GetModuleError(), "request_id", msg.GetId(), "agent", agentID)
		}
	}
}

// ExecTask spawn a stream with each agents.
//
// It handles the sending of requests and the routing of the response.
// The responses is routed to the dispatcher (usually the forwarder).
func (s *Server) ExecTask(stream proto.Comm_ExecTaskServer) error {
	agent, err := signatureFromContext(stream.Context(), s.config.MTLSEnabled)
	if err != nil {
		return err
	}
	slog.Debug("new 'task' stream with agent", "agent", agent.ID)
	s.shutdownRequest[agent.ID] = make(chan struct{})
	defer func() {
		delete(s.shutdownRequest, agent.ID)
	}()

	for !s.Inventory.IsRegistered(agent) {
		slog.Debug("cannot start 'Task' stream", "error", "agent not registered", "retry_in", fmt.Sprintf("%d", int(config.AgentRetryDelay.Seconds())))
		time.Sleep(config.AgentRetryDelay)
	}

	if err := s.taskDispatcher.RegisterAgent(agent.ID); err != nil {
		slog.Warn("closing stream", "msg", err, "agent", agent.ID, "peer", agent.Address)
		return err
	}
	s.Inventory.MarkAgentStateChange(agent.ID, true)

	defer func() {
		slog.Debug("deleting agent dispatcher", "agent", agent.ID)
		s.taskDispatcher.UnregisterAgent(agent.ID)
		s.Inventory.MarkAgentStateChange(agent.ID, false)
	}()

	responsesCh := make(map[int64]chan *proto.TaskResponse)
	lock := &sync.Mutex{} // TODO: replace map[]chan by a single thread channel manager
	errCh := make(chan error)

	go func() {
		err := s.dispatchAgentResponse(stream, agent.ID, responsesCh, lock)
		slog.Debug("closing agent dispatcher", "agent", agent.ID)
		s.taskDispatcher.Close(agent.ID)
		slog.Debug("agent dispatcher closed", "agent", agent.ID)
		errCh <- err
	}()

	err = s.dispatchRequestsToAgent(agent.ID, stream, responsesCh, lock)

	return errors.Join(err, <-errCh)
}

func (s *Server) CollectAgentsSpecs(ctx context.Context) {
	timeout := config.DefaultTaskTimeout
	tick := time.NewTicker(config.SpecCollectionInterval)
	for {
		agents, _ := s.taskDispatcher.TargetedAgents("*", proto.TargetMode_GLOB)

		wg := sync.WaitGroup{}
		slog.Debug("starting agents specs collector")
		for agt, connected := range agents {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if !connected {
				slog.Debug("cannot get spec of agent", "agent", agt, "error", "disconnected")
				continue
			}

			wg.Add(1)
			go func() {
				slog.Debug("collecting specs", "agent", agt)
				defer wg.Done()

				req := &proto.TaskRequest{
					Task:    "specs:all",
					Input:   &proto.Input{Args: &structpb.ListValue{}},
					Timeout: helper.DurationToUint32(timeout),
				}

				resp := make(chan *proto.TaskResponse)
				task := forwarder.Task[*proto.TaskRequest, *proto.TaskResponse]{
					Request:    req,
					ResponseCh: resp,
				}
				// timeout+time.Second to get last moment response
				if err := s.taskDispatcher.Send(agent.ID(agt), task, timeout+time.Second); err != nil {
					slog.Warn("failed to send collect specs request", "agent", agt, "error", err)
					close(resp)
					return
				}

				res := <-resp
				if res.GetError() != "" {
					slog.Warn("failed to collect specs", "agent", agt, "error", res.GetError())
					return
				}

				specs := make(map[string]any)
				if err := serializer.JSON.Unmarshal(res.GetOutput(), &specs); err != nil {
					slog.Warn("failed to parse specs", "agent", agt, "error", err)
					return
				}

				if err := s.Inventory.SetSpec(agent.ID(agt), specs); err != nil {
					slog.Warn("failed to set specs", "agent", agt, "error", err)
					return
				}

				slog.Debug("specs collected", "agent", agt)
			}()
		}
		wg.Wait()

		select {
		case <-tick.C:
		case <-ctx.Done():
			return
		}
	}
}
