package lexl

import (
	"fmt"
	"testing"
)

func TestInterval(t *testing.T) {

	iv := []*characterRange{
		&characterRange{least: 5, greatest: 10},
		&characterRange{least: 2, greatest: 4},
		&characterRange{least: 6, greatest: 11},
		&characterRange{least: 180, greatest: 190},
		&characterRange{least: 13, greatest: 20},
		&characterRange{least: 18, greatest: 23},
	}

	iv = regularizeBoundedIntervals(iv)

	for _, r := range iv {
		fmt.Printf("(%d,%d)\n", r.least, r.greatest)
	}

	fmt.Println("---")
	iv = invertRegularizedIntervals(iv)

	for _, r := range iv {
		fmt.Printf("(%d,%d)\n", r.least, r.greatest)
	}
	
	ivs := []*interval{
		&interval{4, 8, 5, "A"},
		&interval{6, 10, 8, "B"},
		&interval{2, 20, 1, "C"},
	}
	
	ivs = resolveIntervals(ivs)
	for i, r := range ivs {
		var name string
		if str, ok := r.data.(string); ok {
			name = str
		} else {
			name = "*"
		}
		fmt.Printf("%d: {%s: %d - %d (%d)}\n", i, name, r.first, r.last, r.priority)
	}
}
