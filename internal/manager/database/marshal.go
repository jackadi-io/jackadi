package database

import (
	"encoding/json"

	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/proto"
)

// MarshalTask serializes a task with its agent and result for database storage.
func MarshalTask(agentID agent.ID, result *proto.TaskResponse) ([]byte, error) {
	task := Task{Agent: agentID, Result: result}
	return json.Marshal(task)
}

// UnmarshalTask deserializes task data from the database.
func UnmarshalTask(data []byte) (*Task, error) {
	var task Task
	err := json.Unmarshal(data, &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// MarshalRequest serializes a request for database storage.
func MarshalRequest(req *Request) ([]byte, error) {
	return json.Marshal(req)
}

// UnmarshalRequest deserializes request data from the database.
func UnmarshalRequest(data []byte) (*Request, error) {
	var request Request
	err := json.Unmarshal(data, &request)
	if err != nil {
		return nil, err
	}
	return &request, nil
}
