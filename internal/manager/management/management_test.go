package management

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/manager/database"
	"github.com/jackadi-io/jackadi/internal/manager/inventory"
	"github.com/jackadi-io/jackadi/internal/proto"
	"github.com/jackadi-io/jackadi/internal/serializer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockServerInterface is a mock implementation of ServerInterface for testing.
type mockServerInterface struct {
	inventory         *inventory.Agents
	shutdownRequests  []agent.ID
	shutdownError     error
	requestShutdownFn func(agentID agent.ID) error
}

func (m *mockServerInterface) RequestShutdown(agentID agent.ID) error {
	if m.requestShutdownFn != nil {
		return m.requestShutdownFn(agentID)
	}
	m.shutdownRequests = append(m.shutdownRequests, agentID)
	return m.shutdownError
}

func (m *mockServerInterface) GetInventory() *inventory.Agents {
	return m.inventory
}

func newMockServer() *mockServerInterface {
	inv := inventory.New()
	inv.DisableRegistryFile()
	return &mockServerInterface{
		inventory:        &inv,
		shutdownRequests: []agent.ID{},
	}
}

func setupTestDB(t *testing.T) *badger.DB {
	t.Helper()
	opts := badger.DefaultOptions("").WithInMemory(true)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("Failed to open in-memory badger DB: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	return db
}

func TestNew(t *testing.T) {
	mockServer := newMockServer()
	db := setupTestDB(t)

	api := New(mockServer, db)

	if api.server != mockServer {
		t.Error("Expected server to be set")
	}
	if api.db != db {
		t.Error("Expected db to be set")
	}
}

func TestListAgents(t *testing.T) {
	tests := []struct {
		name               string
		setupInventory     func(*inventory.Agents)
		expectedAccepted   int
		expectedCandidates int
		expectedRejected   int
	}{
		{
			name: "empty inventory",
			setupInventory: func(inv *inventory.Agents) {
				// No setup needed
			},
			expectedAccepted:   0,
			expectedCandidates: 0,
			expectedRejected:   0,
		},
		{
			name: "with accepted agents",
			setupInventory: func(inv *inventory.Agents) {
				agent1 := inventory.AgentIdentity{
					ID:          "agent1",
					Address:     "addr1",
					Certificate: "cert1",
				}
				agent2 := inventory.AgentIdentity{
					ID:          "agent2",
					Address:     "addr2",
					Certificate: "cert2",
				}
				_ = inv.AddCandidate(agent1)
				_ = inv.AddCandidate(agent2)
				_ = inv.Register(agent1, true)
				_ = inv.Register(agent2, true)
			},
			expectedAccepted:   2,
			expectedCandidates: 0,
			expectedRejected:   0,
		},
		{
			name: "with candidates",
			setupInventory: func(inv *inventory.Agents) {
				_ = inv.AddCandidate(inventory.AgentIdentity{
					ID:          "candidate1",
					Address:     "addr1",
					Certificate: "cert1",
				})
			},
			expectedAccepted:   0,
			expectedCandidates: 1,
			expectedRejected:   0,
		},
		{
			name: "with rejected agents",
			setupInventory: func(inv *inventory.Agents) {
				_ = inv.Reject(inventory.AgentIdentity{
					ID:          "rejected1",
					Address:     "addr1",
					Certificate: "cert1",
				})
			},
			expectedAccepted:   0,
			expectedCandidates: 0,
			expectedRejected:   1,
		},
		{
			name: "mixed inventory",
			setupInventory: func(inv *inventory.Agents) {
				agent1 := inventory.AgentIdentity{
					ID:          "agent1",
					Address:     "addr1",
					Certificate: "cert1",
				}
				_ = inv.AddCandidate(agent1)
				_ = inv.Register(agent1, true)

				_ = inv.AddCandidate(inventory.AgentIdentity{
					ID:          "candidate1",
					Address:     "addr2",
					Certificate: "cert2",
				})

				_ = inv.Reject(inventory.AgentIdentity{
					ID:          "rejected1",
					Address:     "addr3",
					Certificate: "cert3",
				})
			},
			expectedAccepted:   1,
			expectedCandidates: 1,
			expectedRejected:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := newMockServer()
			db := setupTestDB(t)
			api := New(mockServer, db)

			tt.setupInventory(mockServer.inventory)

			ctx := context.Background()
			req := &proto.ListAgentsRequest{}

			resp, err := api.ListAgents(ctx, req)

			if err != nil {
				t.Fatalf("ListAgents returned error: %v", err)
			}

			if len(resp.Accepted) != tt.expectedAccepted {
				t.Errorf("Expected %d accepted agents, got %d", tt.expectedAccepted, len(resp.Accepted))
			}
			if len(resp.Candidates) != tt.expectedCandidates {
				t.Errorf("Expected %d candidate agents, got %d", tt.expectedCandidates, len(resp.Candidates))
			}
			if len(resp.Rejected) != tt.expectedRejected {
				t.Errorf("Expected %d rejected agents, got %d", tt.expectedRejected, len(resp.Rejected))
			}
		})
	}
}

