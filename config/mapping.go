package config

import (
	"maps"
	"slices"
	"strconv"

	economysim "github.com/Whealinghold/economy-sim"
)

// tbl is a table of the merged tree being read. It records the keys that were
// read so that finish can reject the rest (ErrUnknownKey, at any depth).
type tbl struct {
	l    *loader
	path []string
	m    map[string]any
	used map[string]struct{}
}

// table wraps v as a table at path, or reports ErrWrongType and returns nil.
func (l *loader) table(path []string, v any) *tbl {
	m, ok := v.(map[string]any)
	if !ok {
		l.fail(economysim.ErrWrongType, path)
		return nil
	}
	return &tbl{l: l, path: path, m: m, used: map[string]struct{}{}}
}

// at is the path of a key of the table, in a slice of its own.
func (t *tbl) at(key string) []string {
	return append(t.path[:len(t.path):len(t.path)], key)
}

// get returns the value of key and marks it read.
func (t *tbl) get(key string) (any, bool) {
	v, ok := t.m[key]
	if ok {
		t.used[key] = struct{}{}
	}
	return v, ok
}

// finish reports every key that was not read as ErrUnknownKey.
func (t *tbl) finish() {
	for k := range t.m {
		if _, ok := t.used[k]; !ok {
			t.l.fail(economysim.ErrUnknownKey, t.at(k))
		}
	}
}

// need reports ErrMissingField for a required key that is absent.
func (t *tbl) need(key string, present bool) {
	if !present {
		t.l.fail(economysim.ErrMissingField, t.at(key))
	}
}

// integer reads an integer key into dst. A float is ErrNotInteger and any other
// type ErrWrongType. It returns whether the key is present.
func integer[T ~int64](t *tbl, key string, dst *T) bool {
	v, ok := t.get(key)
	if !ok {
		return false
	}
	switch n := v.(type) {
	case int64:
		*dst = T(n)
	case float64:
		t.l.fail(economysim.ErrNotInteger, t.at(key))
	default:
		t.l.fail(economysim.ErrWrongType, t.at(key))
	}
	return true
}

// text reads a string key into dst and returns whether the key is present.
func (t *tbl) text(key string, dst *string) bool {
	v, ok := t.get(key)
	if !ok {
		return false
	}
	if s, isString := v.(string); isString {
		*dst = s
	} else {
		t.l.fail(economysim.ErrWrongType, t.at(key))
	}
	return true
}

// flag reads a bool key into dst and returns whether the key is present.
func (t *tbl) flag(key string, dst *bool) bool {
	v, ok := t.get(key)
	if !ok {
		return false
	}
	if b, isBool := v.(bool); isBool {
		*dst = b
	} else {
		t.l.fail(economysim.ErrWrongType, t.at(key))
	}
	return true
}

// strings reads an array of strings into dst, which an empty array makes nil.
func (t *tbl) strings(key string, dst *[]string) bool {
	arr, ok := t.array(key)
	if !ok {
		return false
	}
	var out []string
	for i, v := range arr {
		s, isString := v.(string)
		if !isString {
			t.l.fail(economysim.ErrWrongType, append(t.at(key), index(i)))
			continue
		}
		out = append(out, s)
	}
	*dst = out
	return true
}

// array returns the elements of an array key, or reports ErrWrongType. The
// second result is whether the key is present.
func (t *tbl) array(key string) ([]any, bool) {
	v, ok := t.get(key)
	if !ok {
		return nil, false
	}
	arr, isArray := v.([]any)
	if !isArray {
		t.l.fail(economysim.ErrWrongType, t.at(key))
		return nil, true
	}
	return arr, true
}

// index is the path segment of an array element.
func index(i int) string { return "[" + strconv.Itoa(i) + "]" }

// rows reads an array of inline tables, calling each for every element in the
// order written, and then rejects the keys it did not read. It returns whether
// the key is present.
func (t *tbl) rows(key string, each func(e *tbl)) bool {
	arr, ok := t.array(key)
	for i, v := range arr {
		e := t.l.table(append(t.at(key), index(i)), v)
		if e == nil {
			continue
		}
		each(e)
		e.finish()
	}
	return ok
}

// sub returns the sub-table at key, or nil when it is absent or not a table.
func (t *tbl) sub(key string) *tbl {
	v, ok := t.get(key)
	if !ok {
		return nil
	}
	return t.l.table(t.at(key), v)
}

// entities calls each, in the order of the IDs, for the table at key whose
// entries are tables keyed by ID, and rejects the keys each did not read. The
// table itself is not finished: its keys are all entities.
func (t *tbl) entities(key string, each func(id string, e *tbl)) {
	if et := t.sub(key); et != nil {
		et.eachSub(each)
	}
}

// eachSub calls each, in the order of the keys, for every key of the table, all
// of whose values must be tables, and rejects the keys each did not read. It is
// the second level of a row keyed by two IDs.
func (t *tbl) eachSub(each func(id string, e *tbl)) {
	for _, id := range slices.Sorted(maps.Keys(t.m)) {
		e := t.sub(id)
		if e == nil {
			continue
		}
		each(id, e)
		e.finish()
	}
}

