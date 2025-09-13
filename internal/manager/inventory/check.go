package inventory

func (a *Agents) isCandidate(agent AgentIdentity) (int, bool) {
	for index, candidate := range a.registry.candidates {
		if agent == candidate {
			return index, true
		}
	}
	return -1, false
}

func (a *Agents) isRejected(agent AgentIdentity) (int, bool) {
	for index, rejected := range a.registry.Rejected {
		if agent == rejected {
			return index, true
		}
	}
	return -1, false
}

func (a *Agents) isRegistered(agent AgentIdentity) bool {
	existing, ok := a.registry.Accepted[agent.ID]
	return ok && agent == existing
}

func (a *Agents) IsRegistered(agent AgentIdentity) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	return a.isRegistered(agent)
}
