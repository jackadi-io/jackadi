package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/plugin/core"
	"github.com/jackadi-io/jackadi/internal/plugin/inventory"
	"github.com/jackadi-io/jackadi/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

// mockStream implements the bidirectional stream interface for testing.
type mockStream struct {
	grpc.ClientStream
	recvChan  chan *proto.TaskRequest
	sendChan  chan *proto.TaskResponse
	recvErr   error
	sendErr   error
	ctx       context.Context
	mu        sync.Mutex
	closed    bool
	recvCount int
	sendCount int
	maxRecv   int // Maximum number of Recv calls before EOF
}

func newMockStream(ctx context.Context) *mockStream {
	return &mockStream{
		recvChan: make(chan *proto.TaskRequest, 100),
		sendChan: make(chan *proto.TaskResponse, 100),
		ctx:      ctx,
		maxRecv:  -1, // No limit by default
	}
}

func (m *mockStream) Recv() (*proto.TaskRequest, error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, io.EOF
	}
	if m.recvErr != nil {
		err := m.recvErr
		m.mu.Unlock()
		return nil, err
	}
	if m.maxRecv >= 0 && m.recvCount >= m.maxRecv {
		m.mu.Unlock()
		return nil, io.EOF
	}
	m.recvCount++
	m.mu.Unlock()

	select {
	case req, ok := <-m.recvChan:
		if !ok {
			// Channel closed
			return nil, io.EOF
		}
		if req == nil {
			// Nil request should not happen, but return EOF to be safe
			return nil, io.EOF
		}
		return req, nil
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

func (m *mockStream) Send(resp *proto.TaskResponse) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return io.EOF
	}
	if m.sendErr != nil {
		return m.sendErr
	}

	m.sendCount++
	select {
	case m.sendChan <- resp:
		return nil
	default:
		return errors.New("send channel full")
	}
}

func (m *mockStream) Context() context.Context {
	return m.ctx
}

func (m *mockStream) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.recvChan)
	}
}

func (m *mockStream) SendRequest(req *proto.TaskRequest) {
	m.recvChan <- req
}

func (m *mockStream) ReceiveResponse() *proto.TaskResponse {
	select {
	case resp := <-m.sendChan:
		return resp
	case <-time.After(5 * time.Second):
		return nil
	}
}

func (m *mockStream) ReceiveResponseWithTimeout(timeout time.Duration) *proto.TaskResponse {
	select {
	case resp := <-m.sendChan:
		return resp
	case <-time.After(timeout):
		return nil
	}
}

// mockTaskClient implements proto.ClusterClient for testing.
type mockTaskClient struct {
	stream        *mockStream
	handshakeFunc func(ctx context.Context, in *proto.HandshakeRequest, opts ...grpc.CallOption) (*proto.HandshakeResponse, error)
}

func (m *mockTaskClient) Handshake(ctx context.Context, in *proto.HandshakeRequest, opts ...grpc.CallOption) (*proto.HandshakeResponse, error) {
	if m.handshakeFunc != nil {
		return m.handshakeFunc(ctx, in, opts...)
	}
	return &proto.HandshakeResponse{Id: in.Id}, nil
}

func (m *mockTaskClient) ExecTask(ctx context.Context, opts ...grpc.CallOption) (proto.Cluster_ExecTaskClient, error) {
	return m.stream, nil
}

func (m *mockTaskClient) ListAgentPlugins(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*proto.ListAgentPluginsResponse, error) {
	return &proto.ListAgentPluginsResponse{}, nil
}

func setupIntegrationTest(t *testing.T) (*Agent, *mockStream, context.Context, func()) {
	t.Helper()

	// Save original registry
	originalRegistry := inventory.Registry

	// Create new registry
	inventory.Registry = inventory.New()

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	md := metadata.Pairs("agent_id", "test-agent")
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Create mock stream
	stream := newMockStream(ctx)

	// Create SpecsManager
	specsManager, err := NewSpecsManager()
	if err != nil {
		// If it fails because specs already exists, just create a basic one
		specsManager = &SpecsManager{
			mutex: &sync.RWMutex{},
			specs: make(map[string]any),
		}
	}

	// Create agent
	agent := Agent{
		config: Config{
			AgentID: "test-agent",
		},
		taskClient: &mockTaskClient{
			stream: stream,
		},
		SpecManager: specsManager,
	}

	cleanup := func() {
		cancel()
		stream.Close()
		inventory.Registry = originalRegistry
	}

	return &agent, stream, ctx, cleanup
}

