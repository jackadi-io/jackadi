package inventory

type diff struct {
	Key      string
	Expected any
	Got      any
}

func Compare(node1, node2 NodeIdentity) []diff {
	diffs := []diff{}
	if node1.ID != node2.ID {
		diffs = append(diffs, diff{"ID", node1.ID, node2.ID})
	}
	if node1.Address != node2.Address {
		diffs = append(diffs, diff{"address", node1.Address, node2.Address})
	}
	if node1.Certificate != node2.Certificate {
		diffs = append(diffs, diff{"certificate", "hidden", "hidden"}) // it would make the output unreadable
	}

	return diffs
}
