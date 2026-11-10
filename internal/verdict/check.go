package verdict

import "slices"

// FactClass is the class of a fact (STY 3.2): X exact, O ordinal, B banded, R
// report, P pack-asserted.
type FactClass string

// The closed set of fact classes.
const (
	ClassX FactClass = "X"
	ClassO FactClass = "O"
	ClassB FactClass = "B"
	ClassR FactClass = "R"
	ClassP FactClass = "P"
)

// Valid reports whether c is one of the five classes.
func (c FactClass) Valid() bool {
	return slices.Contains([]FactClass{ClassX, ClassO, ClassB, ClassR, ClassP}, c)
}

// Asserted reports whether a build fails on a fact of class c (STY 3.2, 8.1,
// FR-STY-011): X and O always, B and P only when every key of the fact has a
// reviewed value (for P, the caller asks for the pack that names the fact), and
// R never. A fact that is not asserted is reported.
func Asserted(c FactClass, keysSet bool) bool {
	switch c {
	case ClassX, ClassO:
		return true
	case ClassB, ClassP:
		return keysSet
	}
	return false
}

// Result is the verdict of a check (STY 7.2). These three words are the only
// place the module uses the word that 7.3 reserves (FR-STY-096).
type Result uint8

// The three results of a check.
const (
	// Undetermined: a B key is unset, a detector returned unknown, a blind-spot
	// field is outside its bound, or a needed overlay does not exist.
	Undetermined Result = iota
	// Stable: every conjunct is asserted and held, every B key is set and every
	// detector returned none.
	Stable
	// NotStable: a conjunct failed or a detector returned detected.
	NotStable
)

// String returns the word of the result as a record spells it.
func (r Result) String() string {
	switch r {
	case Stable:
		return "STABLE"
	case NotStable:
		return "NOT STABLE"
	}
	return "UNDETERMINED"
}

// The three checks of STY 7.1.
const (
	CheckBP = "C-BP" // bounded prices
	CheckNC = "C-NC" // no cobweb blow-up
	CheckWS = "C-WS" // a warm start stays near the steady state
)

// Checks returns the names of the three checks in the order of STY 7.1.
func Checks() []string { return []string{CheckBP, CheckNC, CheckWS} }

// required are the conjuncts every check needs, by the names the caller gives
// (STY 7.1). The lane part of C-NC (R6 over the Spread series, STY-P06's worst
// lane) is added by the caller where trade is on.
func required(check string) []string {
	switch check {
	case CheckBP:
		return []string{"STY-P01", "STY-P02", "R5a", "R5b"}
	case CheckNC:
		return []string{"STY-P03", "STY-D03", "R6"}
	case CheckWS:
		return []string{"STY-H01", "STY-H02", "STY-H03", "STY-H04", "STY-P04", "R7", "R8"}
	}
	return nil
}

// Required returns the conjuncts of a check by name, or nil for a name that is
// not a check.
func Required(check string) []string { return slices.Clone(required(check)) }

// Outcome is what one conjunct came to over the gate seeds.
type Outcome uint8

// The outcomes of a conjunct. The zero value is Open: nothing was asserted.
const (
	// Open: not run, a B key unset, or a blind-spot field outside its bound.
	Open Outcome = iota
	// Held: asserted and it held on every seed (SR1 for a detector and an
	// exact fact, SR2 for a band).
	Held
	// Failed: it failed, or a detector returned detected.
	Failed
)

// Conjunct is one fact or detector of a check, summarised over the seeds.
type Conjunct struct {
	// Name is a fact ID ("STY-P02") or a detector ("R5a").
	Name    string
	Outcome Outcome
	// Seed is the first seed on which it failed (Failed only).
	Seed int64
	// Reason says why it is Open: a closed phrase such as "band unset",
	// "unknown (windows)" or "overlay missing".
	Reason string
}

// OfVerdict summarises the verdict of a detector on one seed: none held,
// detected failed, unknown is open with its reason.
func OfVerdict(v Verdict, seed int64) Conjunct {
	c := Conjunct{Name: string(v.Detector), Seed: seed}
	switch v.Class {
	case None:
		c.Outcome = Held
	case Detected:
		c.Outcome = Failed
	default:
		c.Reason = "unknown (" + v.Reason + ")"
	}
	return c
}

// Report is the verdict of one check with what decided it.
type Report struct {
	Check  string
	Result Result
	// Failed lists the conjuncts that failed, with their seed, in the order
	// given; Open lists those that could not be judged, with the reason.
	Failed, Open []Conjunct
}

// Check combines the conjuncts of a check (STY 7.2, FR-STY-095): NOT STABLE if
// any failed, else UNDETERMINED if any is open or a conjunct the check needs is
// not given, else STABLE. A check that is not one of the three, or is given no
// conjunct, is UNDETERMINED. The conjuncts the check needs but the caller did
// not give are listed as open with the reason "not run".
func Check(check string, got []Conjunct) Report {
	rep := Report{Check: check}
	names := map[string]bool{}
	for _, c := range got {
		names[c.Name] = true
		switch c.Outcome {
		case Failed:
			rep.Failed = append(rep.Failed, c)
		case Open:
			rep.Open = append(rep.Open, c)
		}
	}
	for _, n := range required(check) {
		if !names[n] {
			rep.Open = append(rep.Open, Conjunct{Name: n, Reason: "not run"})
		}
	}
	switch {
	case len(rep.Failed) > 0:
		rep.Result = NotStable
	case len(rep.Open) > 0 || len(got) == 0 || required(check) == nil:
		rep.Result = Undetermined
	default:
		rep.Result = Stable
	}
	return rep
}
