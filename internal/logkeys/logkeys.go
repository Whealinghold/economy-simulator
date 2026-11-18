// Package logkeys holds the attribute keys of the library's log lines, defined
// once so that a key is spelled one way (observability-and-logging.md 3.3.2).
//
// Only the keys that the lines of the first slice use are here; a key is added
// in the change that first logs it, together with its row in the table of
// 3.3.2. The package holds constants only and imports nothing.
package logkeys

// The keys, in the order of the table of observability-and-logging.md 3.3.2.
const (
	// Tick is the simulation tick.
	Tick = "tick"
	// Ticks is the number of ticks requested or completed by a Run.
	Ticks = "ticks"
	// Seed is the random seed.
	Seed = "seed"
	// Commodity is a commodity ID.
	Commodity = "commodity"
	// Role is a role ID.
	Role = "role"
	// Recipe is a recipe ID.
	Recipe = "recipe"
	// Agent is an agent ID.
	Agent = "agent"
	// Price is a price in minor units.
	Price = "price"
	// Quantity is a quantity in whole units.
	Quantity = "quantity"
	// Side is bid or ask.
	Side = "side"
	// Reason is a machine-readable reason.
	Reason = "reason"
	// Error is an error value, logged with slog.Any.
	Error = "error"
	// Region is a region ID.
	Region = "region"
	// Bound is the bound a price belief sits on, floor or ceiling.
	Bound = "bound"
	// Agents is a count of agents.
	Agents = "agents"
	// Households is a count of households, the sum of agent weights.
	Households = "households"
	// Unfed is a count of households that went unfed.
	Unfed = "unfed"
	// Path is the path of a configuration field.
	Path = "path"
	// Rule is the name of a configuration validation rule.
	Rule = "rule"
	// ConfigHash is the first 8 bytes of the configuration hash.
	ConfigHash = "config_hash"
	// Prices is a count of price rows of the warm-start table.
	Prices = "prices"
	// Wages is a count of wage rows of the warm-start table.
	Wages = "wages"
	// Groups is a count of group rows of the warm-start table.
	Groups = "groups"
	// Jobs is a count of job rows of the warm-start table.
	Jobs = "jobs"
	// Cause is the machine-readable cause of a removal or a change.
	Cause = "cause"
	// Want is a want ID.
	Want = "want"
	// Commodities is a count of commodities in the construction summary.
	Commodities = "commodities"
	// Recipes is a count of recipes in the construction summary.
	Recipes = "recipes"
	// Roles is a count of roles in the construction summary.
	Roles = "roles"
	// Regions is a count of regions in the construction summary.
	Regions = "regions"
	// Edges is a count of directed edges of the region graph in the construction
	// summary.
	Edges = "edges"
	// Components is a count of strongly connected components of the region graph
	// in the construction summary; 0 when the world has one region.
	Components = "components"
	// FromRegion and ToRegion are the region IDs of the two ends of a relocation
	// (Region stays the key for a single region).
	FromRegion = "from_region"
	ToRegion   = "to_region"
	// Distance is an edge length, int64, in the pack's distance unit.
	Distance = "distance"
	// Closed is whether an edge is closed, a bool.
	Closed = "closed"
	// FactorBP is an edge length factor in basis points, int64.
	FactorBP = "factor_bp"
	// Modules is the names of the modules that are on.
	Modules = "modules"
	// WarmStart is whether a warm start was present, a bool.
	WarmStart = "warm_start"
	// Account is a ledger account ID.
	Account = "account"
	// Amount is a sum of money in minor units.
	Amount = "amount"
	// Outcomes is a count of order outcomes, the denominator of the belief
	// Warn.
	Outcomes = "outcomes"
	// Previous is a price in minor units before an update; Price is the one
	// after.
	Previous = "previous"
	// Format is the checkpoint format version, logged as slog.Int64(int64(f)).
	Format = "format"
	// Bytes is the size of an encoded checkpoint, in bytes.
	Bytes = "bytes"
	// Flow is a money flow ID, also the reason of its boundary rows.
	Flow = "flow"
	// Due and Applied are money in minor units: the amount a money flow computed
	// for a recipient or region, and the amount actually applied.
	Due     = "due"
	Applied = "applied"
	// Births and Deaths are counts of households born and of households that died.
	Births = "births"
	Deaths = "deaths"
	// From and To are a value before and after a change, in basis points
	// (FromRegion and ToRegion stay the keys of two regions).
	From = "from"
	To   = "to"
	// Resource is a resource ID; Stock the units in an endowment; Short the units
	// asked of an endowment and not granted.
	Resource = "resource"
	Stock    = "stock"
	Short    = "short"
	// OwnUse is the households in the own-use mode of the subsistence module, the
	// sum of agent weights (int64).
	OwnUse = "own_use"
	// Shipment is a shipment ID and Mode a transport mode ID; Dest is the
	// destination region ID of a shipment (Region is the origin), and Modes a count
	// of transport modes in the construction summary.
	Shipment = "shipment"
	Mode     = "mode"
	Dest     = "dest"
	Modes    = "modes"
	// Pairs is a count of ordered region pairs, carried by the warning
	// rumour_horizon_short.
	Pairs = "pairs"
	// Rumours is the count of rumours offered to one agent, and Lag a lag in ticks
	// (int64), in the "rumours served" line of the information module.
	Rumours = "rumours"
	Lag     = "lag"
)

