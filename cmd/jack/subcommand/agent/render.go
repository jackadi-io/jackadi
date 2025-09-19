package agent

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackadi-io/jackadi/cmd/jack/option"
	"github.com/jackadi-io/jackadi/cmd/jack/style"
	"github.com/jackadi-io/jackadi/internal/proto"
)

func sortAgentFunc(a *proto.AgentInfo, b *proto.AgentInfo) int {
	if a == nil || b == nil {
		return 0
	}
	return strings.Compare(a.Id, b.Id)
}

func prettyAgentListSprint(agents []*proto.AgentInfo, showDetails bool) string {
	items := ""
	if option.GetSortOutput() {
		slices.SortFunc(agents, sortAgentFunc)
	}

	for _, agent := range agents {
		if !showDetails {
			items += style.Item(agent.GetId())
			continue
		}
		if agent.GetCertificate() != "" {
			items += style.Item(fmt.Sprintf("%s (%s %s)", agent.GetId(), agent.GetAddress(), agent.GetCertificate()))
		} else {
			items += style.Item(fmt.Sprintf("%s (%s)", agent.GetId(), agent.GetAddress()))
		}
	}

	return style.SpacedBlock(items)
}

func prettyTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format("January 2, 2006 at 15:04 UTC")
}

func prettyAgentsHealthSprint(agents []*proto.AgentInfo, showDetails bool) string {
	items := ""
	if option.GetSortOutput() {
		slices.SortFunc(agents, sortAgentFunc)
	}

	for _, agent := range agents {
		connectedState := "connected"

		if !agent.GetIsConnected() {
			connectedState = "disconnected"
		}

		lastActive := agent.GetLastMsg().AsTime()

		if !showDetails {
			items += style.Item(fmt.Sprintf("%s (%s)", agent.GetId(), connectedState))
			continue
		}

		items += style.Item(agent.GetId())
		items += style.SubItem(fmt.Sprintf("state: %s", connectedState))
		items += style.SubItem(fmt.Sprintf("%s since: %s", connectedState, prettyTime(agent.GetSince().AsTime())))
		items += style.SubItem(fmt.Sprintf("last event: %s", prettyTime(lastActive)))
	}

	return style.SpacedBlock(items)
}
