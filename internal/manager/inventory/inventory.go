package inventory

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/jackadi-io/jackadi/internal/agent"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/serializer"
)

var ErrAgentAlreadyRegistered = errors.New("agent already registered")
var ErrAgentAlreadyCandidate = errors.New("agent already known but not yet accepted")
var ErrAgentNotFound = errors.New("unknown agent")
var ErrAggentRejected = errors.New("rejected agent")

type RogueAgentError struct {
	diffs []diff
}

func (e *RogueAgentError) Error() string {
	return fmt.Sprintf("potential rogue agent detected: diff between rogue and existing agent: %+v", e.diffs)
}

type AgentState struct {
	Connected bool
	Since     time.Time
	LastMsg   time.Time
	specs     map[string]any
}

// IsActive returns true if the agent has been active within the last 60 seconds.
func (a AgentState) IsActive() bool {
	return time.Since(a.LastMsg) <= config.AgentActiveThreshold
}

func NewAgentState() AgentState {
	return AgentState{
		specs: make(map[string]any),
	}
}

type AgentIdentity struct {
	ID          agent.ID
	Address     string
	Certificate string
}

type registry struct {
	Accepted map[agent.ID]AgentIdentity
	States   map[agent.ID]AgentState

	// Agents which have been rejected manually
	Rejected []AgentIdentity

	// candidates are agents which has not been registered yet.
	candidates []AgentIdentity
}

type Agents struct {
	mutex                *sync.Mutex
	registry             registry
	registryPath         string
	registryFileDisabled bool
}

func New() Agents {
	return Agents{
		mutex:                &sync.Mutex{},
		registryPath:         config.RegistryFileName,
		registryFileDisabled: false,
		registry: registry{
			Accepted: make(map[agent.ID]AgentIdentity),
			States:   make(map[agent.ID]AgentState),
		},
	}
}

// LoadRegistry loads the registry file from disk if it exists.
func (a *Agents) LoadRegistry() error {
	if a.registryFileDisabled {
		return nil
	}
	return a.loadRegistryFile()
}

// DisableRegistryFile disables registry file persistence for testing purposes.
func (a *Agents) DisableRegistryFile() {
	a.registryFileDisabled = true
}

func (a *Agents) loadRegistryFile() error {
	data, err := os.ReadFile(a.registryPath)
	if err != nil {
		return err
	}
	return serializer.JSON.Unmarshal(data, &a.registry)
}

func (a *Agents) saveRegistryFile() error {
	if a.registryFileDisabled {
		return nil
	}

	data, err := serializer.JSON.Marshal(a.registry)
	if err != nil {
		return err
	}

	return os.WriteFile(a.registryPath, data, 0600)
}

func (a *Agents) List() ([]AgentIdentity, []AgentIdentity, []AgentIdentity, map[agent.ID]AgentState) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	accepted := make([]AgentIdentity, 0, len(a.registry.Accepted))

	for _, agent := range a.registry.Accepted {
		accepted = append(accepted, agent)
	}
	candidates := slices.Clone(a.registry.candidates)
	rejected := slices.Clone(a.registry.Rejected)

	states := maps.Clone(a.registry.States)

	return accepted, candidates, rejected, states
}

func (a *Agents) AddCandidate(agent AgentIdentity) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if slices.Contains(a.registry.candidates, agent) {
		return ErrAgentAlreadyCandidate
	}

	if _, rejected := a.isRejected(agent); rejected {
		return ErrAggentRejected
	}
	a.registry.candidates = append(a.registry.candidates, agent)
	return nil
}

func (a *Agents) removeCandidate(agent AgentIdentity) error {
	for i, c := range a.registry.candidates {
		if c == agent {
			a.registry.candidates = slices.Delete(a.registry.candidates, i, i+1)
			return nil
		}
	}
	return ErrAgentNotFound
}

// TODO: is it still needed?
func (a *Agents) RemoveCandidate(agent AgentIdentity) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.removeCandidate(agent)
}

