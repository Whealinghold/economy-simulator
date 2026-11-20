package economysim

// This file is the cover round of the insurance module (spec-insurance.md 5.2, 7; the
// continuation of insurance.go, architecture.md 2.6): the requests and offers of the
// plans, kept for P3.3c, and the scratch the round works in. The round takes no word of
// the stream: every tie is the stateless key K(t, agent) (FR-INS-033).

// coverRequest is a request as a plan returned it, with the agent and its number in the
// plan, kept for the round (the plan is reused by the next agent). store says it is the
// plan's store request and sc is read; otherwise lane is.
type coverRequest struct {
	agent AgentID
	seq   int
	store bool
	lane  LaneCover
	sc    StoreCover
}

// coverOffer is an offer as a plan returned it, with the agent.
type coverOffer struct {
	agent AgentID
	o     CoverOffer
}

// coverRound is the storage of the round: the submitted requests and offers, reused
// every tick.
type coverRound struct {
	reqs []coverRequest
	offs []coverOffer
}