// intField is an integer key and the field it fills.
type intField struct {
	key string
	dst *int64
}

// readInts reads every listed integer key that is present.
func readInts(t *tbl, fields []intField) {
	for _, f := range fields {
		integer(t, f.key, f.dst)
	}
}

// items reads a list of (commodity, quantity) as an array of inline tables.
func (t *tbl) items(key string, dst *[]economysim.Item) bool {
	var out []economysim.Item
	ok := t.rows(key, func(e *tbl) {
		var it economysim.Item
		e.need("commodity", e.text("commodity", &it.Commodity))
		e.need("quantity", integer(e, "quantity", &it.Quantity))
		out = append(out, it)
	})
	if ok {
		*dst = out
	}
	return ok
}

// config maps the merged tree to a Config and rejects the keys it does not
// know (spec-config.md 6.2, 6.3). Rejections are collected in l.
func (l *loader) config(tree map[string]any) economysim.Config {
	root := &tbl{l: l, m: tree, used: map[string]struct{}{}}
	root.get("schema")
	cfg := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}

	if w := root.sub("world"); w != nil {
		l.world(w, &cfg.World)
		w.finish()
	}
	if c := root.sub("calendar"); c != nil {
		l.calendar(c, &cfg.Calendar)
		c.finish()
	}
	root.entities("commodity", func(id string, e *tbl) {
		cfg.Commodities = append(cfg.Commodities, l.commodity(id, e))
	})
	root.entities("recipe", func(id string, e *tbl) {
		cfg.Recipes = append(cfg.Recipes, l.recipe(id, e))
	})
	root.entities("role", func(id string, e *tbl) {
		cfg.Roles = append(cfg.Roles, l.role(id, e))
	})
	root.entities("region", func(id string, e *tbl) {
		cfg.Regions = append(cfg.Regions, l.region(id, e, &cfg))
	})
	if ws := root.sub("warm_start"); ws != nil {
		cfg.WarmStart = l.warmStart(ws)
		ws.finish()
	}
	// The table of a module is [modules.<name>] and is present if and only if the
	// module is on (spec-config.md 6.2.1); see modules.
	if m := root.sub("modules"); m != nil {
		l.modules(m, &cfg)
	}
	root.finish()
	return cfg
}

// modules reads the [modules] table. A module that is not built stays an unknown
// key; each module's change adds its table here, in the order of the Modules row.
func (l *loader) modules(m *tbl, cfg *economysim.Config) {
	if d := m.sub("demand"); d != nil {
		cfg.Modules.Demand = l.demand(d)
		d.finish()
	}
	if ms := m.sub("money_supply"); ms != nil {
		cfg.Modules.MoneySupply = l.moneySupply(ms)
		ms.finish()
	}
	if ps := m.sub("population"); ps != nil {
		cfg.Modules.Population = l.population(ps)
		ps.finish()
	}
	if rs := m.sub("resources"); rs != nil {
		cfg.Modules.Resources = l.resources(rs)
		rs.finish()
	}
	if ss := m.sub("subsistence"); ss != nil {
		cfg.Modules.Subsistence = l.subsistence(ss)
		ss.finish()
	}
	if ts := m.sub("transport"); ts != nil {
		cfg.Modules.Transport = l.transport(ts)
		ts.finish()
	}
	if is := m.sub("information"); is != nil {
		cfg.Modules.Information = l.information(is)
		is.finish()
	}
	l.laterModules(m, cfg)
	m.finish()
}

// laterModules reads the tables of the modules of the later waves: migration,
// merchants, estates and employment (a continuation of modules for the Lizard limit).
func (l *loader) laterModules(m *tbl, cfg *economysim.Config) {
	if gs := m.sub("migration"); gs != nil {
		cfg.Modules.Migration = l.migration(gs)
		gs.finish()
	}
	if ms := m.sub("merchants"); ms != nil {
		cfg.Modules.Merchants = l.merchants(ms)
		ms.finish()
	}
	if xs := m.sub("external"); xs != nil {
		cfg.Modules.External = l.external(xs)
		xs.finish()
	}
	if es := m.sub("estates"); es != nil {
		cfg.Modules.Estates = &economysim.Estates{}
		es.finish()
	}
	if es := m.sub("employment"); es != nil {
		cfg.Modules.Employment = &economysim.Employment{}
		es.finish()
	}
	if ps := m.sub("polity"); ps != nil {
		cfg.Modules.Polity = l.polity(ps)
		ps.finish()
	}
	if ss := m.sub("storage"); ss != nil {
		cfg.Modules.Storage = l.storage(ss)
		ss.finish()
	}
}

// storage reads [modules.storage] (spec-config.md 6.2.1, spec-storage.md 12.5): the
// base room as [base] (keys mass, bulk), a role row as [role.<id>] (key stores, and
// a [role.<id>.base] table), and a class as [class.<id>] (keys mass, bulk, needs,
// label). A key-less table switches the module on; a base table with no key is a
// base room of 0, and an absent one is unlimited. The rows come out in ID order
// because every level is read in the order of the keys.
func (l *loader) storage(t *tbl) *economysim.Storage {
	var m economysim.Storage
	m.Base = capacity(t.sub("base"))
	t.entities("role", func(id string, e *tbl) {
		r := economysim.RoleRoom{Role: id}
		e.strings("stores", &r.Stores)
		r.Base = capacity(e.sub("base"))
		m.Roles = append(m.Roles, r)
	})
	t.entities("class", func(id string, e *tbl) {
		c := economysim.StoreClass{ID: id}
		readInts(e, []intField{{"mass", &c.Mass}, {"bulk", &c.Bulk}})
		e.text("needs", &c.Needs)
		e.text("label", &c.Label)
		m.Classes = append(m.Classes, c)
	})
	return &m
}

