package inventory

import "github.com/jackadi-io/jackadi/internal/agent"

func (a *Agents) GetMatchingAccepted(agentID agent.ID, address, certificate *string) []AgentIdentity {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	accepted := []AgentIdentity{}
	for _, candidate := range a.registry.Accepted {
		if candidate.ID != agentID {
			continue
		}
		if address != nil && candidate.Address != *address {
			continue
		}
		if certificate != nil && candidate.Certificate != *certificate {
			continue
		}
		accepted = append(accepted, candidate)
	}

	return accepted
}
func (a *Agents) GetMatchingCandidates(agentID agent.ID, address, certificate *string) []AgentIdentity {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	candidates := []AgentIdentity{}
	for _, candidate := range a.registry.candidates {
		if candidate.ID != agentID {
			continue
		}
		if address != nil && candidate.Address != *address {
			continue
		}
		if certificate != nil && candidate.Certificate != *certificate {
			continue
		}
		candidates = append(candidates, candidate)
	}

	return candidates
}

func (a *Agents) GetMatchingRejected(agentID agent.ID, address, certificate *string) []AgentIdentity {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	rejected := []AgentIdentity{}
	for _, candidate := range a.registry.Rejected {
		if candidate.ID != agentID {
			continue
		}
		if address != nil && candidate.Address != *address {
			continue
		}
		if certificate != nil && candidate.Certificate != *certificate {
			continue
		}
		rejected = append(rejected, candidate)
	}

	return rejected
}
