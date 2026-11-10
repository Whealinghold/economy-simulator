package otelobserver

import (
	economysim "github.com/Whealinghold/economy-sim"
	"go.opentelemetry.io/otel/attribute"
)

// maxReasons is the most rejection reasons the hot accumulators hold; a reason
// beyond it drops the sample (plan 2-22 D646).
const maxReasons = 32

// names is the names table of the world, as attributes: one slice per kind of
// ID, indexed by the ID. It is a copy, built once at WorldCreated or
// WorldRestored, so a later change to the event's slices changes nothing here.
type names struct {
	commodity []attribute.KeyValue
	region    []attribute.KeyValue
	role      []attribute.KeyValue
	recipe    []attribute.KeyValue
	want      []attribute.KeyValue
	resource  []attribute.KeyValue
	mode      []attribute.KeyValue
	reasons   intern

	// tab is the attribute of every place of each key except economysim.reason,
	// whose open values are in other (cells.go).
	tab   map[string][]attribute.KeyValue
	other intern
}

// newNames builds the attribute table of the names of an event.
func newNames(n economysim.Names) names {
	strs := map[string][]string{keyCommodity: n.Commodities, keyRegion: n.Regions, keyRole: n.Roles,
		keyRecipe: n.Recipes, keyWant: n.Wants, keyResource: n.Resources, keyMode: n.Modes}
	nm := names{
		commodity: kvs(keyCommodity, n.Commodities),
		region:    kvs(keyRegion, n.Regions),
		role:      kvs(keyRole, n.Roles),
		recipe:    kvs(keyRecipe, n.Recipes),
		want:      kvs(keyWant, n.Wants),
		resource:  kvs(keyResource, n.Resources),
		mode:      kvs(keyMode, n.Modes),
		reasons:   newIntern(keyReason, maxReasons),
		tab:       map[string][]attribute.KeyValue{},
		other:     newIntern(keyReason, maxInterned),
	}
	for i := range catalogue {
		for _, key := range catalogue[i].attrs {
			if _, ok := nm.tab[key]; !ok && key != keyReason {
				nm.tab[key] = attrTable(key, strs)
			}
		}
	}
	return nm
}

func kvs(key string, ss []string) []attribute.KeyValue {
	out := make([]attribute.KeyValue, len(ss))
	for i, s := range ss {
		out[i] = attribute.String(key, s)
	}
	return out
}

// intern gives a dense index to each distinct value of an open string, up to a
// cap, and the attribute of each.
type intern struct {
	key string
	max int
	idx map[string]int
	kv  []attribute.KeyValue
}

func newIntern(key string, limit int) intern {
	return intern{key: key, max: limit, idx: map[string]int{}}
}

// id returns the index of s, adding it if there is room, or -1 beyond the cap.
func (t *intern) id(s string) int {
	if i, ok := t.idx[s]; ok {
		return i
	}
	if len(t.kv) >= t.max {
		return -1
	}
	t.idx[s] = len(t.kv)
	t.kv = append(t.kv, attribute.String(t.key, s))
	return len(t.kv) - 1
}

// sideAttr is the attribute of a side of the book, indexed by economysim.Side.
var sideAttr = [2]attribute.KeyValue{
	attribute.String(keySide, economysim.Bid.String()),
	attribute.String(keySide, economysim.Ask.String()),
}