// capacity reads a room table (keys mass and bulk) and finishes it; nil for an
// absent table.
func capacity(t *tbl) *economysim.Capacity {
	if t == nil {
		return nil
	}
	var c economysim.Capacity
	readInts(t, []intField{{"mass", &c.Mass}, {"bulk", &c.Bulk}})
	t.finish()
	return &c
}

// polity reads [modules.polity] (spec-config.md 6.2, spec-polity.md 3.4.2): the
// jurisdictions as [jurisdiction.<id>] (keys parent, seat, start_money, label) and
// the membership as [member] with one key per region, <region> = "<jurisdiction>".
// A key-less table switches the module on. The rows come out in ID order because
// every level is read in the order of the keys.
func (l *loader) polity(t *tbl) *economysim.PolityConfig {
	var m economysim.PolityConfig
	t.entities("jurisdiction", func(id string, e *tbl) {
		j := economysim.Jurisdiction{ID: id}
		e.text("parent", &j.Parent)
		e.text("seat", &j.Seat)
		integer(e, "start_money", &j.StartMoney)
		e.text("label", &j.Label)
		m.Jurisdictions = append(m.Jurisdictions, j)
	})
	if mt := t.sub("member"); mt != nil {
		for _, region := range slices.Sorted(maps.Keys(mt.m)) {
			row := economysim.Member{Region: region}
			mt.text(region, &row.Jurisdiction)
			m.Members = append(m.Members, row)
		}
		mt.finish()
	}
	return &m
}

// world reads the [world] table over DefaultWorldParams (spec-config.md 3.7.2).
func (l *loader) world(t *tbl, w *economysim.WorldParams) {
	readInts(t, []intField{
		{"scale", &w.Scale}, {"min_agents", &w.MinAgents}, {"bulk_factor", &w.BulkFactor},
		{"welfare_window", &w.WelfareWindow}, {"window_cycles", &w.WindowCycles},
		{"cover_ticks", &w.CoverTicks}, {"input_jobs", &w.InputJobs},
		{"output_cover_ticks", &w.OutputCoverTicks}, {"favour_floor", &w.FavourFloor},
		{"reserve_ticks", &w.ReserveTicks}, {"spend_max_share", &w.SpendMaxShare},
		{"spend_half_ticks", &w.SpendHalfTicks}, {"rest_band", &w.RestBand},
		{"produce_halt_ratio", &w.ProduceHaltRatio}, {"basket_tolerance", &w.BasketTolerance},
		{"belief_step", &w.BeliefStep}, {"belief_adjust_prob", &w.BeliefAdjustProb},
		{"belief_significance", &w.BeliefSignificance}, {"belief_stock_low", &w.BeliefStockLow},
		{"belief_stock_high", &w.BeliefStockHigh}, {"belief_max_age", &w.BeliefMaxAge},
		{"belief_width_init", &w.BeliefWidthInit}, {"belief_width_min", &w.BeliefWidthMin},
		{"belief_width_max", &w.BeliefWidthMax}, {"belief_width_step", &w.BeliefWidthStep},
		{"belief_evidence_share", &w.BeliefEvidenceShare}, {"cost_floor", &w.CostFloor},
		{"cost_ceiling", &w.CostCeiling}, {"ref_band_low", &w.RefBandLow},
		{"ref_band_high", &w.RefBandHigh}, {"price_window", &w.PriceWindow},
		{"review_ticks", &w.ReviewTicks}, {"entry_share", &w.EntryShare},
		{"role_hysteresis", &w.RoleHysteresis}, {"role_cooldown", &w.RoleCooldown},
		{"entry_ratio", &w.EntryRatio}, {"change_privation", &w.ChangePrivation},
		{"remove_privation", &w.RemovePrivation}, {"death_step", &w.DeathStep},
		{"exit_share", &w.ExitShare}, {"renewal", &w.Renewal},
		{"entry_pool_share", &w.EntryPoolShare}, {"anchor_hold", &w.AnchorHold},
	})
	t.items("upkeep", &w.Upkeep)
	t.text("spawner", &w.Spawner)
	t.text("clearing", &w.Clearing)
	t.text("distance_label", &w.DistanceLabel)
}

