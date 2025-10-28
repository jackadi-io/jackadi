package agent

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/plugin/inventory"
	"github.com/jackadi-io/jackadi/internal/plugin/loader/hcplugin"
	"github.com/jackadi-io/jackadi/internal/proto"
	"github.com/jackadi-io/jackadi/internal/serializer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
	"google.golang.org/grpc/status"
)

// Config holds the configuration for creating a new Agent.
type Config struct {
	ManagerAddress   string
	ManagerPort      string
	AgentID          string
	MTLSEnabled      bool
	MTLSCert         string
	MTLSKey          string
	MTLSManagerCA    string
	PluginDir        string
	PluginServerPort string
	CustomResolvers  []string
}

type Agent struct {
	config               Config
	taskClient           proto.ClusterClient
	conn                 *grpc.ClientConn
	pluginLoader         hcplugin.Loader
	connectedManagerAddr string
	SpecManager          *SpecsManager
}

// New returns a new Client and an initialized context containing values like agent_id.
//
// While the agent is the client from GRPC perspective, it is the server from application perspective,
// i.e. the manager (GRPC server) send request to the agent (GRPC client) via a bidi GRPC stream.
func New(cfg Config) (Agent, context.Context, error) {
	md := metadata.Pairs("agent_id", cfg.AgentID)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	specsManager, err := NewSpecsManager()
	if err != nil {
		return Agent{}, nil, fmt.Errorf("failed to load SpecsManager: %w", err)
	}

	c := Agent{
		config:      cfg,
		SpecManager: specsManager,
	}
	return c, ctx, nil
}

func (a *Agent) Connect(ctx context.Context) error {
	var err error
	slog.Info("connecting to the manager", "address", a.config.ManagerAddress, "port", a.config.ManagerPort)
	managerHost := net.JoinHostPort(a.config.ManagerAddress, a.config.ManagerPort)

	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.ClientKeepaliveTime,
			Timeout:             config.ClientKeepaliveTimeout,
			PermitWithoutStream: true,
		}),
	}

	if len(a.config.CustomResolvers) > 0 {
		r := manual.NewBuilderWithScheme("jack")
		var addresses []resolver.Address
		for _, addr := range a.config.CustomResolvers {
			addresses = append(addresses, resolver.Address{Addr: addr})
		}
		r.InitialState(resolver.State{
			Addresses: addresses,
		})
		managerHost = fmt.Sprintf("%s:///%s", r.Scheme(), a.config.ManagerAddress)
		opts = append(opts, grpc.WithResolvers(r))
	}

	if a.config.MTLSEnabled {
		certs, ca, err := config.GetMTLSCertificate(a.config.MTLSCert, a.config.MTLSKey, a.config.MTLSManagerCA)
		if err != nil {
			return err
		}
		tlsCfg := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			ServerName:   a.config.ManagerAddress,
			Certificates: certs,
			RootCAs:      ca,
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		slog.Warn("unsecured connection with the manager, you should enable mTLS")
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	a.conn, err = grpc.NewClient(managerHost, opts...)
	if err != nil {
		return err
	}
	a.taskClient = proto.NewClusterClient(a.conn)

	return nil
}

func (a *Agent) Close() error {
	if a.conn == nil {
		return errors.New("trying to close a nil connection")
	}
	return a.conn.Close()
}

func (a *Agent) Handshake(ctx context.Context) error {
	res, err := a.taskClient.Handshake(ctx, &proto.HandshakeRequest{Id: 1})
	if err != nil {
		return fmt.Errorf("handshake failed: %w", err)
	}
	slog.Debug("received response to ping", "got", res)
	return nil
}

