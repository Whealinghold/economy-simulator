package economysim

// This file is the validation of the insurance module (spec-insurance.md 14; the
// continuation file of insurance.go, architecture.md 2.6): the rules of 14.1 that
// Validate checks and the eleven warnings of 14.2.

// insuranceBase is the path of the module.
var insuranceBase = []string{"modules", "insurance"}

// validateInsurance checks the insurance module and the cover blocks of the roles
// (spec-insurance.md 14.1, FR-INS-010, FR-INS-011). It runs in three phases, in the
// order of 14.1, and a phase that finds an error ends the run, so the first error is
// the same for the same mistake: the bounds of the module and of every block, then
// the shape of every block (ErrCoverShape), then the dependencies (the module needs
// Risk and Credit, a block needs the module, a route request needs Transport and a
// store request Storage). The reason collision is the money flows' rule
// (moneyflow.go). A field absent from a file is ErrMissingField, reported by the
// loader.
func (v *validator) validateInsurance() {
	steps := []func() bool{v.insuranceBounds, v.insuranceShapes, v.insuranceDependencies}
	for _, step := range steps {
		if !step() {
			return
		}
	}
}

// insuranceBounds checks the five fields of the module and every field of every block
// against its bound of spec-insurance.md 13, and reports whether all are inside.
func (v *validator) insuranceBounds() bool {
	n := len(v.rep.errs)
	if m := v.c.Modules.Insurance; m != nil {
		v.ranges(insuranceBase, []fieldRange{
			{"value_cap_bp", m.ValueCapBP, 1, MaxValueCapBP},
			{"payout_ticks", m.PayoutTicks, 1, MaxCreditTerm},
			{"min_sum", m.MinSum, 1, MaxMoney},
			{"max_policies", m.MaxPolicies, 1, MaxOpenPolicies},
			{"max_cover_requests", m.MaxCoverRequests, 1, MaxCoverRequestsBound},
		})
	}
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		if ro.Cover != nil {
			v.coverBlockBounds(path([]string{"roles", ro.ID}, "cover"), ro.Cover)
		}
	}
	return len(v.rep.errs) == n
}

// coverBlockBounds checks the fields of the sub-blocks of one role's block.
func (v *validator) coverBlockBounds(base []string, t *CoverTerms) {
	if u := t.Underwrite; u != nil {
		v.ranges(path(base, "underwrite"), []fieldRange{
			{"loading_bp", u.LoadingBP, 0, MaxLoadingBound},
			{"store_rate_bp", u.StoreRateBP, 0, MaxStoreRateBound},
			{"cover_bp", u.CoverBP, 1, wholeBP},
			{"cap_bp", u.CapBP, 1, MaxCapBound},
			{"share_bp", u.ShareBP, 1, wholeBP},
		})
	}
	if s := t.Insure; s != nil {
		v.ranges(path(base, "insure"), []fieldRange{
			{"min_loss_bp", s.MinLossBP, 0, wholeBP},
			{"ruin_share_bp", s.RuinShareBP, 0, MaxRuinBound},
			{"max_loading_bp", s.MaxLoadingBP, 0, MaxLoadingBound},
			{"min_cover_bp", s.MinCoverBP, 1, wholeBP},
			{"store_share_bp", s.StoreShareBP, 0, wholeBP},
			{"max_store_rate_bp", s.MaxStoreRateBP, 0, MaxStoreRateBound},
		})
	}
}

// insuranceShapes reports ErrCoverShape for a block with both Underwrite and Insure or
// with neither, and whether every block is right.
func (v *validator) insuranceShapes() bool {
	n := len(v.rep.errs)
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		if t := ro.Cover; t != nil && (t.Underwrite == nil) == (t.Insure == nil) {
			v.rep.add(ErrCoverShape, path([]string{"roles", ro.ID}, "cover"), coverShapeValue(t), "1")
		}
	}
	return len(v.rep.errs) == n
}

// coverShapeValue names what a bad block holds, "both" or "neither", for the Value of
// the error.
func coverShapeValue(t *CoverTerms) string {
	if t.Underwrite != nil {
		return "both"
	}
	return "neither"
}

