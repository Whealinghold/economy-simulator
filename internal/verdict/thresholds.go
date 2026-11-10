package verdict

// The keys of bands/regimes.toml, in the order of STY 6.2; the shared keys come
// last. A key is a rule constant: its name is part of the band file.
const (
	kR1Rise       = "r1.rise_bp"
	kR1Windows    = "r1.windows"
	kR1MFactor    = "r1.m_factor_bp"
	kR2Vol        = "r2.vol_bp"
	kR2Share      = "r2.share_bp"
	kR2Unsold     = "r2.unsold_bp"
	kR2Top        = "r2.top_bp"
	kR2Windows    = "r2.windows"
	kR3Fall       = "r3.fall_bp"
	kR3Recover    = "r3.recover_bp"
	kR3Factor     = "r3.horizon_factor"
	kR3Horizon    = "r3.horizon"
	kR3RoleTicks  = "r3.role_ticks"
	kR4Vol        = "r4.vol_bp"
	kR4Windows    = "r4.windows"
	kR5Pinned     = "r5.pinned_bp"
	kR5PinnedTick = "r5.pinned_ticks"
	kR5Freeze     = "r5.freeze_ticks"
	kR6Factor     = "r6.factor"
	kR6Growth     = "r6.growth_bp"
	kR6Windows    = "r6.windows"
	kR6Swing      = "r6.swing_bp"
	kR7Disp       = "r7.disp_bp"
	kR7Spike      = "r7.spike_factor"
	kR7First      = "r7.first_ticks"
	kR8Offset     = "r8.offset_bp"
	kR8Unmet      = "r8.unmet_bp"
	kR9Fall       = "r9.fall_bp"
	kR9Windows    = "r9.windows"
	kStaleMax     = "stale_max_bp"
	kActiveMin    = "active_min_bp"
	kMinWindows   = "min_windows"
)

// Key is one threshold. A key with Set false has no value line in the band
// file: it is unset, and a value of 0 is a value (STY 6.5.1).
type Key struct {
	Name  string
	Value int64
	Set   bool
}

// Thresholds is the ordered list of the keys of regimes.toml, in the order of
// STY 6.2 (D582). The loader fills it; a detector that needs an unset key
// returns Unknown with ReasonThresholdUnset.
type Thresholds []Key

// ThresholdNames returns the names of the keys in the order of STY 6.2.
func ThresholdNames() []string {
	return []string{
		kR1Rise, kR1Windows, kR1MFactor,
		kR2Vol, kR2Share, kR2Unsold, kR2Top, kR2Windows,
		kR3Fall, kR3Recover, kR3Factor, kR3Horizon, kR3RoleTicks,
		kR4Vol, kR4Windows,
		kR5Pinned, kR5PinnedTick, kR5Freeze,
		kR6Factor, kR6Growth, kR6Windows, kR6Swing,
		kR7Disp, kR7Spike, kR7First,
		kR8Offset, kR8Unmet,
		kR9Fall, kR9Windows,
		kStaleMax, kActiveMin, kMinWindows,
	}
}

// NewThresholds returns every key of ThresholdNames, unset.
func NewThresholds() Thresholds {
	names := ThresholdNames()
	t := make(Thresholds, len(names))
	for i, n := range names {
		t[i] = Key{Name: n}
	}
	return t
}

// Set gives the named key a value and reports whether the name is a key.
func (t Thresholds) Set(name string, value int64) bool {
	for i := range t {
		if t[i].Name == name {
			t[i].Value, t[i].Set = value, true
			return true
		}
	}
	return false
}

// Get returns the value of the named key and whether it is set.
func (t Thresholds) Get(name string) (int64, bool) {
	for i := range t {
		if t[i].Name == name {
			return t[i].Value, t[i].Set
		}
	}
	return 0, false
}