// calendar reads the [calendar] table over the Earth calendar. A months or
// seasons array replaces the default list whole (spec-calendar.md 3.1).
func (l *loader) calendar(t *tbl, c *economysim.Calendar) {
	readInts(t, []intField{
		{"hours_per_day", &c.HoursPerDay}, {"days_per_week", &c.DaysPerWeek},
		{"ticks_per_day", &c.TicksPerDay}, {"days_per_tick", &c.DaysPerTick},
		{"start_day", &c.StartDay}, {"season_start", &c.SeasonStart},
	})
	var months []economysim.Month
	if t.rows("months", func(e *tbl) {
		var m economysim.Month
		e.need("id", e.text("id", &m.ID))
		e.need("days", integer(e, "days", &m.Days))
		months = append(months, m)
	}) {
		c.Months = months
	}
	var seasons []economysim.Season
	if t.rows("seasons", func(e *tbl) {
		var s economysim.Season
		e.need("id", e.text("id", &s.ID))
		e.need("days", integer(e, "days", &s.Days))
		seasons = append(seasons, s)
	}) {
		c.Seasons = seasons
	}
}

// commodity reads one [commodity.<id>] table over DefaultCommodity.
func (l *loader) commodity(id string, t *tbl) economysim.Commodity {
	c := economysim.DefaultCommodity()
	c.ID = id
	readInts(t, []intField{
		{"mass", &c.Mass}, {"bulk", &c.Bulk}, {"shelf_life", &c.ShelfLife}, {"durability", &c.Durability},
	})
	t.strings("classes", &c.Classes)
	t.flag("raw", &c.Raw)
	t.need("reference_price", integer(t, "reference_price", &c.ReferencePrice))
	t.text("label", &c.Label)
	t.text("unit_label", &c.UnitLabel)
	return c
}

// recipe reads one [recipe.<id>] table over DefaultRecipe. The cost share of an
// output is 10 000 when the recipe has one output that is not discarded, and
// required otherwise (spec-production.md 3.2).
func (l *loader) recipe(id string, t *tbl) economysim.Recipe {
	r := economysim.DefaultRecipe()
	r.ID = id
	integer(t, "duration", &r.Duration)
	t.items("inputs", &r.Inputs)
	var draws []economysim.Draw
	if t.rows("draws", func(e *tbl) {
		var d economysim.Draw
		e.need("resource", e.text("resource", &d.Resource))
		e.need("quantity", integer(e, "quantity", &d.Quantity))
		draws = append(draws, d)
	}) {
		r.Draws = draws
	}
	var outs []economysim.Output
	var shared []bool // for each output, whether its cost share was written
	t.need("outputs", t.rows("outputs", func(e *tbl) {
		var o economysim.Output
		e.need("commodity", e.text("commodity", &o.Commodity))
		e.need("quantity", integer(e, "quantity", &o.Quantity))
		written := integer(e, "cost_share", &o.CostShare)
		e.flag("discard", &o.Discard)
		outs = append(outs, o)
		shared = append(shared, written)
	}))
	kept := 0
	for _, o := range outs {
		if !o.Discard {
			kept++
		}
	}
	for i := range outs {
		switch {
		case shared[i]:
		case kept == 1 && !outs[i].Discard:
			outs[i].CostShare = 10_000
		default:
			l.fail(economysim.ErrMissingField, append(t.at("outputs"), index(i), "cost_share"))
		}
	}
	r.Outputs = outs
	var requires []economysim.Tool
	if t.rows("requires", func(e *tbl) {
		var tool economysim.Tool
		e.need("commodity", e.text("commodity", &tool.Commodity))
		e.need("wear_per_use", integer(e, "wear_per_use", &tool.WearPerUse))
		requires = append(requires, tool)
	}) {
		r.Requires = requires
	}
	t.flag("sited", &r.Sited)
	t.flag("trainee_eligible", &r.TraineeEligible)
	integer(t, "trainee_yield", &r.TraineeYield)
	t.text("label", &r.Label)
	return r
}

// role reads one [role.<id>] table over DefaultRole.
func (l *loader) role(id string, t *tbl) economysim.Role {
	r := economysim.DefaultRole()
	r.ID = id
	t.strings("recipes", &r.Recipes)
	t.items("upkeep", &r.Upkeep)
	t.strings("trades", &r.Trades)
	integer(t, "start_money", &r.StartMoney)
	t.items("start_stock", &r.StartStock)
	t.text("strategy", &r.Strategy)
	t.flag("unskilled", &r.Unskilled)
	readInts(t, []intField{
		{"target_ratio", &r.TargetRatio}, {"target_spread", &r.TargetSpread},
		{"default_ratio", &r.DefaultRatio}, {"in_kind", &r.InKind},
		{"expected_work_ticks", &r.ExpectedWorkTicks}, {"min_work_share", &r.MinWorkShare},
		{"training_ticks", &r.TrainingTicks}, {"max_trainees", &r.MaxTrainees},
		{"teaching_load", &r.TeachingLoad},
	})
	t.flag("closed", &r.Closed)
	t.flag("informed", &r.Informed)
	t.flag("fixed", &r.Fixed)
	ms := t.sub("merchant")
	if ms != nil {
		r.MerchantFeeShare = economysim.DefaultMerchantFeeShare // the default of FR-CAR-012 applies to a merchant role
	}
	integer(t, "merchant_fee_share", &r.MerchantFeeShare)
	if ms != nil {
		m := l.merchant(ms)
		ms.finish()
		r.Merchant = &m
	}
	t.flag("hireable", &r.Hireable)
	if es := t.sub("employer"); es != nil {
		e := economysim.DefaultEmployer()
		integer(es, "payroll_bp", &e.PayrollBP)
		es.finish()
		r.Employer = &e
	}
	t.text("label", &r.Label)
	return r
}

