package indicator

import (
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
)

// Classes is the table of the class tags of the commodities: for each class
// name, the sorted set of commodity IDs that carry it
// (spec-indicators.md 4.4). The zero value has no class.
type Classes struct {
	names []string
	sets  [][]economysim.CommodityID
	all   []economysim.CommodityID
}

// NewClasses builds the table from a configuration as World.Config returns it,
// whose commodity order is the order of the CommodityIDs. No class name is
// built in: the names are the example pack's data (architecture.md 9.1).
func NewClasses(cfg economysim.Config) Classes {
	var c Classes
	for i, cm := range cfg.Commodities {
		id := economysim.CommodityID(i)
		c.all = append(c.all, id)
		for _, name := range cm.Classes {
			j, found := slices.BinarySearch(c.names, name)
			if !found {
				c.names = slices.Insert(c.names, j, name)
				c.sets = slices.Insert(c.sets, j, nil)
			}
			c.sets[j] = append(c.sets[j], id) // IDs ascend, so the set stays sorted
		}
	}
	return c
}

// Of returns the sorted commodities that carry the class, as a copy, or
// ErrUnknownClass when none does.
func (c Classes) Of(name string) ([]economysim.CommodityID, error) {
	j, found := slices.BinarySearch(c.names, name)
	if !found {
		return nil, wrap("Classes.Of", name, ErrUnknownClass)
	}
	return slices.Clone(c.sets[j]), nil
}

// All returns every commodity once, in ID order, as a copy. A commodity of two
// classes is in it once: a measure over all commodities never sums class
// measures.
func (c Classes) All() []economysim.CommodityID { return slices.Clone(c.all) }

// Names returns the class names in ascending order.
func (c Classes) Names() []string { return slices.Clone(c.names) }