func (a *Agents) Register(agent AgentIdentity, allowRejected bool) error {
	slog.Debug("registering agent", "agent", agent.ID)
	a.mutex.Lock()
	defer a.mutex.Unlock()

	existing, ok := a.registry.Accepted[agent.ID]
	if ok {
		diffs := Compare(existing, agent)
		if len(diffs) == 0 {
			return ErrAgentAlreadyRegistered
		}
		return &RogueAgentError{diffs}
	}

	candidateIndex, candidate := a.isCandidate(agent)
	rejectedIndex, rejected := a.isRejected(agent)

	if !candidate && !rejected {
		return ErrAgentNotFound
	}

	if candidate && rejected {
		// in theory it should not happen
		return errors.New("agent is both candidate and rejected")
	}

	if candidate {
		a.registry.candidates = slices.Delete(a.registry.candidates, candidateIndex, candidateIndex+1)
	}

	if rejected {
		if !allowRejected {
			return errors.New("cannot register: agent is rejected")
		}
		a.registry.Rejected = slices.Delete(a.registry.Rejected, rejectedIndex, rejectedIndex+1)
	}

	a.registry.Accepted[agent.ID] = agent
	if err := a.saveRegistryFile(); err != nil {
		slog.Error("unable to permanently register agent", "error", err)
	}

	return nil
}

func (a *Agents) unregister(agent AgentIdentity) error {
	for name, registered := range a.registry.Accepted {
		if registered == agent {
			delete(a.registry.Accepted, name)
			if err := a.saveRegistryFile(); err != nil {
				return fmt.Errorf("unable to permanently remove agent: %w", err)
			}
			return nil
		}
	}

	for name := range a.registry.States {
		if name == agent.ID {
			delete(a.registry.States, name)
			return nil
		}
	}
	return ErrAgentNotFound
}

func (a *Agents) removeStats(agent agent.ID) {
	for name := range a.registry.States {
		if name == agent {
			delete(a.registry.States, name)
			return
		}
	}
}

func (a *Agents) Unregister(agent AgentIdentity) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.unregister(agent)
}

func (a *Agents) Reject(agent AgentIdentity) error {
	slog.Debug("reject request received", "agent", agent.ID)
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if ok := a.isRegistered(agent); ok {
		if err := a.unregister(agent); err != nil {
			return fmt.Errorf("reject did not unregistered: %w", err)
		}
	}

	if indexToDelete, ok := a.isCandidate(agent); ok {
		a.registry.candidates = slices.Delete(a.registry.candidates, indexToDelete, indexToDelete+1)
	}

	a.registry.Rejected = append(a.registry.Rejected, agent)

	slog.Debug("agent rejected", "agent", agent.ID)
	if err := a.saveRegistryFile(); err != nil {
		return fmt.Errorf("unable to permanently reject agent: %w", err)
	}
	return nil
}

func (a *Agents) Remove(agent AgentIdentity) error {
	slog.Debug("remove request received", "agent", agent.ID)
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if registered := a.isRegistered(agent); registered {
		if err := a.unregister(agent); err != nil {
			return fmt.Errorf("remove did not unregistered: %w", err)
		}
	}

	if candidateIndex, candidate := a.isCandidate(agent); candidate {
		a.registry.candidates = slices.Delete(a.registry.candidates, candidateIndex, candidateIndex+1)
	}

	if rejectedIndex, rejected := a.isRejected(agent); rejected {
		a.registry.Rejected = slices.Delete(a.registry.Rejected, rejectedIndex, rejectedIndex+1)
	}

	a.removeStats(agent.ID)

	slog.Debug("agent removed", "agent", agent.ID)
	if err := a.saveRegistryFile(); err != nil {
		return fmt.Errorf("unable to permanently remove agent: %w", err)
	}
	return nil
}

func (a *Agents) MarkAgentActive(agent agent.ID) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	agt, ok := a.registry.States[agent]
	if !ok {
		a.registry.States[agent] = NewAgentState()
	}

	agt.LastMsg = time.Now()
	a.registry.States[agent] = agt
}

func (a *Agents) MarkAgentStateChange(agent agent.ID, connected bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	agt, ok := a.registry.States[agent]
	if !ok {
		a.registry.States[agent] = NewAgentState()
	}

	agt.Connected = connected
	agt.Since = time.Now()
	a.registry.States[agent] = agt
}

func (a *Agents) GetSpec(agent agent.ID) map[string]any {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	agt, ok := a.registry.States[agent]
	if !ok {
		return nil
	}

	return agt.specs
}

func (a *Agents) GetAllSpecs() map[agent.ID]map[string]any {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	out := make(map[agent.ID]map[string]any, len(a.registry.States))
	for k, v := range a.registry.States {
		out[k] = maps.Clone(v.specs)
	}

	return out
}

func (a *Agents) SetSpec(agent agent.ID, specs map[string]any) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if specs == nil {
		return nil
	}

	agt, ok := a.registry.States[agent]
	if !ok {
		return fmt.Errorf("agent not connected: %s %p", string(agent), a)
	}

	agt.specs = specs
	a.registry.States[agent] = agt

	return nil
}