// insuranceDependencies reports ErrModuleDependency for the module with Risk or Credit
// off, for a role with a cover block while Modules.Insurance is nil, for an Insure
// block with MinLossBP above 0 and no Modules.Transport and for one with StoreShareBP
// above 0 and no Modules.Storage, and whether every one is right.
func (v *validator) insuranceDependencies() bool {
	n := len(v.rep.errs)
	m := &v.c.Modules
	if m.Insurance != nil {
		v.insuranceNeeds(insuranceBase, "risk", m.Risk != nil)
		v.insuranceNeeds(insuranceBase, "credit", m.Credit != nil)
	}
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		if ro.Cover == nil {
			continue
		}
		p := path([]string{"roles", ro.ID}, "cover")
		if m.Insurance == nil {
			v.rep.add(ErrModuleDependency, p, "cover block", "modules.insurance")
			continue
		}
		v.insuranceInsureNeeds(p, ro.Cover.Insure)
	}
	return len(v.rep.errs) == n
}

// insuranceNeeds reports ErrModuleDependency at p for a module the insurance module
// needs and that is off.
func (v *validator) insuranceNeeds(p []string, module string, on bool) {
	if !on {
		v.rep.add(ErrModuleDependency, p, "modules.insurance", "modules."+module)
	}
}

// insuranceInsureNeeds reports ErrModuleDependency for an Insure block that asks for
// route cover without Transport or for store cover without Storage.
func (v *validator) insuranceInsureNeeds(p []string, s *InsureTerms) {
	if s == nil {
		return
	}
	if s.MinLossBP > 0 && v.c.Modules.Transport == nil {
		v.rep.add(ErrModuleDependency, path(p, "insure", "min_loss_bp"), "route cover", "modules.transport")
	}
	if s.StoreShareBP > 0 && v.c.Modules.Storage == nil {
		v.rep.add(ErrModuleDependency, path(p, "insure", "store_share_bp"), "store cover", "modules.storage")
	}
}

// coverRoles are the roles of a configuration that carry a cover block, by the half
// they carry, in the order the roles were written.
type coverRoles struct {
	underwriters, insured []*Role
}

// coverRolesOf collects the underwriter and the insured roles of c. A block with the
// wrong shape is counted by the half it has, so the warnings never panic on one that
// Validate refuses.
func coverRolesOf(c *Config) coverRoles {
	var cr coverRoles
	for i := range c.Roles {
		ro := &c.Roles[i]
		if ro.Cover == nil {
			continue
		}
		if ro.Cover.Underwrite != nil {
			cr.underwriters = append(cr.underwriters, ro)
		}
		if ro.Cover.Insure != nil {
			cr.insured = append(cr.insured, ro)
		}
	}
	return cr
}

// insuranceWarn is a warning of the module itself, at modules.insurance.
func insuranceWarn(rule string) Warning {
	return Warning{Path: insuranceBase, Rule: rule}
}

// insuranceWarnings returns the eleven warnings of spec-insurance.md 14.2 that this
// module raises, unsorted (the caller sorts). It is empty with the module off and
// none fails validation. cover_no_insured is raised from the configuration alone: a
// consumer's strategy that could request is not known to it, so the warning is
// informational, as the credit module's.
func insuranceWarnings(c *Config) []Warning {
	m := c.Modules.Insurance
	if m == nil {
		return nil
	}
	cr := coverRolesOf(c)
	out := insuranceSetWarnings(c, cr)
	out = append(out, insuranceRoleWarnings(c, cr)...)
	return append(out, insuranceModuleWarnings(c, m, cr)...)
}

// insuranceSetWarnings are the warnings about the set of roles: no_cover_roles,
// cover_no_underwriter, cover_no_insured, cover_without_risk and loading_above_limits.
func insuranceSetWarnings(c *Config, cr coverRoles) []Warning {
	var out []Warning
	switch {
	case len(cr.underwriters)+len(cr.insured) == 0:
		return []Warning{insuranceWarn("no_cover_roles")}
	case len(cr.underwriters) == 0:
		out = append(out, insuranceWarn("cover_no_underwriter"))
	case len(cr.insured) == 0:
		out = append(out, insuranceWarn("cover_no_insured"))
	}
	if coverWithoutRisk(c, cr) {
		out = append(out, insuranceWarn("cover_without_risk"))
	}
	if loadingAboveLimits(cr) {
		out = append(out, insuranceWarn("loading_above_limits"))
	}
	return out
}