func (a *Agent) ListenTaskRequest(ctx context.Context) error {
	runningTasks := make(chan struct{}, config.MaxConcurrentTasks)
	runningWriteTask := make(chan struct{}, 1) // Only one write task at a time
	requestsQueue := make(chan struct{}, config.MaxWaitingRequests)
	exclusiveLock := sync.RWMutex{}

	stream, err := a.taskClient.ExecTask(ctx)
	if err != nil {
		return fmt.Errorf("client failed: %w", err)
	}

	a.updateKnownManagerAddress(stream)

	defer slog.Debug("exiting task handler")

	wg := sync.WaitGroup{}
	for {
		slog.Debug("waiting for new requests")
		req, err := stream.Recv()
		if err != nil || req == nil {
			slog.Debug("stream error", "error", err, "component", "task listener")
			wg.Wait()
			if errors.Is(err, io.EOF) {
				slog.Debug("stream closed", "component", "task listener")
				return nil
			}
			slog.Debug("stream failure", "component", "task listener")
			return fmt.Errorf("client failed: %w", err)
		}

		// answers immediately to instant healthcheck
		if req.Task == fmt.Sprintf("health:%s", config.InstantPingName) {
			out, _ := serializer.JSON.Marshal(true)
			resp := proto.TaskResponse{
				Id:      req.GetId(),
				GroupID: req.GroupID,
				Output:  out,
			}
			if err := stream.Send(&resp); err != nil {
				slog.Error("failed to send back health:ping")
			}
			continue
		}

		// Resolve the effective lock mode - use CLI override or plugin default
		lockMode := effectiveLockMode(req)

		var taskSlot chan struct{}
		if lockMode == proto.LockMode_NO_LOCK {
			taskSlot = runningTasks
		} else {
			taskSlot = runningWriteTask
		}

		// TODO: implement FIFO queue. Be careful, we will still want to send the timeout response as soon as possible,
		// But we need to ensure the FIFO queue will discard it to and avoid channel deadlock.

		// trying to reserve a spot in the queue
		select {
		case requestsQueue <- struct{}{}:
		default:
			resp := proto.TaskResponse{
				Id:            req.GetId(),
				GroupID:       req.GroupID,
				InternalError: proto.InternalError_FULL_QUEUE,
			}
			if err := stream.Send(&resp); err != nil {
				slog.Error("failed to send back BUSY_QUEUE")
			}
			slog.Error("too many waiting requests", "error", "BUSY_QUEUE")
			continue
		}

		// executes the task as soon as possible
		wg.Add(1)
		go func() {
			defer func() {
				<-requestsQueue
				wg.Done()
			}()

			slog.Debug("exec request received", "id", req.Id, "group", req.GetGroupID(), "task", req.Task, "args", req.Input)
			timeout := uint32(config.TaskTimeout.Seconds())
			if val := req.GetTimeout(); val > 0 {
				timeout = val
			}
			slog.Debug("timeout", "value_set", time.Duration(timeout)*time.Second)

			t := time.NewTimer(time.Duration(timeout) * time.Second)

			var resp *proto.TaskResponse
			select {
			case taskSlot <- struct{}{}:
				defer func() { <-taskSlot }()

				// some task must be the only one to run, like plugin sync
				if lockMode == proto.LockMode_EXCLUSIVE {
					slog.Debug("lock")
					exclusiveLock.Lock()
					defer exclusiveLock.Unlock()
					defer slog.Debug("unlock")
				} else {
					slog.Debug("rlock")
					exclusiveLock.RLock()
					defer exclusiveLock.RUnlock()
					defer slog.Debug("read unlock")
				}

				finished := make(chan struct{}, 1)
				go func() {
					// send the IDs back to the client for task not finished in time.
					// even if the task finishes after the timeout, the result will still be sent.
					select {
					case <-finished:
						slog.Debug("task done", "id", req.Id)
					case <-t.C:
						slog.Debug("started task timeout", "id", req.Id)
						respErrTimeout := &proto.TaskResponse{
							Id:            req.GetId(),
							GroupID:       req.GroupID,
							InternalError: proto.InternalError_STARTED_TIMEOUT,
						}
						if err := stream.Send(respErrTimeout); err != nil {
							slog.Error("failed to send response", "err", err)
						}
					}
				}()

				// We do not use the context of stream, because we don't want to cancel a maintenance
				// in case of temporary disconnection.
				resp = doTask(ctx, req)
				if !t.Stop() {
					// Timer already fired, timeout response already sent
					return
				}
				finished <- struct{}{}

			case <-t.C:
				slog.Debug("task not executed: waiting timeout reached", "id", req.Id)
				resp = &proto.TaskResponse{
					Id:            req.GetId(),
					GroupID:       req.GroupID,
					InternalError: proto.InternalError_TIMEOUT,
				}

			case <-ctx.Done():
				slog.Debug("task context closed")
			}

			slog.Debug("sending response", "id", req.Id)
			if err = stream.Send(resp); err != nil {
				slog.Error("failed to send response", "err", err)
			}
			slog.Debug("response sent", "id", req.Id)
		}()
	}
}

