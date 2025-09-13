package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/jackadi-io/jackadi/internal/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Server) loadAgentPlugins() (map[string][]string, error) {
	s.pluginList.lock.Lock()
	defer s.pluginList.lock.Unlock()
	if time.Since(s.pluginList.lastUpdate) < 10*time.Second {
		slog.Debug("get plugin list from cache")
		return s.pluginList.cache, nil
	}

	configFile := path.Join(s.config.ConfigDir, "plugins.yaml")
	slog.Debug("get plugin list from file", "file", configFile)
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin list file: %w", err)
	}

	list := map[string][]string{}
	if err := yaml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("invalid plugin list file: %w", err)
	}

	s.pluginList.cache = list
	s.pluginList.lastUpdate = time.Now()

	return list, nil
}

func (s *Server) ListAgentPlugins(ctx context.Context, req *emptypb.Empty) (*proto.ListAgentPluginsResponse, error) {
	resp := &proto.ListAgentPluginsResponse{}
	agentInfo, err := signatureFromContext(ctx, s.config.MTLSEnabled)
	if err != nil {
		return resp, status.Error(codes.InvalidArgument, err.Error())
	}

	agentPlugins, err := s.loadAgentPlugins()
	if err != nil {
		return resp, status.Error(codes.NotFound, fmt.Sprintf("failed to load plugin list: %s", err))
	}

	list := []string{}
	for pattern, plugins := range agentPlugins {
		matched, err := filepath.Match(pattern, string(agentInfo.ID))
		if err != nil {
			return nil, fmt.Errorf("invalid pattern '%s': %w ", pattern, err)
		}
		if !matched {
			continue
		}
		for _, p := range plugins {
			if !slices.Contains(list, p) {
				list = append(list, p)
			}
		}
	}
	resp.Plugin = list
	return resp, nil
}