// region reads one [region.<id>] table over DefaultRegion, and appends the
// edges, reference-price overrides, groups and yield profiles written under it
// to cfg (spec-config.md 6.2.2, spec-regions.md 4.1, 5.2).
// Regions are read in ID order and, under each, roles and recipes in ID order,
// so the lists come out in the canonical order of spec-config.md 3.4.
func (l *loader) region(id string, t *tbl, cfg *economysim.Config) economysim.Region {
	r := economysim.DefaultRegion()
	r.ID = id
	integer(t, "reference_basket", &r.ReferenceBasket)
	t.items("extra_upkeep", &r.ExtraUpkeep)
	integer(t, "capital_return", &r.CapitalReturn)
	integer(t, "estate_markdown", &r.EstateMarkdown)
	integer(t, "pool_release", &r.PoolRelease)
	integer(t, "relief_below", &r.ReliefBelow)
	t.text("label", &r.Label)
	t.entities("edge", func(to string, e *tbl) {
		edge := economysim.Edge{From: id, To: to}
		e.need("distance", integer(e, "distance", &edge.Distance))
		cfg.Edges = append(cfg.Edges, edge)
	})
	if p := t.sub("reference_price"); p != nil {
		for _, commodity := range slices.Sorted(maps.Keys(p.m)) {
			price := economysim.RegionPrice{Region: id, Commodity: commodity}
			integer(p, commodity, &price.Price)
			cfg.RegionPrices = append(cfg.RegionPrices, price)
		}
	}
	t.entities("yield", func(recipe string, e *tbl) {
		y := economysim.YieldProfile{Region: id, Recipe: recipe}
		y.HasAll = integer(e, "all", &y.All)
		if s := e.sub("seasons"); s != nil {
			for _, season := range slices.Sorted(maps.Keys(s.m)) {
				var bp int64
				integer(s, season, &bp)
				y.Seasons = append(y.Seasons, economysim.SeasonYield{Season: season, BP: bp})
			}
		}
		cfg.Yields = append(cfg.Yields, y)
	})
	t.entities("group", func(role string, e *tbl) {
		g := economysim.Group{Region: id, Role: role}
		e.need("households", integer(e, "households", &g.Households))
		integer(e, "scale", &g.Scale)
		cfg.Groups = append(cfg.Groups, g)
	})
	return r
}

// warmStart reads [warm_start] (spec-config.md 6.2.2): the four lists are
// written once per region. The table being present makes the WarmStart non-nil.
func (l *loader) warmStart(t *tbl) *economysim.WarmStart {
	ws := &economysim.WarmStart{}
	t.entities("region", func(region string, e *tbl) {
		var wage economysim.Money
		if integer(e, "wage", &wage) {
			ws.Wages = append(ws.Wages, economysim.SeedWage{Region: region, Wage: wage})
		}
		var pool economysim.Money
		if integer(e, "pool", &pool) {
			ws.Pools = append(ws.Pools, economysim.WarmPool{Region: region, Money: pool})
		}
		if p := e.sub("price"); p != nil {
			for _, commodity := range slices.Sorted(maps.Keys(p.m)) {
				var price economysim.Price
				integer(p, commodity, &price)
				ws.Prices = append(ws.Prices, economysim.SeedPrice{Region: region, Commodity: commodity, Price: price})
			}
		}
		e.entities("group", func(role string, g *tbl) {
			wg := economysim.WarmGroup{Region: region, Role: role}
			g.need("money", integer(g, "money", &wg.Money))
			g.items("stock", &wg.Stock)
			var jobs []economysim.JobSpec
			if g.rows("jobs", func(j *tbl) {
				var js economysim.JobSpec
				j.need("recipe", j.text("recipe", &js.Recipe))
				j.need("in_progress", integer(j, "in_progress", &js.InProgressBP))
				jobs = append(jobs, js)
			}) {
				wg.Jobs = jobs
			}
			ws.Groups = append(ws.Groups, wg)
		})
	})
	// The lists come out sorted by region, then commodity or role, because the
	// regions are read in ID order and each inner key in ID order.
	return ws
}

// demand reads [modules.demand] (spec-config.md 6.2.3, spec-demand.md 3.2): the
// wants as [want.<id>], the tastes as [taste.<role>.<want>] and the region tastes
// as [region_taste.<region>.<want>]. The table being present switches the module
// on, even with no key. Each row starts from its constructor, so an absent scale
// is 10 000 and a scale of 0 that the file gives is kept. The lists come out in
// canonical order because every level is read in ID order.
func (l *loader) demand(t *tbl) *economysim.Demand {
	d := &economysim.Demand{}
	t.entities("want", func(id string, e *tbl) { d.Wants = append(d.Wants, l.want(id, e)) })
	t.entities("taste", func(role string, r *tbl) {
		r.eachSub(func(want string, e *tbl) {
			row := economysim.DefaultTaste()
			row.Role, row.Want = role, want
			integer(e, "scale", &row.Scale)
			d.Tastes = append(d.Tastes, row)
		})
	})
	t.entities("region_taste", func(region string, r *tbl) {
		r.eachSub(func(want string, e *tbl) {
			row := economysim.DefaultRegionTaste()
			row.Region, row.Want = region, want
			integer(e, "scale", &row.Scale)
			d.RegionTastes = append(d.RegionTastes, row)
		})
	})
	return d
}

