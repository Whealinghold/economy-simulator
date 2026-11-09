package denomination_test

import (
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