func TestAcceptAgent(t *testing.T) {
	tests := []struct {
		name           string
		setupInventory func(*inventory.Agents)
		request        *proto.AgentRequest
		expectError    bool
		errorCode      codes.Code
		expectedID     string
	}{
		{
			name: "accept candidate agent",
			setupInventory: func(inv *inventory.Agents) {
				_ = inv.AddCandidate(inventory.AgentIdentity{
					ID:          "candidate1",
					Address:     "addr1",
					Certificate: "cert1",
				})
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "candidate1",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError: false,
			expectedID:  "candidate1",
		},
		{
			name: "accept rejected agent",
			setupInventory: func(inv *inventory.Agents) {
				_ = inv.Reject(inventory.AgentIdentity{
					ID:          "rejected1",
					Address:     "addr1",
					Certificate: "cert1",
				})
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "rejected1",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError: false,
			expectedID:  "rejected1",
		},
		{
			name: "agent not found",
			setupInventory: func(inv *inventory.Agents) {
				// Empty inventory
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "unknown",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError: true,
			errorCode:   codes.NotFound,
		},
		{
			name: "already accepted agent",
			setupInventory: func(inv *inventory.Agents) {
				agent1 := inventory.AgentIdentity{
					ID:          "agent1",
					Address:     "addr1",
					Certificate: "cert1",
				}
				_ = inv.AddCandidate(agent1)
				_ = inv.Register(agent1, true)
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "agent1",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError: true,
			errorCode:   codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := newMockServer()
			db := setupTestDB(t)
			api := New(mockServer, db)

			tt.setupInventory(mockServer.inventory)

			ctx := context.Background()
			resp, err := api.AcceptAgent(ctx, tt.request)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("Expected gRPC status error, got: %v", err)
				}
				if st.Code() != tt.errorCode {
					t.Errorf("Expected error code %v, got %v", tt.errorCode, st.Code())
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if resp.Agent.Id != tt.expectedID {
					t.Errorf("Expected agent ID %s, got %s", tt.expectedID, resp.Agent.Id)
				}
			}
		})
	}
}

func TestRemoveAgent(t *testing.T) {
	tests := []struct {
		name            string
		setupInventory  func(*inventory.Agents)
		request         *proto.AgentRequest
		expectError     bool
		errorCode       codes.Code
		expectShutdown  bool
		shutdownAgentID agent.ID
	}{
		{
			name: "remove accepted agent",
			setupInventory: func(inv *inventory.Agents) {
				agent1 := inventory.AgentIdentity{
					ID:          "agent1",
					Address:     "addr1",
					Certificate: "cert1",
				}
				_ = inv.AddCandidate(agent1)
				_ = inv.Register(agent1, true)
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "agent1",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError:     false,
			expectShutdown:  true,
			shutdownAgentID: "agent1",
		},
		{
			name: "remove candidate agent",
			setupInventory: func(inv *inventory.Agents) {
				_ = inv.AddCandidate(inventory.AgentIdentity{
					ID:          "candidate1",
					Address:     "addr1",
					Certificate: "cert1",
				})
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "candidate1",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError:     false,
			expectShutdown:  true,
			shutdownAgentID: "candidate1",
		},
		{
			name: "remove rejected agent",
			setupInventory: func(inv *inventory.Agents) {
				_ = inv.Reject(inventory.AgentIdentity{
					ID:          "rejected1",
					Address:     "addr1",
					Certificate: "cert1",
				})
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "rejected1",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError:     false,
			expectShutdown:  true,
			shutdownAgentID: "rejected1",
		},
		{
			name: "remove non-existent agent",
			setupInventory: func(inv *inventory.Agents) {
				// Empty inventory
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "unknown",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError:    true,
			errorCode:      codes.NotFound,
			expectShutdown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := newMockServer()
			db := setupTestDB(t)
			api := New(mockServer, db)

			tt.setupInventory(mockServer.inventory)

			ctx := context.Background()
			resp, err := api.RemoveAgent(ctx, tt.request)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("Expected gRPC status error, got: %v", err)
				}
				if st.Code() != tt.errorCode {
					t.Errorf("Expected error code %v, got %v", tt.errorCode, st.Code())
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if resp == nil {
					t.Fatal("Expected response but got nil")
				}
			}

			if tt.expectShutdown {
				if len(mockServer.shutdownRequests) != 1 {
					t.Errorf("Expected 1 shutdown request, got %d", len(mockServer.shutdownRequests))
				} else if mockServer.shutdownRequests[0] != tt.shutdownAgentID {
					t.Errorf("Expected shutdown for agent %s, got %s", tt.shutdownAgentID, mockServer.shutdownRequests[0])
				}
			}
		})
	}
}

func TestRejectAgent(t *testing.T) {
	tests := []struct {
		name            string
		setupInventory  func(*inventory.Agents)
		request         *proto.AgentRequest
		expectError     bool
		errorCode       codes.Code
		expectShutdown  bool
		shutdownAgentID agent.ID
	}{
		{
			name: "reject candidate agent",
			setupInventory: func(inv *inventory.Agents) {
				_ = inv.AddCandidate(inventory.AgentIdentity{
					ID:          "candidate1",
					Address:     "addr1",
					Certificate: "cert1",
				})
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "candidate1",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError:    false,
			expectShutdown: false,
		},
		{
			name: "reject accepted agent",
			setupInventory: func(inv *inventory.Agents) {
				agent1 := inventory.AgentIdentity{
					ID:          "agent1",
					Address:     "addr1",
					Certificate: "cert1",
				}
				_ = inv.AddCandidate(agent1)
				_ = inv.Register(agent1, true)
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "agent1",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError:     false,
			expectShutdown:  true,
			shutdownAgentID: "agent1",
		},
		{
			name: "reject non-existent agent",
			setupInventory: func(inv *inventory.Agents) {
				// Empty inventory
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "unknown",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError:    true,
			errorCode:      codes.NotFound,
			expectShutdown: false,
		},
		{
			name: "reject already rejected agent",
			setupInventory: func(inv *inventory.Agents) {
				_ = inv.Reject(inventory.AgentIdentity{
					ID:          "rejected1",
					Address:     "addr1",
					Certificate: "cert1",
				})
			},
			request: &proto.AgentRequest{
				Agent: &proto.AgentInfo{
					Id:          "rejected1",
					Address:     strPtr("addr1"),
					Certificate: strPtr("cert1"),
				},
			},
			expectError:    true,
			errorCode:      codes.NotFound,
			expectShutdown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := newMockServer()
			db := setupTestDB(t)
			api := New(mockServer, db)

			tt.setupInventory(mockServer.inventory)

			ctx := context.Background()
			resp, err := api.RejectAgent(ctx, tt.request)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("Expected gRPC status error, got: %v", err)
				}
				if st.Code() != tt.errorCode {
					t.Errorf("Expected error code %v, got %v", tt.errorCode, st.Code())
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if resp == nil {
					t.Fatal("Expected response but got nil")
				}
			}

			if tt.expectShutdown {
				if len(mockServer.shutdownRequests) != 1 {
					t.Errorf("Expected 1 shutdown request, got %d", len(mockServer.shutdownRequests))
				} else if mockServer.shutdownRequests[0] != tt.shutdownAgentID {
					t.Errorf("Expected shutdown for agent %s, got %s", tt.shutdownAgentID, mockServer.shutdownRequests[0])
				}
			} else {
				if len(mockServer.shutdownRequests) > 0 {
					t.Errorf("Expected no shutdown requests, got %d", len(mockServer.shutdownRequests))
				}
			}
		})
	}
}

func TestGetRequest(t *testing.T) {
	tests := []struct {
		name        string
		setupDB     func(*badger.DB)
		requestID   string
		expectError bool
		expected    string
	}{
		{
			name: "get existing request",
			setupDB: func(db *badger.DB) {
				key := database.GenerateRequestKeyFromString("req123")
				value := []byte(`{"task":"test"}`)
				_ = db.Update(func(txn *badger.Txn) error {
					return txn.Set(key, value)
				})
			},
			requestID:   "req123",
			expectError: false,
			expected:    `{"task":"test"}`,
		},
		{
			name: "get non-existent request",
			setupDB: func(db *badger.DB) {
				// Empty DB
			},
			requestID:   "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := newMockServer()
			db := setupTestDB(t)
			api := New(mockServer, db)

			tt.setupDB(db)

			ctx := context.Background()
			req := &proto.RequestRequest{RequestID: tt.requestID}

			resp, err := api.GetRequest(ctx, req)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if resp.Request != tt.expected {
					t.Errorf("Expected request %s, got %s", tt.expected, resp.Request)
				}
			}
		})
	}
}

func TestGetResults(t *testing.T) {
	tests := []struct {
		name        string
		setupDB     func(*badger.DB)
		resultID    string
		expectError bool
		expected    string
	}{
		{
			name: "get existing result",
			setupDB: func(db *badger.DB) {
				key := database.GenerateResultKey("1234567890")
				value := []byte(`{"status":"success"}`)
				_ = db.Update(func(txn *badger.Txn) error {
					return txn.Set(key, value)
				})
			},
			resultID:    "1234567890",
			expectError: false,
			expected:    `{"status":"success"}`,
		},
		{
			name: "get non-existent result",
			setupDB: func(db *badger.DB) {
				// Empty DB
			},
			resultID:    "9999999999",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := newMockServer()
			db := setupTestDB(t)
			api := New(mockServer, db)

			tt.setupDB(db)

			ctx := context.Background()
			req := &proto.ResultsRequest{ResultID: tt.resultID}

			resp, err := api.GetResults(ctx, req)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if resp.Result != tt.expected {
					t.Errorf("Expected result %s, got %s", tt.expected, resp.Result)
				}
			}
		})
	}
}

func TestListResults(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name           string
		setupDB        func(*badger.DB)
		request        *proto.ListResultsRequest
		expectedCount  int
		validateResult func(*testing.T, *proto.ListResultsResponse)
	}{
		{
			name: "empty database",
			setupDB: func(db *badger.DB) {
				// Empty DB
			},
			request: &proto.ListResultsRequest{
				Limit: 10,
			},
			expectedCount: 0,
		},
		{
			name: "list all results with default limit",
			setupDB: func(db *badger.DB) {
				_ = db.Update(func(txn *badger.Txn) error {
					for i := 1; i <= 5; i++ {
						id := fmt.Sprintf("%d", now-int64(i)*1000)
						key := database.GenerateResultKey(id)
						task := database.Task{
							Agent: agent.ID(fmt.Sprintf("agent%d", i)),
							Result: &proto.TaskResponse{
								InternalError: proto.InternalError_OK,
							},
						}
						value, _ := serializer.JSON.Marshal(task)
						_ = txn.Set(key, value)
					}
					return nil
				})
			},
			request: &proto.ListResultsRequest{
				Limit: 10,
			},
			expectedCount: 5,
		},
		{
			name: "pagination with offset and limit",
			setupDB: func(db *badger.DB) {
				_ = db.Update(func(txn *badger.Txn) error {
					for i := 1; i <= 10; i++ {
						id := fmt.Sprintf("%d", now-int64(i)*1000)
						key := database.GenerateResultKey(id)
						task := database.Task{
							Agent: agent.ID(fmt.Sprintf("agent%d", i)),
							Result: &proto.TaskResponse{
								InternalError: proto.InternalError_OK,
							},
						}
						value, _ := serializer.JSON.Marshal(task)
						_ = txn.Set(key, value)
					}
					return nil
				})
			},
			request: &proto.ListResultsRequest{
				Offset: 3,
				Limit:  3,
			},
			expectedCount: 3,
		},
		{
			name: "filter by date range",
			setupDB: func(db *badger.DB) {
				_ = db.Update(func(txn *badger.Txn) error {
					for i := 1; i <= 10; i++ {
						id := fmt.Sprintf("%d", now-int64(i)*1000)
						key := database.GenerateResultKey(id)
						task := database.Task{
							Agent: agent.ID(fmt.Sprintf("agent%d", i)),
							Result: &proto.TaskResponse{
								InternalError: proto.InternalError_OK,
							},
						}
						value, _ := serializer.JSON.Marshal(task)
						_ = txn.Set(key, value)
					}
					return nil
				})
			},
			request: &proto.ListResultsRequest{
				FromDate: int64Ptr(now - 6000),
				ToDate:   int64Ptr(now - 2000),
				Limit:    10,
			},
			expectedCount: 5, // Results 2, 3, 4, 5, 6 (within range, boundaries inclusive)
		},
		{
			name: "filter by target agent",
			setupDB: func(db *badger.DB) {
				_ = db.Update(func(txn *badger.Txn) error {
					for i := 1; i <= 5; i++ {
						id := fmt.Sprintf("%d", now-int64(i)*1000)
						key := database.GenerateResultKey(id)
						agentID := "agent1"
						if i%2 == 0 {
							agentID = "agent2"
						}
						task := database.Task{
							Agent: agent.ID(agentID),
							Result: &proto.TaskResponse{
								InternalError: proto.InternalError_OK,
							},
						}
						value, _ := serializer.JSON.Marshal(task)
						_ = txn.Set(key, value)
					}
					return nil
				})
			},
			request: &proto.ListResultsRequest{
				Targets: []string{"agent1"},
				Limit:   10,
			},
			expectedCount: 3, // agent1 appears in positions 1, 3, 5
		},
		{
			name: "result with error status",
			setupDB: func(db *badger.DB) {
				_ = db.Update(func(txn *badger.Txn) error {
					id := fmt.Sprintf("%d", now)
					key := database.GenerateResultKey(id)
					task := database.Task{
						Agent: "agent1",
						Result: &proto.TaskResponse{
							InternalError: proto.InternalError_OK,
							Error:         "some error occurred",
						},
					}
					value, _ := serializer.JSON.Marshal(task)
					_ = txn.Set(key, value)
					return nil
				})
			},
			request: &proto.ListResultsRequest{
				Limit: 10,
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, resp *proto.ListResultsResponse) {
				t.Helper()
				if len(resp.Results) != 1 {
					return
				}
				if resp.Results[0].Status != "error" {
					t.Errorf("Expected status 'error', got '%s'", resp.Results[0].Status)
				}
				if resp.Results[0].Error != "some error occurred" {
					t.Errorf("Expected error message 'some error occurred', got '%s'", resp.Results[0].Error)
				}
			},
		},
		{
			name: "result with internal error",
			setupDB: func(db *badger.DB) {
				_ = db.Update(func(txn *badger.Txn) error {
					id := fmt.Sprintf("%d", now)
					key := database.GenerateResultKey(id)
					task := database.Task{
						Agent: "agent1",
						Result: &proto.TaskResponse{
							InternalError: proto.InternalError_TIMEOUT,
						},
					}
					value, _ := serializer.JSON.Marshal(task)
					_ = txn.Set(key, value)
					return nil
				})
			},
			request: &proto.ListResultsRequest{
				Limit: 10,
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, resp *proto.ListResultsResponse) {
				t.Helper()
				if len(resp.Results) != 1 {
					return
				}
				if resp.Results[0].Status != "internal error" {
					t.Errorf("Expected status 'internal error', got '%s'", resp.Results[0].Status)
				}
				if resp.Results[0].InternalError != proto.InternalError_TIMEOUT {
					t.Errorf("Expected internal error TIMEOUT, got %v", resp.Results[0].InternalError)
				}
			},
		},
		{
			name: "grouped result",
			setupDB: func(db *badger.DB) {
				_ = db.Update(func(txn *badger.Txn) error {
					id := fmt.Sprintf("%d", now)
					key := database.GenerateResultKey(id)
					value := []byte("grouped:some-data")
					_ = txn.Set(key, value)
					return nil
				})
			},
			request: &proto.ListResultsRequest{
				Limit: 10,
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, resp *proto.ListResultsResponse) {
				t.Helper()
				t.Helper()
				if len(resp.Results) != 1 {
					return
				}
				if resp.Results[0].Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Results[0].Status)
				}
				if resp.Results[0].Agent != "grouped:some-data" {
					t.Errorf("Expected agent 'grouped:some-data', got '%s'", resp.Results[0].Agent)
				}
			},
		},
		{
			name: "invalid result data",
			setupDB: func(db *badger.DB) {
				_ = db.Update(func(txn *badger.Txn) error {
					id := fmt.Sprintf("%d", now)
					key := database.GenerateResultKey(id)
					value := []byte("invalid json data")
					_ = txn.Set(key, value)
					return nil
				})
			},
			request: &proto.ListResultsRequest{
				Limit: 10,
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, resp *proto.ListResultsResponse) {
				t.Helper()
				if len(resp.Results) != 1 {
					return
				}
				if resp.Results[0].Status != "unknown" {
					t.Errorf("Expected status 'unknown', got '%s'", resp.Results[0].Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := newMockServer()
			db := setupTestDB(t)
			api := New(mockServer, db)

			tt.setupDB(db)

			ctx := context.Background()
			resp, err := api.ListResults(ctx, tt.request)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(resp.Results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(resp.Results))
			}

			if tt.validateResult != nil {
				tt.validateResult(t, resp)
			}
		})
	}
}

