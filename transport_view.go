package economysim

// This file is the view side of the transport module (spec-transport.md 9): the
// tables fixed at P3.1, the filler of the AgentView, the routes, the queries and
// the transit-factor slot.

// p31Fix is P3.1: the shipments in flight, which fix Inbound and Outbound for every
// agent of the tick before any agent is called, so that they do not depend on the
// order of agents (spec-transport.md 5.3.3, 9.1). Nothing is drawn.
func (w *World) p31Fix() {}
