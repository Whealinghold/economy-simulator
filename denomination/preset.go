package denomination

// RPG returns the fantasy-RPG ladder: gold (gp, 10 000 minor units), silver (sp,
// 1 000) and copper (cp, 100), largest first. Each call returns a fresh slice.
func RPG() Ladder {
	return Ladder{
		{Name: "Gold", Symbol: "gp", Size: 10_000},
		{Name: "Silver", Symbol: "sp", Size: 1_000},
		{Name: "Copper", Symbol: "cp", Size: 100},
	}
}

// Single returns a ladder of one Unit of size 1 with the given name and symbol,
// which formats every amount as a plain count. Each call returns a fresh slice.
func Single(name, symbol string) Ladder {
	return Ladder{{Name: name, Symbol: symbol, Size: 1}}
}
