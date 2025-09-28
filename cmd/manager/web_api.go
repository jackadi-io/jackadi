package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/proto"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	protobuf "google.golang.org/protobuf/proto"
)

type proxyResponse struct {
	Id            int64
	GroupID       *int64
	Output        json.RawMessage
	Error         string
	Retcode       int32
	InternalError string
	ModuleError   string
}

type Htpasswd struct {
	creds map[string]string
}

func NewHtpasswd() Htpasswd {
	return Htpasswd{creds: make(map[string]string)}
}

func (h *Htpasswd) Get(user string) (string, error) {
	password, ok := h.creds[user]
	if !ok {
		return "", errors.New("unknown user")
	}
	if password == "" {
		return "", errors.New("empty password")
	}

	return password, nil
}

func (h *Htpasswd) load(file string) error {
	fd, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("htpasswd not loaded: %w", err)
	}
	defer func() {
		_ = fd.Close()
	}()

	creds := make(map[string]string)
	sc := bufio.NewScanner(fd)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		creds[parts[0]] = parts[1]
	}
	h.creds = creds
	if len(h.creds) == 0 {
		return errors.New("no credentials in htpasswd file")
	}
	return nil
}

func (h *Htpasswd) basicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		expectedHash, err := h.Get(username)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"Unauthorized","message":"Authentication required","status":401}`))
			return
		}

		if !ok || bcrypt.CompareHashAndPassword([]byte(expectedHash), []byte(password)) != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"Unauthorized","message":"Authentication required","status":401}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func responseEnvelope(_ context.Context, response protobuf.Message) (any, error) {
	if out, ok := response.(*proto.FwdResponse); ok {
		decodedResponses := make(map[string]*proxyResponse, len(out.GetResponses()))

		for agentName, response := range out.GetResponses() {
			decodedResponse := proxyResponse{
				Id:            response.GetId(),
				GroupID:       response.GroupID,
				Output:        response.GetOutput(),
				Error:         response.GetError(),
				InternalError: response.GetInternalError().String(),
				ModuleError:   response.GetModuleError(),
			}
			decodedResponses[agentName] = &decodedResponse
		}
		return decodedResponses, nil
	}

	return response, nil
}

func startHTTPProxy(ctx context.Context, cfg managerConfig) error {
	cancelableCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithForwardResponseRewriter(responseEnvelope),
	)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	endpoint := fmt.Sprintf("unix:///%s", config.CLISocket)

	if err := proto.RegisterForwarderHandlerFromEndpoint(cancelableCtx, mux, endpoint, opts); err != nil {
		return err
	}

	if err := proto.RegisterAPIHandlerFromEndpoint(cancelableCtx, mux, endpoint, opts); err != nil {
		return err
	}
	// Wrap the mux with basic auth middleware
	slog.Info("loading htpasswd")
	htpasswd := NewHtpasswd()
	err := htpasswd.load(filepath.Join(cfg.configDir, config.HTPasswordFile))
	if err != nil {
		slog.Warn("htpasswd not loaded", "error", err)
	}
	authHandler := htpasswd.basicAuthMiddleware(mux)

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	slog.Info("starting Web API")
	httpServer := http.Server{
		Addr:              ":8081", // TOOD: make it configurable
		Handler:           authHandler,
		ReadHeaderTimeout: config.HTTPReadHeaderTimeout,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Warn("web api failed to stop properly", "error", err)
		}
	}()
	return httpServer.ListenAndServe() // TODO: TLS
}
