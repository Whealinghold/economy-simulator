package economysim

// This file is the test-only seam of scale_pack_test.go (architecture.md 10.2).

// ScaleState counts the private state of a world that must not depend on the
// weights of its agents (spec-scale.md 5.1, FR-SCL-016, NFR-SCL-004).
type ScaleState struct {
	// Agents is the slots of the agent table, Accounts the ledger's accounts,
	// Beliefs the price beliefs of all agents, Residues their output residues and
	// WindowSlots the slots of their welfare rings.
	Agents, Accounts, Beliefs, Residues, WindowSlots int
}

// ScaleStateOf counts the private state of w.
func ScaleStateOf(w *World) ScaleState {
	s := ScaleState{Agents: len(w.agents), Accounts: w.ledger.Len()}
	for i := range w.agents {
		a := &w.agents[i]
		s.Beliefs += len(a.beliefs)
		s.Residues += len(a.residues)
		s.WindowSlots += int(a.history.Window())
	}
	return s
}
