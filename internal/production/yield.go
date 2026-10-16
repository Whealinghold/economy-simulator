package production

import "github.com/Whealinghold/economy-sim/internal/checked"

// BP is one whole unit in basis points: a factor of 10 000 changes nothing.
const BP = 10_000

// SeasonBP is one season entry of a yield profile: the season's index in the
// calendar and its factor in basis points.
type SeasonBP struct {
	Season int
	BP     int64
}

// Profile is a yield profile reduced to primitives. Seasons is sorted by
// ascending Season, as the configuration sorts it.
type Profile struct {
	HasAll  bool
	All     int64
	Seasons []SeasonBP
}

// Site returns the site factor of PRO 6.2: the profile's entry for season,
// else All, else 0 when the profile lists seasons, else (no profile, or a
// profile with neither) 10 000 for a recipe that is not sited and 0 for one
// that is. profile is nil for no profile. A season of -1 (none) matches no
// entry. The result is not clamped; validation bounds the table.
func Site(profile *Profile, sited bool, season int) int64 {
	if profile == nil {
		if sited {
			return 0
		}
		return BP
	}
	if season >= 0 {
		for _, s := range profile.Seasons {
			if s.Season == season {
				return s.BP
			}
			if s.Season > season {
				break
			}
		}
	}
	if profile.HasAll {
		return profile.All
	}
	if len(profile.Seasons) > 0 {
		return 0
	}
	// A profile with neither All nor seasons is not valid; read it as absent.
	if sited {
		return 0
	}
	return BP
}

// Effective returns the effective factor of PRO 6.3:
// site * shock / 10 000, then * tech / 10 000, then * trainee / 10 000, then
// * abundance / 10 000, each step truncated toward zero. ok is false when a
// step does not fit int64; nothing wraps.
func Effective(site, shock, tech, trainee, abundance int64) (e int64, ok bool) {
	e = site
	for _, f := range [...]int64{shock, tech, trainee, abundance} {
		if e, ok = checked.MulDiv(e, f, BP); !ok {
			return 0, false
		}
	}
	return e, true
}

// Output returns the units a job of k executions of an output of q units per
// execution creates at effective factor e, and the new residue (PRO 6.6):
// (k * q * e + residue) / 10 000 and its remainder. The sum is never formed;
// the product's quotient and remainder are added apart, so a carry of at most
// 1 reaches the quotient. residue must be in [0, 10 000) and k, q, e not
// negative; ok is false otherwise, or when produced does not fit int64.
func Output(k, q, e, residue int64) (produced, newResidue int64, ok bool) {
	if k < 0 || q < 0 || e < 0 || residue < 0 || residue >= BP {
		return 0, 0, false
	}
	kq, ok := checked.Mul(k, q)
	if !ok {
		return 0, 0, false
	}
	produced, rem, ok := checked.MulDivRem(kq, e, BP)
	if !ok {
		return 0, 0, false
	}
	rem += residue // below 2 * 10 000
	if rem >= BP {
		rem -= BP
		if produced, ok = checked.Add(produced, 1); !ok {
			return 0, 0, false
		}
	}
	return produced, rem, true
}

// Rescale applies the fall of an executor's weight to w to a job of k
// executions (PRO 5.4): the executions become min(k, w) and each pending
// quantity becomes floor(p * min(k, w) / k), in place. A weight that is not
// below k changes nothing. It returns the new executions; ok is false, and
// nothing changes, when k is not positive or w is negative.
func Rescale(k, w int64, pending []int64) (executions int64, ok bool) {
	if k <= 0 || w < 0 {
		return k, false
	}
	if w >= k {
		return k, true
	}
	for i, p := range pending {
		v, ok := checked.MulDivFloor(p, w, k)
		if !ok {
			return k, false
		}
		pending[i] = v
	}
	return w, true
}
