package economysim

// This file is the storage module (spec-storage.md; architecture.md 2.10.1): what
// the module reads of its configuration. The validator and the warnings are in
// storage_validate.go. The module has no state, no position and no event yet: it is
// configured and validated and does nothing (plan 3-sto Phase 3); its account
// rules are in internal/room and internal/ledger. Outside this file and its
// continuation the module appears only in the catalogue files (config.go, hash.go,
// bounds.go, defaults.go, errors.go, validate.go), in value.go (StoreClassID), in
// role.go (the resolved base room of a role) and at the named hooks of ADR 0010.

// baseOf is the resolved base room per household of a role (spec-storage.md 3.3.1):
// the Base of the role's row when it has one, else the module's Base; nil is an
// unlimited role. The second result is the index of the row, or -1 when the
// module's Base is the answer. It is nil, -1 for a nil module (off: no role is
// limited).
func (m *Storage) baseOf(role string) (*Capacity, int) {
	if m == nil {
		return nil, -1
	}
	for i := range m.Roles {
		if m.Roles[i].Role == role {
			return m.Roles[i].Base, i
		}
	}
	return m.Base, -1
}

// classSet is the IDs of the classes of the module, a membership test only.
func (m *Storage) classSet() map[string]bool {
	set := make(map[string]bool, len(m.Classes))
	for _, c := range m.Classes {
		set[c.ID] = true
	}
	return set
}

// roomOf is a copy of the resolved base room of a role, nil for an unlimited role
// and for the module off: the role table holds a value of its own, so a change of
// the consumer's Config after construction moves no room.
func (m *Storage) roomOf(role string) *Capacity {
	b, _ := m.baseOf(role)
	if b == nil {
		return nil
	}
	room := *b
	return &room
}
