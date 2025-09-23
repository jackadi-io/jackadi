package core

import (
	"context"

	goplugin "github.com/hashicorp/go-plugin"
	"github.com/jackadi-io/jackadi/internal/plugin/core/protoplugin"
	"google.golang.org/grpc"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = goplugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   "JACKADI_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "hv43fBvWLYYm0FWmur3V5KmJQXPR0woBg2F2MbyAcE8UuSCQGTlFJvTQUncCY0Xm",
}

type CollectionPlugin struct {
	goplugin.Plugin
	Impl Collection
}

func (p *CollectionPlugin) GRPCServer(broker *goplugin.GRPCBroker, s *grpc.Server) error {
	protoplugin.RegisterPluginCollectionServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (*CollectionPlugin) GRPCClient(ctx context.Context, broker *goplugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &GRPCClient{client: protoplugin.NewPluginCollectionClient(c)}, nil
}
