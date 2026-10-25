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
)
