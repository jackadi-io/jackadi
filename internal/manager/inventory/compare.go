package inventory

type diff struct {
	Key      string
	Expected any
	Got      any
}

func Compare(agent1, agent2 AgentIdentity) []diff {
	diffs := []diff{}
	if agent1.ID != agent2.ID {
		diffs = append(diffs, diff{"ID", agent1.ID, agent2.ID})
	}
	if agent1.Address != agent2.Address {
		diffs = append(diffs, diff{"address", agent1.Address, agent2.Address})
	}
	if agent1.Certificate != agent2.Certificate {
		diffs = append(diffs, diff{"certificate", "hidden", "hidden"}) // it would make the output unreadable
	}

	return diffs
}