// want reads one [modules.demand.want.<id>] table over DefaultWant. The items and
// the quantity are required; an empty items array is read and left to Validate
// (ErrWantItems).
func (l *loader) want(id string, t *tbl) economysim.Want {
	w := economysim.DefaultWant()
	w.ID = id
	var items []economysim.WantItem
	t.need("items", t.rows("items", func(e *tbl) {
		it := economysim.DefaultWantItem()
		e.need("commodity", e.text("commodity", &it.Commodity))
		integer(e, "weight", &it.Weight)
		items = append(items, it)
	}))
	w.Items = items
	t.need("quantity", integer(t, "quantity", &w.Quantity))
	readInts(t, []intField{
		{"period", &w.Period}, {"rank", &w.Rank}, {"elasticity", &w.Elasticity},
		{"ceiling", &w.Ceiling}, {"substitution", &w.Substitution},
	})
	t.text("label", &w.Label)
	return w
}

// moneySupply reads [modules.money_supply] (spec-config.md 6.2.4,
// spec-money-supply.md 4.1): the flows as [flow.<id>], each with an inline or
// nested target = { kind, region, role }. The table being present switches the
// module on, even with no flow. Every default is the zero value, so an absent
// key is absent in the struct and the flows come out in ID order.
func (l *loader) moneySupply(t *tbl) *economysim.MoneySupply {
	m := &economysim.MoneySupply{}
	t.entities("flow", func(id string, e *tbl) {
		f := economysim.MoneyFlow{ID: id}
		e.need("kind", e.text("kind", &f.Kind))
		tg := e.sub("target")
		e.need("target", tg != nil)
		if tg != nil {
			tg.text("kind", &f.Target.Kind)
			tg.text("region", &f.Target.Region)
			tg.text("role", &f.Target.Role)
			tg.text("store", &f.Target.Store)
			tg.finish()
		}
		readInts(e, []intField{
			{"amount", &f.Amount}, {"level", &f.Level}, {"rate_bp", &f.RateBP},
			{"exempt", &f.Exempt}, {"start", &f.Start}, {"until", &f.Until},
			{"limit", &f.Limit},
		})
		e.text("per", &f.Per)
		e.flag("not_income", &f.NotIncome)
		e.text("label", &f.Label)
		m.Flows = append(m.Flows, f)
	})
	return m
}

// resources reads [modules.resources] (spec-config.md 6.2, spec-resources.md
// 3.3.2): the resources as [resource.<id>] over DefaultResource and the rows as
// [endowment.<region>.<resource>], each key of a row setting its Has flag. The
// table being present switches the module on, even with no key. The horizon
// starts from its default. The lists come out in canonical order because every
// level is read in ID order.
func (l *loader) resources(t *tbl) *economysim.Resources {
	m := economysim.DefaultResources()
	integer(t, "horizon_years", &m.HorizonYears)
	t.entities("resource", func(id string, e *tbl) { m.Resources = append(m.Resources, l.resource(id, e)) })
	t.entities("endowment", func(region string, r *tbl) {
		r.eachSub(func(resource string, e *tbl) {
			row := economysim.Endowment{Region: region, Resource: resource}
			row.HasCapacity = integer(e, "capacity", &row.Capacity)
			row.HasInitial = integer(e, "initial", &row.Initial)
			row.HasRegenScale = integer(e, "regen_scale_bp", &row.RegenScaleBP)
			m.Endowments = append(m.Endowments, row)
		})
	})
	return &m
}

// resource reads one [modules.resources.resource.<id>] table over
// DefaultResource. The curve and the season profile are arrays of inline tables
// in the order written; Validate checks them.
func (l *loader) resource(id string, t *tbl) economysim.Resource {
	r := economysim.DefaultResource()
	r.ID = id
	t.text("shape", &r.Shape)
	readInts(t, []intField{
		{"rate_per_year", &r.RatePerYear}, {"rate_bp", &r.RateBP}, {"refuge", &r.Refuge},
		{"capacity", &r.Capacity}, {"initial_bp", &r.InitialBP}, {"plan_share_bp", &r.PlanShareBP},
	})
	t.rows("abundance", func(e *tbl) {
		var k economysim.AbundanceKnot
		e.need("share", integer(e, "share", &k.Share))
		e.need("factor", integer(e, "factor", &k.Factor))
		r.Abundance = append(r.Abundance, k)
	})
	t.rows("seasons", func(e *tbl) {
		var s economysim.SeasonRegen
		e.need("season", e.text("season", &s.Season))
		e.need("bp", integer(e, "bp", &s.BP))
		r.Seasons = append(r.Seasons, s)
	})
	t.text("label", &r.Label)
	return r
}

