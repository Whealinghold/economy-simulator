package saving

// PerTick returns the value per tick of a recipe, scaled by 10 000 so that two
// values are compared and subtracted in integers (spec 5.3.1, FR-SAV-009):
// trunc(net * 10 000 / duration). A duration below 1 is taken as 1.
func PerTick(net, duration int64) int64 {
	if duration < 1 {
		duration = 1
	}
	return satMulDiv(net, scale, duration)
}

// JobValue returns J(c, K) of spec 5.3.1 (FR-SAV-010): the best job value per
// tick of an agent of weight w with k executions of capacity of the class,
// max(pvB * min(w, k), pvA * w), where pvB is the best value of a recipe that
// needs the class and pvA the best of one that does not (0 if none).
func JobValue(pvB, pvA, w, k int64) int64 {
	w, k = nonneg(w), nonneg(k)
	n := w
	if k < n {
		n = k
	}
	with, without := satMul(nonneg(pvB), n), satMul(nonneg(pvA), w)
	if without > with {
		return without
	}
	return with
}

// BuildGain returns the gain per tick of raising the job value from before to
// after: at least 0 (FR-SAV-010).
func BuildGain(before, after int64) int64 { return nonneg(satSub(after, before)) }

// Rate returns the condition units a facility loses a tick, decay +
// ceil(wear * k / duration) (spec 5.3.1, FR-SAV-011). A duration below 1 is
// taken as 1.
func Rate(decay, wear, k, duration int64) int64 {
	if duration < 1 {
		duration = 1
	}
	return satAdd(nonneg(decay), satMulDivCeil(nonneg(wear), nonneg(k), duration))
}

// WorthRate is the rate of a class with a Worth row (spec 5.3.2): decay +
// wear * min(w, capacity), a duration of one tick being assumed.
func WorthRate(decay, wear, w, capacity int64) int64 {
	n := nonneg(w)
	if c := nonneg(capacity); c < n {
		n = c
	}
	return satAdd(nonneg(decay), satMul(nonneg(wear), n))
}

// ServiceTime returns T_p = min(floor(durability / rate), payback), payback for
// a rate of 0 (spec 5.3.1, FR-SAV-011).
func ServiceTime(durability, rate, payback int64) int64 {
	if rate <= 0 {
		return payback
	}
	if t := nonneg(durability) / rate; t < payback {
		return t
	}
	return payback
}

// SiteBenefit returns dJ * tp / 10 000 truncated, saturated at maxMoney
// (spec 5.3.1, FR-SAV-012 and 014).
func SiteBenefit(dJ, tp, maxMoney int64) int64 {
	return capAt(satMulDiv(nonneg(dJ), nonneg(tp), scale), maxMoney)
}

// WorthBenefit returns perTick * tp while the agent holds fewer than count
// facilities of the class, 0 at count, saturated at maxMoney (spec 5.3.2).
func WorthBenefit(perTick, tp, held, count, maxMoney int64) int64 {
	if held >= count {
		return 0
	}
	return capAt(satMul(nonneg(perTick), nonneg(tp)), maxMoney)
}