// updateKnownManagerAddress updates stored resolved manager address (useful for plugin sync for instance).
func (a *Agent) updateKnownManagerAddress(stream grpc.BidiStreamingClient[proto.TaskResponse, proto.TaskRequest]) {
	p, ok := peer.FromContext(stream.Context())
	if ok {
		socket, err := netip.ParseAddrPort(p.Addr.String())
		if err != nil || !socket.IsValid() {
			slog.Error("failed to resolve manager address", "addr", p.Addr.String(), "error", err)
			return
		}
		a.connectedManagerAddr = socket.Addr().String()
		slog.Info("manager address resolved", "addr", socket.Addr().String())
	}
}

// effectiveLockMode determines the lock mode to use for a task.
//
// Precedence order: override from request, default mode set at task level.
func effectiveLockMode(req *proto.TaskRequest) proto.LockMode {
	// request override
	if req.GetLockMode() != proto.LockMode_UNSPECIFIED {
		return req.GetLockMode()
	}

	// get lock mode from the task itself
	var plugin, task string
	parts := strings.Split(req.GetTask(), config.PluginSeparator)

	switch {
	case len(parts) == 1:
		plugin = parts[0]
		task = parts[0]
	case len(parts) == 2:
		plugin = parts[0]
		task = parts[1]
	default:
		slog.Debug("bad task name, using NO_LOCK", "task", req.GetTask())
		return proto.LockMode_NO_LOCK
	}

	coll, err := inventory.Registry.Get(plugin)
	if err != nil {
		slog.Debug("plugin not found, using NO_LOCK", "plugin", plugin, "error", err)
		return proto.LockMode_NO_LOCK
	}

	pluginLockMode, err := coll.GetTaskLockMode(task)
	if err != nil {
		slog.Debug("failed to get plugin lock mode, using NO_LOCK", "task", task, "error", err)
		return proto.LockMode_NO_LOCK
	}

	slog.Debug("using plugin default lock mode", "task", req.GetTask(), "lockMode", pluginLockMode.String())
	return pluginLockMode
}

// doTask route the request to the plugin containing the wanted task.
func doTask(ctx context.Context, req *proto.TaskRequest) *proto.TaskResponse {
	slog.Debug("starting task", "id", req.Id)
	var plugin, task string
	parts := strings.Split(req.GetTask(), config.PluginSeparator)

	switch {
	case len(parts) == 1:
		plugin = parts[0]
		task = parts[0]
	case len(parts) == 2:
		plugin = parts[0]
		task = parts[1]
	default:
		slog.Error("bad task name")
		return &proto.TaskResponse{
			Id:            req.GetId(),
			GroupID:       req.GroupID,
			InternalError: proto.InternalError_UNKNOWN_TASK,
		}
	}

	t, err := inventory.Registry.Get(plugin)
	if err != nil {
		slog.Error("bad request", "error", err)
		return &proto.TaskResponse{
			Id:            req.GetId(),
			GroupID:       req.GroupID,
			InternalError: proto.InternalError_UNKNOWN_TASK,
		}
	}

	response, err := t.Do(ctx, task, req.GetInput())

	r := proto.TaskResponse{
		Id:      req.GetId(),
		GroupID: req.GroupID,
		Output:  response.Output,
		Error:   response.Error,
		Retcode: response.Retcode,
	}
	if err != nil {
		r.InternalError = proto.InternalError_MODULE_ERROR
		e := status.Convert(err)
		if e.Code() == codes.Unknown {
			r.ModuleError = e.Message()
		} else {
			r.ModuleError = fmt.Sprintf("code=%s, error=%s", e.Code(), e.Message())
		}
	}

	slog.Debug("task finished", "id", req.Id)

	return &r
}
