package saving

// Precaution returns the buffer of spec 4.2: min(capTicks * upkeep,
// share * drawdown / 10 000), 0 at a share of 0. A negative argument counts as
// 0. The products are in 128 bits and the quotient truncates toward zero.
func Precaution(share, capTicks, upkeep, drawdown int64) int64 {
	share, drawdown = nonneg(share), nonneg(drawdown)
	if share == 0 {
		return 0
	}
	ceiling := satMul(nonneg(capTicks), nonneg(upkeep))
	b := satMulDiv(share, drawdown, scale)
	if b > ceiling {
		return ceiling
	}
	return b
}

// Floor returns max(reserve, precaution): the flat reserve is the floor of the
// buffer, which only ever raises the target (spec 4.2).
func Floor(reserve, precaution int64) int64 {
	if precaution > reserve {
		return precaution
	}
	return reserve
}

// Target returns S* = floor + goal (spec 3.2.2), saturating.
func Target(floor, goal int64) int64 { return satAdd(floor, nonneg(goal)) }
