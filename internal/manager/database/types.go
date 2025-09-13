package database

import (
	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/proto"
)

const (
	ResultKeyPrefix  = "res"
	RequestKeyPrefix = "req"
)

type Task struct {
	Agent  agent.ID
	Result *proto.TaskResponse
}

type Request struct {
	Task               string
	ConnectedTarget    []string
	DisconnectedTarget []string
}

type Key struct {
	Prefix string
	ID     string
}