func TestListResultsOrdering(t *testing.T) {
	mockServer := newMockServer()
	db := setupTestDB(t)
	api := New(mockServer, db)

	// Insert results with specific timestamps
	timestamps := []int64{1000, 2000, 3000, 4000, 5000}
	_ = db.Update(func(txn *badger.Txn) error {
		for _, ts := range timestamps {
			id := strconv.FormatInt(ts, 10)
			key := database.GenerateResultKey(id)
			task := database.Task{
				Agent: agent.ID(fmt.Sprintf("agent-%d", ts)),
				Result: &proto.TaskResponse{
					InternalError: proto.InternalError_OK,
				},
			}
			value, _ := serializer.JSON.Marshal(task)
			_ = txn.Set(key, value)
		}
		return nil
	})

	ctx := context.Background()
	req := &proto.ListResultsRequest{Limit: 10}

	resp, err := api.ListResults(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(resp.Results) != 5 {
		t.Fatalf("Expected 5 results, got %d", len(resp.Results))
	}

	// Verify reverse chronological order (newest first)
	for i := 0; i < len(resp.Results); i++ {
		expectedTS := timestamps[len(timestamps)-1-i]
		if resp.Results[i].Id != expectedTS {
			t.Errorf("Result %d: expected ID %d, got %d", i, expectedTS, resp.Results[i].Id)
		}
	}
}

func TestToProtoAgentSlice(t *testing.T) {
	now := time.Now()
	agents := []inventory.AgentIdentity{
		{ID: "agent1", Address: "addr1", Certificate: "cert1"},
		{ID: "agent2", Address: "addr2", Certificate: "cert2"},
	}

	states := map[agent.ID]inventory.AgentState{
		"agent1": {
			Connected: true,
			Since:     now,
			LastMsg:   now.Add(5 * time.Second),
		},
	}

	result := toProtoAgentSlice(agents, states)

	if len(result) != 2 {
		t.Fatalf("Expected 2 agents, got %d", len(result))
	}

	// Check agent1 with state
	if result[0].Id != "agent1" {
		t.Errorf("Expected agent1, got %s", result[0].Id)
	}
	if result[0].IsConnected == nil || !*result[0].IsConnected {
		t.Error("Expected agent1 to be connected")
	}
	if result[0].Since == nil {
		t.Error("Expected agent1 to have Since timestamp")
	}
	if result[0].LastMsg == nil {
		t.Error("Expected agent1 to have LastMsg timestamp")
	}

	// Check agent2 without state
	if result[1].Id != "agent2" {
		t.Errorf("Expected agent2, got %s", result[1].Id)
	}
	if result[1].IsConnected != nil {
		t.Error("Expected agent2 IsConnected to be nil")
	}
}

// Helper functions.
func strPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}
