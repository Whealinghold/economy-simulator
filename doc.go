// Package economysim simulates a market economy with configurable supply
// chains, in the style of bazaarBot. It is a library: a consumer (a game loop,
// a server, a command) builds a configuration, creates a world and steps it.
//
// The module path is github.com/Whealinghold/economy-sim, whose last element is
// not a valid identifier, so the package name differs from it. Import it with
// an explicit name:
//
//	import economysim "github.com/Whealinghold/economy-sim"
//
// # Principles
//
// Commodities, recipes and roles are data, never code. The library is
// deterministic: a seed, a configuration and a sequence of calls give
// bit-identical results. Money, prices and quantities are int64 fixed-point
// values, and arithmetic that can overflow is checked, never wrapped. The
// library has no goroutines, no global state and no output unless a logger or
// an observer is injected.
//
// # Glossary
//
// A new domain term is added here before code uses it (architecture.md 3.2).
// The terms below are the ones the code of the foundations uses; the fuller
// list is in doc/specs/README.md.
//
//   - Account: a slot of the ledger that holds money, and goods as lots, and has
//     a kind, an owner reference and a region. IDs are assigned in creation
//     order and never reused; a closed account keeps its slot.
//   - Ageing: the end-of-tick pass of the ledger that moves every expired lot
//     whole to the spoilage boundary; it visits only accounts whose expiry
//     hint has been reached.
//   - Agent: a participant of the market that holds an account, takes a role and
//     trades. Identified by an AgentID.
//   - Audit: an option (WithAudit) that makes a world check its invariants as it
//     runs; it changes no result and is not part of the configuration hash.
//   - Balance sheet: a read-only view of a non-boundary account: its money, its
//     goods valued at prices the caller supplies, the claims it holds and the
//     claims against it, and the net worth. It is measurement only; no rule of
//     the engine reads one.
//   - Basis point: one ten-thousandth. Ratios, rates and factors are integers in
//     basis points and are applied as v * bp / 10 000.
//   - Boundary flow: money or goods that enter the economy (a faucet) or leave
//     it (a sink) through a ledger reason; BoundaryFlow reports the total per
//     tick, region and reason.
//   - Boundary account: the ledger account of one reason and one thing (money
//     or a commodity) that stands for everything outside the economy's
//     conservation. It is the only kind that may go negative, and its balance
//     is the net amount of that reason that has entered or left.
//   - Born tick: the tick a lot of a spoiling commodity was made; its age is
//     measured from it. It is 0 for a commodity that does not spoil, and
//     negative only for a lot created at construction.
//   - Bound: a constant of bounds.go that a configuration value is checked
//     against, so that arithmetic cannot overflow and validation work is
//     bounded. A bound is not a tunable.
//   - Bucket: a value in [0, P) derived from an agent's ID by the second Weyl
//     value, which places the agent on a tick of a rotation of period P.
//   - Construction bucket: the flow totals of what is moved while a world is
//     being built, kept apart from the totals of tick 0. It is read-only and
//     stays readable for the run.
//   - Construction phase: the time from the first line of NewWorld to the
//     start of the first step, in which the ledger's current tick is 0 and
//     its moves are totalled in the construction bucket.
//   - Calendar: the structure of time in a configuration: hours, days, months,
//     the length of a tick and the seasons. Earth by default.
//   - Absolute day: the calendar day of a tick counted from the first day of
//     the year in which tick 0 falls, StartDay plus the days elapsed; its year
//     and day of year are its quotient and remainder by the days of a year.
//   - Calendar day: the calendar's unit of time, hours per day of it; a tick
//     may be a part of one or a number of them, and is a calendar day only
//     when TicksPerDay and DaysPerTick are both 1.
//   - Moment: the value Calendar.At returns for a tick: year, day of year,
//     month, day of month, weekday, tick of day, hour, season and day of
//     season. It is derived from the calendar and the tick and never stored.
//   - Tick of day: the zero-based position of a tick within its calendar day,
//     0 unless TicksPerDay is above 1.
//   - Ticks per year: the ticks in a year, derived as the days of the year times
//     TicksPerDay over DaysPerTick, which must divide exactly; it is never
//     configured.
//   - Canonical order: the order of the entities of a configuration after
//     sorting by ID, which fixes the typed IDs and the configuration hash
//     whatever order the entities were written in.
//   - Cause: a member of a closed set (Cause) that names why an agent was removed,
//     reduced or changed role, such as insolvent or starved.
//   - Claim: the ledger's record of an amount one account owes another: a
//     creditor, a debtor, a principal and an interest, with what is still
//     owed. A claim with nothing owed does not exist.
//   - Commodity: a good that can be held and traded, identified by a CommodityID.
//     NoCommodity stands for money where a commodity field is also used for it.
//   - Configuration: the Config value that defines a world; plain data that a
//     file loader or Go code builds.
//   - Configuration hash: the SHA-256 of the canonical encoding of a Config,
//     which a checkpoint stores and a restore compares.
//   - Creditor: the account a claim is owed to, an agent or an estate account.
//   - Debtor: the agent account that owes a claim.
//   - Default constructor: DefaultWorld, DefaultRole and the like, the one
//     place a default is defined. The zero value of an entity is not a default.
//   - Draw site: a place in the code where the world takes a random word, named
//     by a constant that never changes and keyed by (site, tick, id).
//   - Earmark: the units of a lot that a goods reservation has set aside; an
//     unmarked move cannot take them and a move from reserved consumes them.
//   - Effective seed: the seed a world draws from after any reseed; it is the one
//     value of the random state that a checkpoint carries.
//   - Configuration section: a struct of config.go that Config holds, one per
//     part of the file: World, Calendar, Month, Season, Commodity, Item, Output,
//     Tool, Recipe, YieldProfile, SeasonYield, Role, Region, Group, WarmStart,
//     SeedPrice, SeedWage, WarmGroup, JobSpec and WarmPool.
//   - Durable: a commodity that wears; its items are held as lots that share a
//     remaining durability, and a move takes the most worn first.
//   - Estate: the goods and money an agent leaves behind when it is removed; the
//     money goes to the regional estate pool, the goods are listed for sale.
//   - Mass: the load-bearing weight of one unit of a commodity, a commodity
//     attribute; with bulk it makes a load.
//   - Bulk: the room one unit of a commodity takes up, a commodity attribute;
//     with mass it makes a load.
//   - Load: the pair of the sum of quantity times mass and the sum of quantity
//     times bulk over a set of goods, computed with checked arithmetic.
//   - Container: anything with a capacity, itself a pair of a mass capacity and
//     a bulk capacity; a load fits it if and only if both sums are within them.
//   - Chargeable size: the one number a price per unit would use for a unit of a
//     commodity, the greater of its mass and its bulk times BulkFactor over
//     10 000, rounded up. No module of version 1 reads it.
//   - Chargeable capacity: the number of chargeable units of any single
//     commodity a container is sure to carry, the smaller of its mass capacity
//     and its bulk capacity times BulkFactor over 10 000, rounded down.
//   - Estate pool: the money of a region's estates, which can fund the starting
//     money of a new agent; WarmPool seeds it.
//   - Event: a plain value that a world emits to its observers to report a
//     transition; the closed set of event types is sealed to this package.
//   - Event type: one of the sealed struct types of event.go that implement
//     Event: WorldCreated, RunStarted, RunEnded, TickStarted, TickEnded,
//     OrderPlaced, OrderRejected, TradeExecuted, MarketCleared, BeliefsUpdated,
//     RecipeExecuted, RecipeAbandoned, ProductionHalted, UpkeepShortfall,
//     AgentSpawned, AgentBankrupted, EstateOpened, WeightChanged, RoleChanged,
//     EntryLimited and BoundaryFlow. The first field of each is Tick.
//   - Faucet: a boundary flow that brings money or goods into the economy
//     from outside; its Kind is KindFaucet.
//   - Field id: the position of a field in its struct, counted from 1, under
//     which the configuration hash writes it. An id is never reused or moved.
//   - Finaliser: the SplitMix64 output function, a bijection of uint64 that mixes
//     a word so that near inputs give unrelated outputs (ordering.Finalise).
//   - Flow totals: the ledger's per-tick total moved for each region, reason
//     and money or commodity, in plain integer slots cleared at the end of
//     the tick.
//   - Fixed point: money, prices and quantities are whole numbers of their
//     smallest unit; no floating point affects a result.
//   - Group: the households of one role in one region, with an optional scale.
//   - Guard pass: the counting-only parse of a document in config/ that checks
//     nesting depth and keys per table before any tree is built.
//   - Household: the unit of population; an agent holds a weight of them.
//   - Keyed stream: the random stream of one draw site, restarted from the
//     seed and the key (site, tick, id) before each use, so that no draw depends
//     on the order of others.
//   - Kind: the kind of a boundary flow, a closed set with two members, faucet
//     and sink.
//   - Log key: the name of an attribute of a log line, defined once in
//     internal/logkeys and listed in observability-and-logging.md 3.3.2.
//   - Lot: goods of one commodity in an account that share a born tick and a
//     remaining durability; equal lots are merged and an account's lots are
//     kept sorted by commodity, born tick and durability (canonical form).
//   - Minor unit: the smallest unit of money. Money and Price are counts of
//     minor units; the library has no currency, denomination or decimal place.
//   - Module: an optional mechanism that is off unless its entry in
//     Config.Modules is non-nil.
//   - Ledger: the single holder of all money and goods in a world, which
//     changes them only by transfers; its rules live in internal/ledger.
//   - Money: an amount of money, an int64 count of minor units.
//   - Names table: the Names value that maps each typed ID of a world to the
//     string ID of its configuration, carried by events and snapshots.
//   - Named lot: a lot addressed by its key (commodity, born tick, remaining
//     durability) in a reservation or a move, as the estate prices each lot.
//   - Observer: the interface through which a consumer receives events; it is
//     optional and discards by default. warnRun is the run length of a
//     bounded warning.
//   - Pack: a set of configuration documents, layered in order, that defines a
//     supply chain; later documents add to or override earlier ones.
//   - Presentation-only field: a field, such as a label, that changes no result
//     and is not part of the configuration hash.
//   - Price: money per unit of a commodity, an int64 count of minor units.
//   - Producible: a commodity that is sourced, or that some recipe makes from
//     inputs and tools that are all producible; validation rejects a
//     commodity that something consumes and that is not producible.
//   - Pool policy: what the estates module switches on: the release of the
//     estate pool as relief and its use as a money-flow target.
//   - Quantity: a count of whole units of a commodity.
//   - Reason: a machine-readable member of a closed set (Reason) that names why
//     an order was rejected, a job ended without output or a flow crossed the
//     boundary.
//   - Reservation: money, or goods, earmarked in an account so that an order
//     placed now can be settled later; it exists only inside a tick.
//   - Recipe: a rule that turns inputs into outputs over some ticks, identified
//     by a RecipeID.
//   - Region: a place with its own markets, identified by a RegionID. NoRegion
//     stands for no region.
//   - Reseed: the pure function that derives a new effective seed from a seed
//     and a salt, as a restore does; it installs nothing.
//   - Role: the occupation of an agent, which fixes the recipes it may run and
//     what it needs, identified by a RoleID.
//   - Rotation: a cadence that selects an agent on the ticks where
//     (tick + bucket) mod P is 0, so a group of agents is spread over a period.
//   - Schema version: the integer under the key schema in a configuration file
//     that names the version of the file format; the loader accepts one.
//   - Season: a named part of the year, in an ordered list of the calendar.
//   - Row type: a small struct that an event carries in a slice or a field,
//     such as Amount (a quantity of a commodity) and HaltRow (a recipe a
//     production halt skipped and the households it was skipped for).
//   - Seed: the value from which every random word of a world derives; with a
//     configuration and a sequence of calls it fixes a run.
//   - Site constant: the fixed value that names a draw site (siteWork and the
//     like). It never changes and is spaced at least 2^32 from every other.
//   - Side: the side of an order, bid (buys) or ask (sells).
//   - Stream key: the (seed, site, tick, id[, sub]) that fixes the keyed stream
//     of one draw; two equal keys give the same words.
//   - Sink: a boundary flow that takes money or goods out of the economy; its
//     Kind is KindSink.
//   - Source: a recipe with no inputs (spec-production.md 3.1.2); it makes
//     its outputs from nothing but time and tools. A recipe table row says so.
//   - Maker: a recipe that has a commodity among its non-discard outputs; the
//     recipe table lists the makers of each commodity in ascending RecipeID.
//   - Recipe table: the recipes of a configuration in typed-ID form, built once
//     at construction with the makers index; it is derived, not state.
//   - Role table: the roles of a configuration in typed-ID form, built once at
//     construction: recipes, trades, base upkeep, tradable set and the
//     effective upkeep per region. It is derived, not state.
//   - Tradable set: the commodities a role may trade (spec-agents.md 3.4): its
//     effective upkeep in any region, the inputs, tools and non-discard outputs
//     of its recipes, and its Trades; ascending, with no duplicate.
//   - Effective upkeep: what one household of a role needs each tick: the role's
//     list or the world's, merged with a region's extra upkeep by adding
//     quantities of one commodity (spec-agents.md 5.2).
//   - Sourced: a commodity the producibility check takes as supplied from
//     outside the recipe graph: one marked Raw. A recipe with no inputs is a
//     source in the same sense.
//   - Target spread: the spread, in basis points, of the target ratio of a role
//     around its centre; WithTargetSpread overrides it for a whole world.
//   - Stated lot key: the born tick and durability a creation under the
//     initial reason names itself, allowed only while a world is being built.
//   - Expiry hint: the earliest tick at which any lot of an account expires,
//     derived from its lots and recomputed after a restore, never saved.
//   - Transform: the ledger operation of one recipe execution: inputs leave an
//     account to the recipe_input boundary and outputs enter from the
//     recipe_output boundary, all or nothing.
//   - Wear units: the durability removed by wear, counted by commodity in the
//     ledger's per-tick totals, which an item destroyed counts at what it had.
//     WearApplied reports them once per tick and commodity; BoundaryFlow under
//     the wear reason counts only the items retired.
//   - Tick: the unit of simulated time; there is no wall-clock time in the
//     library.
//   - Transfer: the one operation that changes a balance: an amount moves from
//     one account to another under a reason, refused whole or applied whole.
//   - Trainee: an agent that learns a role under a trainer for a number of
//     ticks; a recipe may be trainee-eligible.
//   - Upkeep: the goods one household must consume each tick to stay alive.
//   - Warning: a configuration that is legal but probably a mistake; Warnings
//     returns it as a value with a path and a rule and it never fails
//     validation.
//   - Warm start: an optional table that seeds prices, wages, money, stock and
//     jobs so that a world begins near a steady state.
//   - Weyl value: a fixed per-agent spread, the ID multiplied by an odd constant
//     modulo 2^32. It draws no random number. Weyl32 is used for carries and
//     margins and Weyl2, a second sequence, for the bucket of a rotation.
//   - Word: one uint64 taken from a keyed stream; a draw of n values uses one word.
//   - World: the simulation that a configuration defines and a consumer steps.
//     In Config, World is the section of world-level parameters, such as the
//     scale of a group and the bounds of its agents.
//   - Yield profile: the factor, by season, of a recipe in a region.
//   - Market: the book of one commodity in one region for one tick: the
//     admitted bids and asks, cleared once per tick.
//   - Order: one agent's offer in a market, a side, a limit price and a
//     quantity, made for one tick and, for an ask, optionally of a named lot.
//   - Bid: an order to buy up to a quantity at a limit price or below.
//   - Ask: an order to sell up to a quantity at a limit price or above.
//   - Limit price: the worst price an order accepts: the most a bid pays and
//     the least an ask takes.
//   - Trade: a quantity of a commodity that one buyer and one seller exchange
//     at one price, within both limits.
//   - Admission: the ordered checks that an order passes before it enters a
//     book; the first failing check is its rejection reason.
//   - Available: the units or money of an account that no reservation holds.
//   - Tie-break key: the word drawn for an order in a market that orders equal
//     limit prices; the input position ends the order.
//   - Crossing: a bid whose limit is not below an ask's limit; only crossing
//     pairs trade.
//   - Market price: the volume-weighted mean of a market's trades in a tick,
//     rounded half to even, or the previous price when nothing traded.
//   - Price age: the ticks since a market price was last set by a trade.
//   - Unmatched: the units of admitted orders that did not trade in a tick.
//   - Market report: what one market publishes for a tick: price, age, volume,
//     unmatched units and the best unmatched bid and ask.
//   - Clearing: the rule that matches a market's orders and prices its trades;
//     AverageClearing is the default and Order, Trade and LotKey are its data.
package economysim
