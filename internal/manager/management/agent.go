package management

import (
	"context"
	"errors"
	"log/slog"
	"slices"

	"github.com/dgraph-io/badger/v4"
	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/manager/inventory"
	"github.com/jackadi-io/jackadi/internal/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ServerInterface defines the interface that the management API needs from the server.
// The goal is to ease mocking.
type ServerInterface interface {
	RequestShutdown(agentID agent.ID) error
	GetInventory() *inventory.Agents
}

type apiServer struct {
	proto.UnimplementedAPIServer
	server ServerInterface
	db     *badger.DB
}

func New(server ServerInterface, db *badger.DB) apiServer {
	return apiServer{
		server: server,
		db:     db,
	}
}

func toProtoAgentSlice(agents []inventory.AgentIdentity, agentsState map[agent.ID]inventory.AgentState) []*proto.AgentInfo {
	resp := make([]*proto.AgentInfo, 0, len(agents))
	for _, agent := range agents {
		addr, cert := agent.Address, agent.Certificate
		info := proto.AgentInfo{
			Id:          string(agent.ID),
			Address:     &addr,
			Certificate: &cert,
		}

		if state, ok := agentsState[agent.ID]; ok {
			info.IsConnected = &state.Connected
			info.Since = timestamppb.New(state.Since)
			info.LastMsg = timestamppb.New(state.LastMsg)
		}

		resp = append(resp, &info)
	}
	return resp
}

// ListAgents returns the agents the manager is seeing.
//
//   - accepted: agents managed by the manager.
//   - candidates: agents not yet managed by the manager, waiting to be accepted.
//   - rejected: agents which have been explicitly excluded.
func (a *apiServer) ListAgents(ctx context.Context, req *proto.ListAgentsRequest) (*proto.ListAgentsResponse, error) {
	accepted, candidates, rejected, states := a.server.GetInventory().List()
	return &proto.ListAgentsResponse{
		Accepted:   toProtoAgentSlice(accepted, states),
		Candidates: toProtoAgentSlice(candidates, states),
		Rejected:   toProtoAgentSlice(rejected, states),
	}, nil
}

func (a *apiServer) AcceptAgent(ctx context.Context, req *proto.AgentRequest) (*proto.AgentResponse, error) {
	candidates := a.server.GetInventory().GetMatchingCandidates(agent.ID(req.Agent.GetId()), req.Agent.Address, req.Agent.Certificate)
	rejected := a.server.GetInventory().GetMatchingRejected(agent.ID(req.Agent.GetId()), req.Agent.Address, req.Agent.Certificate)

	agents := slices.Concat(candidates, rejected)
	agent := inventory.AgentIdentity{}
	switch len(agents) {
	case 0:
		return nil, status.Error(codes.NotFound, "agent not in candidate/rejected list")
	case 1:
		agent = agents[0]
	default:
		return nil, status.Error(codes.NotFound, "too many agent matching")
	}

	err := a.server.GetInventory().Register(agent, true)
	if err != nil {
		slog.Debug("agent not registered", "error", err)
	}

	return &proto.AgentResponse{Agent: &proto.AgentInfo{
		Id:          string(agent.ID),
		Address:     &agent.Address,
		Certificate: &agent.Certificate,
	}}, err
}

// RemoveAgent completely removes an agent from all lists: accepted, candidates and rejected.
func (a *apiServer) RemoveAgent(ctx context.Context, req *proto.AgentRequest) (*proto.AgentsResponse, error) {
	accepted := a.server.GetInventory().GetMatchingAccepted(agent.ID(req.Agent.GetId()), req.Agent.Address, req.Agent.Certificate)
	candidates := a.server.GetInventory().GetMatchingCandidates(agent.ID(req.Agent.GetId()), req.Agent.Address, req.Agent.Certificate)
	rejected := a.server.GetInventory().GetMatchingRejected(agent.ID(req.Agent.GetId()), req.Agent.Address, req.Agent.Certificate)

	agents := slices.Concat(rejected, candidates, accepted)
	if len(agents) == 0 {
		return nil, status.Error(codes.NotFound, inventory.ErrAgentNotFound.Error())
	}

	var errs error
	removed := []inventory.AgentIdentity{}
	for _, agent := range agents {
		if err := a.server.RequestShutdown(agent.ID); err != nil {
			slog.Info("cannot shutdown connection with agent", "error", err)
		}

		if err := a.server.GetInventory().Remove(agent); err != nil {
			slog.Info("cannot unregister", "error", err)
			if joinErr := errors.Join(errs, err); joinErr != nil {
				return nil, err
			}
		}
	}

	return &proto.AgentsResponse{
		Agents: toProtoAgentSlice(removed, nil),
	}, errs
}

// RejectAgent place an agent in the rejected list.
//
// If the agent is registered, it disconnects the agent.
// A rejected agent cannot register anymore unless explicitly accepted.
// It rejects only the fully matching agent.
func (a *apiServer) RejectAgent(ctx context.Context, req *proto.AgentRequest) (*proto.AgentsResponse, error) {
	candidates := a.server.GetInventory().GetMatchingCandidates(agent.ID(req.Agent.GetId()), req.Agent.Address, req.Agent.Certificate)
	accepted := a.server.GetInventory().GetMatchingAccepted(agent.ID(req.Agent.GetId()), req.Agent.Address, req.Agent.Certificate)

	agents := slices.Concat(candidates, accepted)
	if len(agents) == 0 {
		return nil, status.Error(codes.NotFound, inventory.ErrAgentNotFound.Error())
	}

	var errs error
	rejected := []inventory.AgentIdentity{}
	for _, agent := range agents {
		if a.server.GetInventory().IsRegistered(agent) {
			err := a.server.RequestShutdown(agent.ID)
			slog.Info("cannot reject because cannot shutdown connection with agent", "error", err)
		}
		if err := a.server.GetInventory().Reject(agent); err != nil {
			if joinErr := errors.Join(errs, err); joinErr != nil {
				return nil, err
			}
		}
		rejected = append(rejected, agent)
	}
	return &proto.AgentsResponse{
		Agents: toProtoAgentSlice(rejected, nil),
	}, errs
}
