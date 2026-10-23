package economysim

import "github.com/Whealinghold/economy-sim/internal/codec"

// This file is the section table of the checkpoint (spec-checkpoint.md 4.2, 5.3;
// plan 2-2 D243). A module appends its section by the protocol of architecture.md
// 2.10.1 and 2.10.5: it writes saveX in its own file over internal/codec with the
// layout of 4.2 for its tag, appends one row to checkpointSections at its tag
// position, deletes its tag from notBuiltSections, and adds its cross-checks and
// its tests; nothing is renumbered, because the tag order is the write order and
// the format depends on it.

// section is one row of the table of spec-checkpoint.md 4.2: the tag, the name of
// the section in that table, when it is present and how it is written. present is
// nil for a section that is always there (tags 1 to 11 and 34).
type section struct {
	tag     uint64
	name    string
	present func(*Config) bool
	save    func(*World, *codec.Writer) error
}

// checkpointSections holds the sections the code writes, in tag order. A test
// compares it, with notBuiltSections, to the table of the spec by a plain text
// parse. It is never written after the package is initialised.
var checkpointSections = [...]section{
	{tag: 1, name: "World", save: (*World).saveWorld},
	{tag: 2, name: "Ledger", save: (*World).saveLedger},
	{tag: 3, name: "Agents", save: (*World).saveAgents},
	{tag: 4, name: "Markets", save: (*World).saveMarkets},
	{tag: 5, name: "Beliefs", save: (*World).saveBeliefs},
	{tag: 6, name: "Production", save: (*World).saveJobs},
	{tag: 7, name: "Welfare", save: (*World).saveWelfare},
	{tag: 8, name: "Estates", save: (*World).saveEstates},
	{tag: 9, name: "Spawner", save: (*World).saveSpawner},
	{tag: 10, name: "Skills", save: (*World).saveSkills},
	{tag: 11, name: "Regions", save: (*World).saveRegions},
	{tag: 34, name: "Extensions", save: (*World).saveExtensions},
}

// notBuiltSection is a row of the table of 4.2 that has no writer yet, with the
// module that gates it (the field of Config.Modules).
type notBuiltSection struct {
	tag  uint64
	name string
	gate string
}

// notBuiltSections names the sections 12 to 33, which the modules that own them
// append. The test fails on a listed tag that has a row in checkpointSections and
// on a tag of the spec that is neither built nor listed.
var notBuiltSections = [...]notBuiltSection{
	{12, "Demand", "Demand"}, {13, "MoneySupply", "MoneySupply"}, {14, "Population", "Population"},
	{15, "Resources", "Resources"}, {16, "Information", "Information"}, {17, "Transport", "Transport"},
	{18, "External", "External"}, {19, "Subsistence", "Subsistence"}, {20, "Migration", "Migration"},
	{21, "Merchants", "Merchants"}, {22, "Integration", "Information"}, {23, "Settlements", "Settlements"},
	{24, "Employment", "Employment"}, {25, "Tenure", "Tenure"}, {26, "Credit", "Credit"},
	{27, "Facilities", "Infrastructure"}, {28, "Risk", "Risk"}, {29, "Insurance", "Insurance"},
	{30, "Fiscal", "Fiscal"}, {31, "Institutions", "Institutions"}, {32, "Policy", "Policy"},
	{33, "Technology", "Technology"},
}

// agentTail is a block that follows the agent list of tag 3 when its module is
// on: the stores of the storage module and the membership of the institutions
// module (spec-checkpoint.md 4.2, tag 3). save writes the block of the agents in
// ascending ID.
type agentTail struct {
	present func(*Config) bool
	save    func(*World, *codec.Writer)
}

// agentTails holds the blocks of tag 3 after the agent list, in module order. It
// is empty until the storage and institutions modules are built, which append to
// it.
var agentTails = [...]agentTail{}