// population reads [modules.population] (spec-config.md 6.2.1,
// spec-population.md 8): the two schedules as arrays of { ratio, rate } over
// DefaultPopulation, replaced whole by a later layer, the two tunables, and the
// region rows as [region.<id>] over DefaultRegionPopulation. The table being
// present switches the module on, even with no key. The rows come out in region
// order because every level is read in ID order.
func (l *loader) population(t *tbl) *economysim.Population {
	m := economysim.DefaultPopulation()
	knots := func(key string, dst *[]economysim.Knot) {
		var ks []economysim.Knot
		if t.rows(key, func(e *tbl) {
			var k economysim.Knot
			e.need("ratio", integer(e, "ratio", &k.Ratio))
			e.need("rate", integer(e, "rate", &k.Rate))
			ks = append(ks, k)
		}) {
			*dst = ks
		}
	}
	knots("birth_schedule", &m.BirthSchedule)
	knots("death_schedule", &m.DeathSchedule)
	readInts(t, []intField{{"min_households", &m.MinHouseholds}, {"birth_stake", &m.BirthStake}})
	t.entities("region", func(id string, e *tbl) {
		row := economysim.DefaultRegionPopulation(id)
		readInts(e, []intField{
			{"birth_scale", &row.BirthScale}, {"death_scale", &row.DeathScale}, {"household_cap", &row.HouseholdCap},
		})
		m.Regions = append(m.Regions, row)
	})
	return &m
}

// subsistence reads [modules.subsistence] (spec-config.md 6.2.1,
// spec-subsistence.md 3.2.2): the default list as recipes, the three tunables
// over DefaultSubsistence, and the role rows as [role.<id>] with recipes. The
// table being present switches the module on, even with no key; a row with no
// key is an empty row, which says the role has none. The rows come out in role
// order because every level is read in ID order.
func (l *loader) subsistence(t *tbl) *economysim.SubsistenceConfig {
	m := economysim.DefaultSubsistence()
	t.strings("recipes", &m.Recipes)
	readInts(t, []intField{
		{"enter_ticks", &m.EnterTicks}, {"exit_ticks", &m.ExitTicks}, {"retry_jobs", &m.RetryJobs},
	})
	t.entities("role", func(id string, e *tbl) {
		row := economysim.RoleOwnUse{Role: id}
		e.strings("recipes", &row.Recipes)
		m.Roles = append(m.Roles, row)
	})
	return &m
}

// transport reads [modules.transport] (spec-config.md 6.2.1, spec-transport.md
// 3.1): the modes as [mode.<id>] tables, with feed an array of { commodity,
// quantity } tables. The table being present switches the module on, even with
// no mode. A mode without a haul commodity starts from DefaultCrew, so an absent
// crew and an explicit 1 hash alike; a hired mode keeps a crew of 0, and one that
// sets crew is refused by Validate (ErrModeShape). The modes come out in ID order
// because every level is read in ID order.
func (l *loader) transport(t *tbl) *economysim.Transport {
	m := &economysim.Transport{}
	t.entities("mode", func(id string, e *tbl) {
		md := economysim.Mode{ID: id}
		e.text("haul", &md.Haul)
		if md.Haul == "" {
			md.Crew = economysim.DefaultCrew
		}
		readInts(e, []intField{
			{"speed", &md.Speed}, {"unit_mass", &md.UnitMass}, {"unit_bulk", &md.UnitBulk},
			{"crew", &md.Crew}, {"wear_per_tick", &md.WearPerTick}, {"haul_span", &md.HaulSpan},
		})
		e.text("vehicle", &md.Vehicle)
		e.items("feed", &md.Feed)
		e.text("label", &md.Label)
		m.Modes = append(m.Modes, md)
	})
	return m
}

// information reads [modules.information] over DefaultInformation
// (spec-config.md 6.2.1, spec-information.md 8): info_speed, rumour_error_bp and
// rumour_max_age. info_speed has no default, so a table that omits it is
// ErrOutOfBounds at Validate. The table being present switches the module on.
func (l *loader) information(t *tbl) *economysim.Information {
	m := economysim.DefaultInformation()
	readInts(t, []intField{
		{"info_speed", &m.InfoSpeed}, {"rumour_error_bp", &m.RumourErrorBP}, {"rumour_max_age", &m.RumourMaxAge},
	})
	return &m
}

// migration reads [modules.migration] over DefaultMigration (spec-config.md 6.2.1,
// spec-migration.md 11, 15.3): move_speed, move_review_ticks, move_below,
// move_gain, move_width, move_cooldown, move_horizon, move_out_share and
// move_in_share, and the region rows as [modules.migration.region.<id>] with
// out_share and in_share. move_speed has no default, so a table that omits it is
// ErrOutOfBounds at Validate. A key a row omits is the world's value, as the
// table of 11.2 says. The table being present switches the module on. The rows
// come out in region order because every level is read in ID order.
func (l *loader) migration(t *tbl) *economysim.Migration {
	m := economysim.DefaultMigration()
	readInts(t, []intField{
		{"move_speed", &m.MoveSpeed}, {"move_review_ticks", &m.MoveReviewTicks}, {"move_below", &m.MoveBelow},
		{"move_gain", &m.MoveGain}, {"move_width", &m.MoveWidth}, {"move_cooldown", &m.MoveCooldown},
		{"move_horizon", &m.MoveHorizon}, {"move_out_share", &m.MoveOutShare}, {"move_in_share", &m.MoveInShare},
	})
	t.entities("region", func(id string, e *tbl) {
		row := economysim.RegionMove{Region: id, OutShare: m.MoveOutShare, InShare: m.MoveInShare}
		readInts(e, []intField{{"out_share", &row.OutShare}, {"in_share", &row.InShare}})
		m.Regions = append(m.Regions, row)
	})
	return &m
}

