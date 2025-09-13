package server

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/manager/inventory"
	"github.com/jackadi-io/jackadi/internal/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// signatureFromContext extracts metadata from gRPC context.
//
// The main information is the agent ID.
func signatureFromContext(ctx context.Context, mTLSEnabled bool) (inventory.AgentIdentity, error) {
	signature := inventory.AgentIdentity{}

	agentID, err := GetMetadataUniqueKey(ctx, "agent_id")
	signature.ID = agent.ID(agentID)
	if err != nil {
		return signature, errors.New("unspecified agent_id")
	}

	peer, ok := peer.FromContext(ctx)
	if !ok {
		return signature, fmt.Errorf("failed to get agent info")
	}

	switch addr := peer.Addr.(type) {
	case *net.TCPAddr:
		signature.Address = addr.IP.String()
	case *net.UDPAddr:
		signature.Address = addr.IP.String()
	}

	if mTLSEnabled {
		tlsInfo, ok := peer.AuthInfo.(credentials.TLSInfo)
		if !ok {
			return signature, fmt.Errorf("unexpected agent credentials")
		}
		if len(tlsInfo.State.PeerCertificates) < 1 {
			return signature, fmt.Errorf("no agent certificate found")
		}

		data, err := x509.MarshalPKIXPublicKey(tlsInfo.State.PeerCertificates[0].PublicKey)
		if err != nil {
			return signature, fmt.Errorf("unable to marshal public key")
		}
		signature.Certificate = base64.StdEncoding.EncodeToString(data)
	}

	return signature, nil
}

// Handshake handle agent registration.
//
// It checks if the agent changed to detect potential rogue.
func (s *Server) Handshake(ctx context.Context, req *proto.HandshakeRequest) (*proto.HandshakeResponse, error) {
	resp := &proto.HandshakeResponse{Id: req.GetId()}
	agent, err := signatureFromContext(ctx, s.config.MTLSEnabled)
	if err != nil {
		return resp, status.Error(codes.InvalidArgument, err.Error())
	}

	if s.Inventory.IsRegistered(agent) {
		return resp, nil
	}

	err = s.Inventory.AddCandidate(agent)
	if err != nil {
		slog.Debug("agent not added to candidates", "error", err, "agent", agent.ID, "address", agent.Address)
	} else {
		slog.Debug("new agent discovered", "agent", agent.ID, "address", agent.Address)
	}

	if !s.config.AutoAccept {
		return resp, status.Error(codes.PermissionDenied, "agent not registered")
	}

	if err := s.Inventory.Register(agent, false); err != nil {
		slog.Debug("agent not auto-registered", "error", err)
		return resp, status.Error(codes.Unknown, fmt.Sprintf("failed to auto-register agent: %s", err))
	}

	// original code:
	return resp, err
}
