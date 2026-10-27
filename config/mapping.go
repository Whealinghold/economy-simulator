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
	// module is on (spec-config.md 6.2.1). A module that is not built stays an
	// unknown key; each module's change adds its table here, in the order of the
	// Modules row.
	if m := root.sub("modules"); m != nil {
		if d := m.sub("demand"); d != nil {
			cfg.Modules.Demand = l.demand(d)
			d.finish()
		}
		if ms := m.sub("money_supply"); ms != nil {
			cfg.Modules.MoneySupply = l.moneySupply(ms)
			ms.finish()
		}
		m.finish()
	}
	root.finish()
	return cfg
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