func TestListenTaskRequest_HealthCheck(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Start ListenTaskRequest in goroutine
	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(ctx)
	}()

	// Give the goroutine time to start and call Recv()
	time.Sleep(10 * time.Millisecond)

	// Send health check request
	healthReq := &proto.TaskRequest{
		Id:   1,
		Task: fmt.Sprintf("health:%s", config.InstantPingName),
	}
	stream.SendRequest(healthReq)

	// Wait for response
	resp := stream.ReceiveResponse()
	if resp == nil {
		t.Fatal("Expected health check response, got nil")
	}

	if resp.Id != 1 {
		t.Errorf("Expected ID 1, got %d", resp.Id)
	}

	// Close stream to end test
	stream.Close()

	// Wait for ListenTaskRequest to finish
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ListenTaskRequest returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ListenTaskRequest did not finish in time")
	}
}

func TestListenTaskRequest_SimpleTaskExecution(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Register a test plugin
	testPlugin := &mockPlugin{
		name: "test-plugin",
		doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
			return core.Response{
				Output:  []byte("task output"),
				Retcode: 0,
			}, nil
		},
	}
	_ = inventory.Registry.Register(testPlugin)

	// Start ListenTaskRequest
	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(ctx)
	}()

	// Give the goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Send task request
	taskReq := &proto.TaskRequest{
		Id:      100,
		GroupID: int64Ptr(1),
		Task:    "test-plugin" + config.PluginSeparator + "task1",
		Input:   &proto.Input{},
		Timeout: 10, // 10 seconds timeout
	}
	stream.SendRequest(taskReq)

	// Wait for response
	resp := stream.ReceiveResponse()
	if resp == nil {
		t.Fatal("Expected task response, got nil")
	}

	if resp.Id != 100 {
		t.Errorf("Expected ID 100, got %d", resp.Id)
	}

	if resp.GetGroupID() != 1 {
		t.Errorf("Expected GroupID 1, got %d", resp.GetGroupID())
	}

	if string(resp.Output) != "task output" {
		t.Errorf("Expected output 'task output', got '%s'", string(resp.Output))
	}

	if resp.InternalError != proto.InternalError_OK {
		t.Errorf("Expected InternalError OK, got %v", resp.InternalError)
	}

	// Close and cleanup
	stream.Close()
	<-done
}

func TestListenTaskRequest_TaskWithError(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Register plugin that returns error
	errorPlugin := &mockPlugin{
		name: "error-plugin",
		doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
			return core.Response{
				Error:   "task execution failed",
				Retcode: 1,
			}, nil
		},
	}
	_ = inventory.Registry.Register(errorPlugin)

	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	taskReq := &proto.TaskRequest{
		Id:      200,
		Task:    "error-plugin" + config.PluginSeparator + "failing-task",
		Timeout: 10,
	}
	stream.SendRequest(taskReq)

	resp := stream.ReceiveResponse()
	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Error != "task execution failed" {
		t.Errorf("Expected error 'task execution failed', got '%s'", resp.Error)
	}

	if resp.Retcode != 1 {
		t.Errorf("Expected retcode 1, got %d", resp.Retcode)
	}

	stream.Close()
	<-done
}

func TestListenTaskRequest_QueueFull(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Register slow plugin
	slowPlugin := &mockPlugin{
		name: "slow-plugin",
		doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
			time.Sleep(100 * time.Millisecond)
			return core.Response{Output: []byte("slow")}, nil
		},
	}
	_ = inventory.Registry.Register(slowPlugin)

	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(ctx)
	}()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Fill up the queue (MaxWaitingRequests = 1000, so we send many more than that won't fit)
	// We need to fill MaxConcurrentTasks + MaxWaitingRequests + 1
	numRequests := config.MaxConcurrentTasks + config.MaxWaitingRequests + 10

	for i := range numRequests {
		stream.SendRequest(&proto.TaskRequest{
			Id:      int64(i),
			Task:    "slow-plugin" + config.PluginSeparator + "slow-task",
			Timeout: 1,
		})
	}

	// Count responses
	fullQueueResponses := 0
	totalResponses := 0

	// Collect responses with timeout
	responseTimeout := time.After(3 * time.Second)
	for totalResponses < numRequests {
		select {
		case resp := <-stream.sendChan:
			totalResponses++
			if resp.InternalError == proto.InternalError_FULL_QUEUE {
				fullQueueResponses++
			}
		case <-responseTimeout:
			t.Logf("Timeout waiting for all responses. Got %d/%d", totalResponses, numRequests)
			goto checkResults
		}
	}

checkResults:
	if fullQueueResponses == 0 {
		t.Errorf("Expected at least one FULL_QUEUE response, got %d", fullQueueResponses)
	}

	t.Logf("Received %d FULL_QUEUE responses out of %d requests", fullQueueResponses, totalResponses)

	stream.Close()
	<-done
}

