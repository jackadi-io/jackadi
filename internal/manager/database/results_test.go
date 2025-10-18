package database

import (
	"testing"

	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/proto"
)

func TestCutGroupPrefix(t *testing.T) {
	tests := []struct {
		name          string
		result        string
		wantValue     string
		wantIsGrouped bool
	}{
		{
			name:          "grouped result",
			result:        "grouped:123,456,789",
			wantValue:     "123,456,789",
			wantIsGrouped: true,
		},
		{
			name:          "non-grouped result",
			result:        "normal result",
			wantValue:     "normal result",
			wantIsGrouped: false,
		},
		{
			name:          "empty string",
			result:        "",
			wantValue:     "",
			wantIsGrouped: false,
		},
		{
			name:          "grouped with empty value",
			result:        "grouped:",
			wantValue:     "",
			wantIsGrouped: true,
		},
		{
			name:          "grouped single ID",
			result:        "grouped:123",
			wantValue:     "123",
			wantIsGrouped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotIsGrouped := CutGroupPrefix(tt.result)
			if gotValue != tt.wantValue {
				t.Errorf("CutGroupPrefix() value = %v, want %v", gotValue, tt.wantValue)
			}
			if gotIsGrouped != tt.wantIsGrouped {
				t.Errorf("CutGroupPrefix() isGrouped = %v, want %v", gotIsGrouped, tt.wantIsGrouped)
			}
		})
	}
}

func TestGetFirstGroupedResultID(t *testing.T) {
	tests := []struct {
		name         string
		groupedValue string
		want         string
	}{
		{
			name:         "single ID",
			groupedValue: "123",
			want:         "123",
		},
		{
			name:         "multiple IDs",
			groupedValue: "123,456,789",
			want:         "123",
		},
		{
			name:         "two IDs",
			groupedValue: "100,200",
			want:         "100",
		},
		{
			name:         "empty string",
			groupedValue: "",
			want:         "",
		},
		{
			name:         "trailing comma",
			groupedValue: "123,",
			want:         "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFirstGroupedResultID(tt.groupedValue)
			if got != tt.want {
				t.Errorf("GetFirstGroupedResultID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractRequestIDFromResult(t *testing.T) {
	tests := []struct {
		name     string
		result   string
		resultID string
		want     string
		wantErr  bool
	}{
		{
			name:     "grouped result",
			result:   "grouped:123,456",
			resultID: "999",
			want:     "req:999",
		},
		{
			name:     "task with group ID",
			result:   mustMarshalTask("agent1", &proto.TaskResponse{GroupID: int64Ptr(100)}),
			resultID: "555",
			want:     "req:100",
		},
		{
			name:     "task with ID",
			result:   mustMarshalTask("agent2", &proto.TaskResponse{Id: 200}),
			resultID: "666",
			want:     "req:200",
		},
		{
			name:     "task with nil result",
			result:   mustMarshalTask("agent3", nil),
			resultID: "777",
			want:     "req:777",
		},
		{
			name:     "task with empty result",
			result:   mustMarshalTask("agent4", &proto.TaskResponse{}),
			resultID: "888",
			want:     "req:888",
		},
		{
			name:     "task with both group ID and ID prefers group ID",
			result:   mustMarshalTask("agent5", &proto.TaskResponse{Id: 300, GroupID: int64Ptr(400)}),
			resultID: "999",
			want:     "req:400",
		},
		{
			name:     "invalid JSON",
			result:   "invalid json",
			resultID: "111",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractRequestIDFromResult(tt.result, tt.resultID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractRequestIDFromResult() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ExtractRequestIDFromResult() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractRequestIDFromResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustMarshalTask(agentID agent.ID, result *proto.TaskResponse) string {
	data, err := MarshalTask(agentID, result)
	if err != nil {
		panic(err)
	}
	return string(data)
}
