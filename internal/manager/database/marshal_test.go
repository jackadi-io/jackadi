package database

import (
	"testing"

	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/proto"
)

func TestMarshalUnmarshalTask(t *testing.T) {
	tests := []struct {
		name    string
		agentID agent.ID
		result  *proto.TaskResponse
	}{
		{
			name:    "task with nil result",
			agentID: "agent1",
			result:  nil,
		},
		{
			name:    "task with empty result",
			agentID: "agent2",
			result:  &proto.TaskResponse{},
		},
		{
			name:    "task with ID",
			agentID: "agent3",
			result: &proto.TaskResponse{
				Id: 123,
			},
		},
		{
			name:    "task with group ID",
			agentID: "agent4",
			result: &proto.TaskResponse{
				GroupID: int64Ptr(456),
			},
		},
		{
			name:    "task with error",
			agentID: "agent5",
			result: &proto.TaskResponse{
				Id:    789,
				Error: "test error",
			},
		},
		{
			name:    "task with internal error",
			agentID: "agent6",
			result: &proto.TaskResponse{
				Id:            999,
				InternalError: proto.InternalError_TIMEOUT,
			},
		},
		{
			name:    "task with output",
			agentID: "agent7",
			result: &proto.TaskResponse{
				Id:     111,
				Output: []byte(`{"key":"value"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := MarshalTask(tt.agentID, tt.result)
			if err != nil {
				t.Errorf("MarshalTask() error = %v", err)
				return
			}

			task, err := UnmarshalTask(data)
			if err != nil {
				t.Errorf("UnmarshalTask() error = %v", err)
				return
			}

			if task.Agent != tt.agentID {
				t.Errorf("Agent = %v, want %v", task.Agent, tt.agentID)
			}

			if tt.result == nil {
				if task.Result != nil {
					t.Errorf("Result = %v, want nil", task.Result)
				}
				return
			}

			if task.Result == nil {
				t.Errorf("Result = nil, want non-nil")
				return
			}

			if task.Result.Id != tt.result.Id {
				t.Errorf("Result.Id = %v, want %v", task.Result.Id, tt.result.Id)
			}
			if (task.Result.GroupID == nil && tt.result.GroupID != nil) || (task.Result.GroupID != nil && tt.result.GroupID == nil) {
				t.Errorf("Result.GroupID = %v, want %v", task.Result.GroupID, tt.result.GroupID)
			} else if task.Result.GroupID != nil && tt.result.GroupID != nil && *task.Result.GroupID != *tt.result.GroupID {
				t.Errorf("Result.GroupID = %v, want %v", *task.Result.GroupID, *tt.result.GroupID)
			}
			if task.Result.Error != tt.result.Error {
				t.Errorf("Result.Error = %v, want %v", task.Result.Error, tt.result.Error)
			}
			if task.Result.InternalError != tt.result.InternalError {
				t.Errorf("Result.InternalError = %v, want %v", task.Result.InternalError, tt.result.InternalError)
			}
			if string(task.Result.Output) != string(tt.result.Output) {
				t.Errorf("Result.Output = %v, want %v", string(task.Result.Output), string(tt.result.Output))
			}
		})
	}
}

func TestUnmarshalTaskInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "invalid JSON",
			data: []byte("invalid json"),
		},
		{
			name: "empty data",
			data: []byte(""),
		},
		{
			name: "malformed JSON",
			data: []byte("{incomplete"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalTask(tt.data)
			if err == nil {
				t.Errorf("UnmarshalTask() expected error, got nil")
			}
		})
	}
}

func TestMarshalUnmarshalRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *Request
	}{
		{
			name: "request with task only",
			request: &Request{
				Task: "test:task",
			},
		},
		{
			name: "request with connected targets",
			request: &Request{
				Task:            "test:task",
				ConnectedTarget: []string{"agent1", "agent2"},
			},
		},
		{
			name: "request with disconnected targets",
			request: &Request{
				Task:               "test:task",
				DisconnectedTarget: []string{"agent3"},
			},
		},
		{
			name: "request with all fields",
			request: &Request{
				Task:               "test:task",
				ConnectedTarget:    []string{"agent1", "agent2"},
				DisconnectedTarget: []string{"agent3", "agent4"},
			},
		},
		{
			name: "request with empty targets",
			request: &Request{
				Task:               "test:task",
				ConnectedTarget:    []string{},
				DisconnectedTarget: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := MarshalRequest(tt.request)
			if err != nil {
				t.Errorf("MarshalRequest() error = %v", err)
				return
			}

			req, err := UnmarshalRequest(data)
			if err != nil {
				t.Errorf("UnmarshalRequest() error = %v", err)
				return
			}

			if req.Task != tt.request.Task {
				t.Errorf("Task = %v, want %v", req.Task, tt.request.Task)
			}

			if len(req.ConnectedTarget) != len(tt.request.ConnectedTarget) {
				t.Errorf("ConnectedTarget length = %v, want %v", len(req.ConnectedTarget), len(tt.request.ConnectedTarget))
			}
			for i, target := range tt.request.ConnectedTarget {
				if i >= len(req.ConnectedTarget) || req.ConnectedTarget[i] != target {
					t.Errorf("ConnectedTarget[%d] = %v, want %v", i, req.ConnectedTarget[i], target)
				}
			}

			if len(req.DisconnectedTarget) != len(tt.request.DisconnectedTarget) {
				t.Errorf("DisconnectedTarget length = %v, want %v", len(req.DisconnectedTarget), len(tt.request.DisconnectedTarget))
			}
			for i, target := range tt.request.DisconnectedTarget {
				if i >= len(req.DisconnectedTarget) || req.DisconnectedTarget[i] != target {
					t.Errorf("DisconnectedTarget[%d] = %v, want %v", i, req.DisconnectedTarget[i], target)
				}
			}
		})
	}
}

func TestUnmarshalRequestInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "invalid JSON",
			data: []byte("invalid json"),
		},
		{
			name: "empty data",
			data: []byte(""),
		},
		{
			name: "malformed JSON",
			data: []byte("{incomplete"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalRequest(tt.data)
			if err == nil {
				t.Errorf("UnmarshalRequest() expected error, got nil")
			}
		})
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}