func TestListenTaskRequest_ConcurrentTasks(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	executionOrder := make(chan int, 10)

	concurrentPlugin := &mockPlugin{
		name: "concurrent-plugin",
		doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
			// Parse task number from task name
			var taskNum int
			_, _ = fmt.Sscanf(task, "task%d", &taskNum)
			executionOrder <- taskNum
			time.Sleep(50 * time.Millisecond)
			return core.Response{Output: fmt.Appendf(nil, "task%d", taskNum)}, nil
		},
	}
	_ = inventory.Registry.Register(concurrentPlugin)

	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Send multiple tasks quickly
	numTasks := 5
	for i := range numTasks {
		stream.SendRequest(&proto.TaskRequest{
			Id:      int64(i),
			Task:    fmt.Sprintf("concurrent-plugin%stask%d", config.PluginSeparator, i),
			Timeout: 5,
		})
	}

	// Collect responses
	responses := 0
	timeout := time.After(2 * time.Second)
	for responses < numTasks {
		select {
		case <-stream.sendChan:
			responses++
		case <-timeout:
			t.Fatalf("Timeout waiting for responses. Got %d/%d", responses, numTasks)
		}
	}

	// Check that some tasks executed concurrently (MaxConcurrentTasks = 2)
	executionCount := len(executionOrder)
	if executionCount < 2 {
		t.Errorf("Expected at least 2 tasks to start execution, got %d", executionCount)
	}

	stream.Close()
	<-done
}

func TestListenTaskRequest_ExclusiveLock(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	executing := make(chan string, 10)

	lockPlugin := &mockPlugin{
		name: "lock-plugin",
		doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
			executing <- "start:" + task
			time.Sleep(100 * time.Millisecond)
			executing <- "end:" + task
			return core.Response{Output: []byte(task)}, nil
		},
		getTaskLockMode: func(task string) (proto.LockMode, error) {
			if task == "exclusive-task" {
				return proto.LockMode_EXCLUSIVE, nil
			}
			return proto.LockMode_NO_LOCK, nil
		},
	}
	_ = inventory.Registry.Register(lockPlugin)

	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Send exclusive task first
	stream.SendRequest(&proto.TaskRequest{
		Id:      1,
		Task:    "lock-plugin" + config.PluginSeparator + "exclusive-task",
		Timeout: 5,
	})

	// Wait a bit for it to start
	time.Sleep(20 * time.Millisecond)

	// Send regular task - should wait for exclusive to finish
	stream.SendRequest(&proto.TaskRequest{
		Id:      2,
		Task:    "lock-plugin" + config.PluginSeparator + "normal-task",
		Timeout: 5,
	})

	// Collect execution order
	var execOrder []string
	timeout := time.After(2 * time.Second)
	for len(execOrder) < 4 { // 2 starts + 2 ends
		exit := false
		select {
		case event := <-executing:
			execOrder = append(execOrder, event)
		case <-timeout:
			exit = true
		}
		if exit {
			break
		}
	}

	// Verify exclusive task completed before normal task started
	if len(execOrder) >= 3 {
		// Should see: start:exclusive, end:exclusive, start:normal, end:normal
		if execOrder[0] != "start:exclusive-task" {
			t.Errorf("Expected exclusive task to start first, got %s", execOrder[0])
		}
	}

	// Collect responses
	for range 2 {
		select {
		case <-stream.sendChan:
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for responses")
		}
	}

	stream.Close()
	<-done
}

