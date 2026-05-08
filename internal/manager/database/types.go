package database

import (
	"github.com/jackadi-io/jackadi/internal/node"
	"github.com/jackadi-io/jackadi/internal/proto"
)

const (
	ResultKeyPrefix  = "res"
	RequestKeyPrefix = "req"
)

type Task struct {
	Node   node.ID
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