// merchants reads [modules.merchants] over DefaultMerchants (spec-config.md 6.2.1,
// spec-merchants.md 13): max_open. The table being present switches the module on.
func (l *loader) merchants(t *tbl) *economysim.Merchants {
	m := economysim.DefaultMerchants()
	integer(t, "max_open", &m.MaxOpen)
	return &m
}

// merchant reads [role.<id>.merchant] over DefaultMerchant (spec-merchants.md 3.2,
// 13): modes, lot_units, fee_share_spread_bp, max_source_age, time_cost_bp,
// consignee_cut_bp, consigns, take_margin_bp and take_units. modes and take_units
// have no default, so a block that omits them is rejected at Validate. The table
// being present makes the role a merchant role.
func (l *loader) merchant(t *tbl) economysim.Merchant {
	m := economysim.DefaultMerchant()
	t.strings("modes", &m.Modes)
	readInts(t, []intField{
		{"lot_units", &m.LotUnits}, {"fee_share_spread_bp", &m.FeeShareSpreadBP}, {"max_source_age", &m.MaxSourceAge},
		{"time_cost_bp", &m.TimeCostBP}, {"consignee_cut_bp", &m.ConsigneeCutBP}, {"take_margin_bp", &m.TakeMarginBP},
	})
	t.flag("consigns", &m.Consigns)
	integer(t, "take_units", &m.TakeUnits)
	return m
}

// external reads [modules.external] (spec-config.md 6.2.3, spec-external-sector.md
// 3.3.1): the nodes as [node.<region>] with lines [line.<commodity>] (keys base,
// trend_bp, floor_bp, ceiling_bp, walk_step_bp, walk_revert_bp, walk_bound_bp and
// the array price_by_season), their sides [sell] and [buy] (margin_bp, capacity,
// total_units, total_value and the array window) and gates [gate.<region>]
// (scale_bp). A key-less node table marks the node, and the table being present
// switches the module on. The rows come out in ID order because every level is
// read in ID order.
func (l *loader) external(t *tbl) *economysim.External {
	var m economysim.External
	t.entities("node", func(region string, n *tbl) {
		node := economysim.OutsideNode{Region: region}
		n.text("label", &node.Label)
		n.entities("line", func(commodity string, e *tbl) { node.Lines = append(node.Lines, l.outsideLine(commodity, e)) })
		n.entities("gate", func(gate string, e *tbl) {
			row := economysim.DefaultOutsideGate(gate)
			integer(e, "scale_bp", &row.ScaleBP)
			node.Gates = append(node.Gates, row)
		})
		m.Nodes = append(m.Nodes, node)
	})
	return &m
}

// outsideLine reads one [modules.external.node.<region>.line.<commodity>] table
// over DefaultOutsideLine, with its two sides. A side is present when its table
// is written with a capacity.
func (l *loader) outsideLine(commodity string, t *tbl) economysim.OutsideLine {
	line := economysim.DefaultOutsideLine(commodity, 0)
	readInts(t, []intField{
		{"trend_bp", &line.TrendBP}, {"floor_bp", &line.FloorBP},
		{"ceiling_bp", &line.CeilingBP}, {"walk_step_bp", &line.WalkStepBP},
		{"walk_revert_bp", &line.WalkRevertBP}, {"walk_bound_bp", &line.WalkBoundBP},
	})
	integer(t, "base", &line.Base)
	line.PriceBySeason = seasonFactors(t, "price_by_season")
	if s := t.sub("sell"); s != nil {
		l.outsideSide(s, &line.Sell)
		s.finish()
	}
	if b := t.sub("buy"); b != nil {
		l.outsideSide(b, &line.Buy)
		b.finish()
	}
	return line
}

// outsideSide reads the keys of one side: margin_bp, capacity, total_units,
// total_value and the array window.
func (l *loader) outsideSide(t *tbl, s *economysim.OutsideSide) {
	readInts(t, []intField{
		{"margin_bp", &s.MarginBP}, {"capacity", &s.Capacity},
		{"total_units", &s.TotalUnits}, {"total_value", &s.TotalValue},
	})
	s.Window = seasonFactors(t, "window")
}

// seasonFactors reads an array of { season, bp } inline tables in the order
// written; Validate checks the seasons.
func seasonFactors(t *tbl, key string) []economysim.SeasonFactor {
	var out []economysim.SeasonFactor
	t.rows(key, func(e *tbl) {
		var f economysim.SeasonFactor
		e.need("season", e.text("season", &f.Season))
		e.need("bp", integer(e, "bp", &f.BP))
		out = append(out, f)
	})
	return out
}
