package otelobserver_test

import (
	"reflect"
	"strings"

	economysim "github.com/Whealinghold/economy-sim"
)

// filler builds an event with every field set: an ID gets the index idx, a
// number 3, a string one of a small pool (valid constants and nonsense), a slice
// one element. It lets a test send every type of event without a table of fields
// that a new field would silently outgrow.
type filler struct {
	idx int
	n   int
}

var stringPool = []string{"insolvent", "starved", "population", "transport", "migration", "consumer", "no_cash",
	"stipend", "faucet", "sink", "bogus", "units", "value"}

// number is the value of an integer field of the named type.
func (f *filler) number(name string) int64 {
	switch {
	case name == "Side":
		return int64((f.idx%2 + 2) % 2)
	case strings.HasSuffix(name, "ID"):
		return int64(f.idx)
	}
	return 3
}

func (f *filler) fill(v reflect.Value) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(f.number(v.Type().Name()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
	case reflect.String:
		f.n++
		v.SetString(stringPool[f.n%len(stringPool)])
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		f.fill(s.Index(0))
		v.Set(s)
	case reflect.Array:
		for i := range v.Len() {
			f.fill(v.Index(i))
		}
	case reflect.Struct:
		for i := range v.NumField() {
			f.fill(v.Field(i))
		}
	}
}

// filled is a copy of the event e with every field set by f.
func (f *filler) filled(e economysim.Event) economysim.Event {
	v := reflect.New(reflect.TypeOf(e)).Elem()
	f.fill(v)
	return v.Interface().(economysim.Event)
}
