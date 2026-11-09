package denomination_test

import (
	"errors"
	"fmt"

	"github.com/Whealinghold/economy-sim/denomination"
)

func ExampleFormatter_Format() {
	for _, p := range []int{0, 2} {
		f, err := denomination.NewFormatter(denomination.RPG(), denomination.WithPrecision(p))
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(f.Format(12_345))
	}
	// Output:
	// 1 gp 2 sp 3 cp
	// 1 gp 2 sp 3.45 cp
}

func ExampleNewFormatter() {
	// A ladder must be strictly decreasing; the error names the unit and wraps a sentinel.
	_, err := denomination.NewFormatter(denomination.Ladder{
		{Name: "Small", Symbol: "s", Size: 1},
		{Name: "Large", Symbol: "l", Size: 12},
	})
	fmt.Println(errors.Is(err, denomination.ErrUnitOrder))

	f, _ := denomination.NewFormatter(denomination.Ladder{
		{Name: "Large", Symbol: "L", Size: 240},
		{Name: "Medium", Symbol: "M", Size: 12},
		{Name: "Small", Symbol: "S", Size: 1},
	})
	fmt.Println(f.Format(500))
	// Output:
	// true
	// 2 L 1 M 8 S
}