// coverWithoutRisk is cover_without_risk: an insured with MinLossBP above 0 and no
// route row of Modules.Risk, or one with StoreShareBP above 0 and no store row.
func coverWithoutRisk(c *Config, cr coverRoles) bool {
	var routes, stores int
	if r := c.Modules.Risk; r != nil {
		routes, stores = len(r.Routes), len(r.Stores)
	}
	for _, ro := range cr.insured {
		s := ro.Cover.Insure
		if (s.MinLossBP > 0 && routes == 0) || (s.StoreShareBP > 0 && stores == 0) {
			return true
		}
	}
	return false
}

// loadingAboveLimits is loading_above_limits: the lowest LoadingBP of any underwriter
// is above the highest MaxLoadingBP of any insured, so no route cover can form. It is
// false when either set is empty.
func loadingAboveLimits(cr coverRoles) bool {
	if len(cr.underwriters) == 0 || len(cr.insured) == 0 {
		return false
	}
	lowest := cr.underwriters[0].Cover.Underwrite.LoadingBP
	for _, ro := range cr.underwriters {
		lowest = min(lowest, ro.Cover.Underwrite.LoadingBP)
	}
	var highest int64
	for _, ro := range cr.insured {
		highest = max(highest, ro.Cover.Insure.MaxLoadingBP)
	}
	return lowest > highest
}

// insuranceRoleWarnings are the warnings of one underwriter role:
// underwriter_without_income and underwriter_cap_above_money. They sit at the role's
// cover block.
func insuranceRoleWarnings(c *Config, cr coverRoles) []Warning {
	var out []Warning
	for _, ro := range cr.underwriters {
		p := path([]string{"roles", ro.ID}, "cover")
		if len(ro.Recipes) == 0 && !flowTargetsRole(c, ro.ID) {
			out = append(out, Warning{Path: p, Rule: "underwriter_without_income"})
		}
		if ro.Cover.Underwrite.CapBP > wholeBP && len(ro.StartStock) == 0 {
			out = append(out, Warning{Path: path(p, "underwrite", "cap_bp"), Rule: "underwriter_cap_above_money"})
		}
	}
	return out
}

// flowTargetsRole reports whether a money flow targets the households of role id: a
// households target that names the role, or names none and so every role.
func flowTargetsRole(c *Config, id string) bool {
	ms := c.Modules.MoneySupply
	if ms == nil {
		return false
	}
	for _, f := range ms.Flows {
		if f.Target.Kind == TargetHouseholds && (f.Target.Role == "" || f.Target.Role == id) {
			return true
		}
	}
	return false
}

// insuranceModuleWarnings are the warnings that read the module's own numbers and
// the credit module's: cover_below_minimum, claims_below_policies,
// no_shield_underwriter and value_cap_unbound.
func insuranceModuleWarnings(c *Config, m *Insurance, cr coverRoles) []Warning {
	var out []Warning
	if coverBelowMinimum(m, cr) {
		out = append(out, insuranceWarn("cover_below_minimum"))
	}
	if cd := c.Modules.Credit; cd != nil {
		if cd.MaxClaims < m.MaxPolicies {
			out = append(out, insuranceWarn("claims_below_policies"))
		}
		if cd.ShieldTicks == 0 && len(cr.underwriters) > 0 {
			out = append(out, insuranceWarn("no_shield_underwriter"))
		}
	}
	if m.ValueCapBP == MaxValueCapBP {
		out = append(out, Warning{Path: path(insuranceBase, "value_cap_bp"), Rule: "value_cap_unbound"})
	}
	return out
}

// coverBelowMinimum is cover_below_minimum: no cargo can reach MinSum for any insured
// role. The configuration holds no shipment, so it reads the largest value a household
// of the role starts with, its StartMoney, as the bound of what a first cargo can be
// worth: MinSum above it for every insured role means no policy can form at the start.
func coverBelowMinimum(m *Insurance, cr coverRoles) bool {
	if len(cr.insured) == 0 {
		return false
	}
	for _, ro := range cr.insured {
		if int64(ro.StartMoney) >= m.MinSum {
			return false
		}
	}
	return true
}
