// Package room holds the pure rules of the storage module over primitive
// values (spec-storage.md 3.3.2, 5.3.2, 5.5.2.1, 6.2.2, 6.3.2, 8.3.2): the room
// of an account, the number of executions a room allows, the scaling of the
// targets to a room, the order in which goods go when a load must fall, and the
// stored load. It knows no commodity, role, agent or store: it receives sizes
// and quantities as numbers, and the root converts at the call (architecture.md
// 2.9). It imports the standard library and internal/checked only.
//
// A mass count is never compared with a bulk count (spec-goods.md 3.2): the
// pair is measured dimension by dimension against its own room.
package room
