package inventory

func (n *Nodes) isCandidate(nd NodeIdentity) (int, bool) {
	for index, candidate := range n.registry.candidates {
		if nd == candidate {
			return index, true
		}
	}
	return -1, false
}

func (n *Nodes) isRejected(nd NodeIdentity) (int, bool) {
	for index, rejected := range n.registry.Rejected {
		if nd == rejected {
			return index, true
		}
	}
	return -1, false
}

func (n *Nodes) isRegistered(nd NodeIdentity) bool {
	existing, ok := n.registry.Accepted[nd.ID]
	return ok && nd == existing
}

func (n *Nodes) IsRegistered(nd NodeIdentity) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	return n.isRegistered(nd)
}