// Contract is a delivery contract ID, Offers and Takes the counts of admitted
// delivery offers and takes of a round, and Merchants a count of agents with a
// merchant block, in the lines of the merchants module.
const (
	Contract  = "contract"
	Offers    = "offers"
	Takes     = "takes"
	Merchants = "merchants"
)

// Node is the region ID of an outside node (Region stays the key of the gate), Value
// money in minor units (int64), as the money paid or received in the trades of an
// outside side, Nodes and Counterparties counts in the construction summary, and
// Drift the drift of an outside price walk in basis points, in the lines of the
// external module.
const (
	Node           = "node"
	Value          = "value"
	Nodes          = "nodes"
	Counterparties = "counterparties"
	Drift          = "drift"
)

// Jurisdictions and Governed are counts of jurisdictions and of governed
// jurisdictions in the construction summary, in the line of the polity module.
const (
	Jurisdictions = "jurisdictions"
	Governed      = "governed"
)

// The keys of the settlements module (spec-settlements.md 10.3): a class (an
// int64 SettlementClassID) and the two ends and the target of a change, and a count
// of classes in the construction summary.
const (
	SettlementClass       = "settlement_class"
	FromSettlementClass   = "from_settlement_class"
	ToSettlementClass     = "to_settlement_class"
	TargetSettlementClass = "target_settlement_class"
	SettlementClasses     = "settlement_classes"
)

// StoreClass is a store class ID, and StoreClasses a count of store classes in the
// construction summary, in the lines of the storage module.
const (
	StoreClass   = "store_class"
	StoreClasses = "store_classes"
)

// Trainer is the agent ID of a trainee's trainer, and Served the ticks of a
// traineeship served, in the line "traineeship ended" of the skill barriers.
const (
	Trainer = "trainer"
	Served  = "served"
)

// Field is a technology field ID, Fields a count of technology fields in the
// construction summary, and Gain a technology gain in basis points, in the lines
// of the technology module.
const (
	Field  = "field"
	Fields = "fields"
	Gain   = "gain"
)

// The keys of the credit module (spec-credit.md 16.2): a claim ID and a count of
// claims, a yearly rate in basis points and a term in ticks (int64), money in
// minor units (int64) for the principal of a claim, the amount owed, the owed on
// overdue claims and the amount lost, and counts of lenders and of borrowers in the
// construction summary. The key amount stays the line of a refused transfer.
const (
	Claim     = "claim"
	Claims    = "claims"
	Rate      = "rate"
	Term      = "term"
	Principal = "principal"
	Owed      = "owed"
	Arrears   = "arrears"
	Lost      = "lost"
	Lenders   = "lenders"
	Borrowers = "borrowers"
)

// Rent is money per unit of an endowment, in minor units (int64), and Share a share
// in basis points (int64), as the share of a title or of a strike, in the lines of
// the tenure module.
const (
	Rent  = "rent"
	Share = "share"
)

// Rows is a count of risk rows (route and store rows), in the construction summary
// of the risk module (spec-risk.md 14.3).
const Rows = "rows"

// The keys of the institutions module (spec-institutions.md 11.3): the ID of a body, of
// a right and of a control, the activity a right gates, the places of a body and the
// households waiting for one, and the counts of bodies, rights and controls in the
// construction summary. The reason value not_member is a log-only reason of this
// module, in the rule table's row for reason.
const (
	Body     = "body"
	Right    = "right"
	Control  = "control"
	Activity = "activity"
	Places   = "places"
	Waiting  = "waiting"
	Bodies   = "bodies"
	Rights   = "rights"
	Controls = "controls"
)
