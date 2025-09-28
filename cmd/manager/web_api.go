package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

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

func (h *Htpasswd) load(file string) (map[string]string, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("htpasswd not loaded: %w", err)
	}

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

	return creds, nil
}

func (h *Htpasswd) basicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		expectedHash, err := h.Get(username)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("401 Unauthorized\n"))
			return
		}

		if !ok || password != bcrypt.CompareHashAndPassword(expectedHash, []byte(password)) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("401 Unauthorized\n"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func responseEnvelope(_ context.Context, response protobuf.Message) (any, error) {
	switch out := response.(type) {
	case *proto.FwdResponse:
		decodedResponses := make(map[string]*proxyResponse)

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

func startHTTPProxy() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithForwardResponseRewriter(responseEnvelope),
	)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	endpoint := fmt.Sprintf("unix:///%s", config.CLISocket)

	if err := proto.RegisterAPIHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
		return err
	}
	if err := proto.RegisterForwarderHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
		return err
	}

	// Wrap the mux with basic auth middleware
	htpasswd := NewHtpasswd()
	htpasswd.load(".htpasswd") // TODO: right location
	authHandler := htpasswd.basicAuthMiddleware(mux)

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return http.ListenAndServe(":8081", authHandler)
}