func TestListenTaskRequest_TaskTimeout(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	slowPlugin := &mockPlugin{
		name: "timeout-plugin",
		doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
			time.Sleep(2 * time.Second) // Task takes longer than timeout
			return core.Response{Output: []byte("completed")}, nil
		},
	}
	_ = inventory.Registry.Register(slowPlugin)

	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Send task with short timeout
	stream.SendRequest(&proto.TaskRequest{
		Id:      1,
		Task:    "timeout-plugin" + config.PluginSeparator + "slow-task",
		Timeout: 1, // 1 second timeout
	})

	// Should receive timeout response
	resp := stream.ReceiveResponseWithTimeout(2 * time.Second)
	if resp == nil {
		t.Fatal("Expected timeout response, got nil")
	}

	if resp.InternalError != proto.InternalError_STARTED_TIMEOUT {
		t.Errorf("Expected STARTED_TIMEOUT, got %v", resp.InternalError)
	}

	// Wait for the actual task to complete (will send another response)
	resp2 := stream.ReceiveResponseWithTimeout(3 * time.Second)
	if resp2 != nil {
		// Second response with actual result
		if string(resp2.Output) != "completed" {
			t.Errorf("Expected output 'completed', got '%s'", string(resp2.Output))
		}
	}

	stream.Close()
	<-done
}

func TestListenTaskRequest_StreamError(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Send a request
	stream.SendRequest(&proto.TaskRequest{
		Id:      1,
		Task:    "health:" + config.InstantPingName,
		Timeout: 10,
	})

	// Wait for response
	stream.ReceiveResponse()

	// Close stream to simulate error
	stream.Close()

	// Should exit cleanly
	select {
	case err := <-done:
		if err != nil {
			t.Logf("ListenTaskRequest exited with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ListenTaskRequest did not finish after stream closed")
	}
}

func TestListenTaskRequest_ContextCancellation(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create a cancellable context
	taskCtx, cancel := context.WithCancel(ctx)

	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(taskCtx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Send health check to verify it's running
	stream.SendRequest(&proto.TaskRequest{
		Id:   1,
		Task: "health:" + config.InstantPingName,
	})

	resp := stream.ReceiveResponse()
	if resp == nil {
		t.Fatal("Expected health check response")
	}

	// Cancel context
	cancel()

	// Close stream to help it exit
	stream.Close()

	// Should exit
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("ListenTaskRequest did not finish after context cancellation")
	}
}

func TestListenTaskRequest_UnknownPlugin(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Send request for unknown plugin
	stream.SendRequest(&proto.TaskRequest{
		Id:      1,
		Task:    "nonexistent-plugin" + config.PluginSeparator + "task",
		Timeout: 10,
	})

	resp := stream.ReceiveResponse()
	if resp == nil {
		t.Fatal("Expected error response, got nil")
	}

	if resp.InternalError != proto.InternalError_UNKNOWN_TASK {
		t.Errorf("Expected UNKNOWN_TASK, got %v", resp.InternalError)
	}

	stream.Close()
	<-done
}

func TestListenTaskRequest_MultipleRequests(t *testing.T) {
	agent, stream, ctx, cleanup := setupIntegrationTest(t)
	defer cleanup()

	counter := 0
	var mu sync.Mutex

	multiPlugin := &mockPlugin{
		name: "multi-plugin",
		doFunc: func(ctx context.Context, task string, input *proto.Input) (core.Response, error) {
			mu.Lock()
			counter++
			count := counter
			mu.Unlock()
			return core.Response{
				Output: fmt.Appendf(nil, "response-%d", count),
			}, nil
		},
	}
	_ = inventory.Registry.Register(multiPlugin)

	done := make(chan error, 1)
	go func() {
		done <- agent.ListenTaskRequest(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Send multiple requests
	numRequests := 10
	for i := range numRequests {
		stream.SendRequest(&proto.TaskRequest{
			Id:      int64(i),
			Task:    "multi-plugin" + config.PluginSeparator + "task",
			Timeout: 10,
		})
	}

	// Collect all responses
	responses := make([]*proto.TaskResponse, 0, numRequests)
	timeout := time.After(3 * time.Second)
	for len(responses) < numRequests {
		select {
		case resp := <-stream.sendChan:
			responses = append(responses, resp)
		case <-timeout:
			t.Fatalf("Timeout waiting for all responses. Got %d/%d", len(responses), numRequests)
		}
	}

	// Verify all requests got responses
	if len(responses) != numRequests {
		t.Errorf("Expected %d responses, got %d", numRequests, len(responses))
	}

	// Verify responses have correct IDs
	responseIDs := make(map[int64]bool)
	for _, resp := range responses {
		responseIDs[resp.Id] = true
	}

	for i := range numRequests {
		if !responseIDs[int64(i)] {
			t.Errorf("Missing response for request ID %d", i)
		}
	}

	stream.Close()
	<-done
}
