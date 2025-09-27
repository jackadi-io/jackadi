package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/proto"
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

func basicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || password != "toto" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("401 Unauthorized\n"))
			return
		}
		_ = username // password is checked, username can be anything
		next.ServeHTTP(w, r)
	})
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
	authHandler := basicAuthMiddleware(mux)

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return http.ListenAndServe(":8081", authHandler)
}
