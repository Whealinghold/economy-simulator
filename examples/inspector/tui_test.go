package main

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// testModel builds a model on the fixture pack with a 50-tick event window.
func testModel(t *testing.T) *model {
	t.Helper()
	m, err := newModel(options{chains: "testdata/chains", pack: "one", seed: 1, win: 50, every: 100, ticks: 0})
	if err != nil {
		t.Fatalf("newModel() error = %v", err)
	}
	return m
}

// press sends a key to the model, as the terminal would.
func press(m *model, k string) tea.Cmd {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	switch k {
	case " ":
		msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	}
	_, cmd := m.Update(msg)
	return cmd
}

// TestTUI_StepAndGoBackIsExact verifies that stepping 10 ticks moves the
// clock, and that going back and forward again gives the same snapshot as the
// first time.
//
// Guard: going back works by building the world again from the seed. If any
// state survived the rebuild, or the run were not deterministic, the screen
// would show a different economy for the same tick.
func TestTUI_StepAndGoBackIsExact(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	m := testModel(t)
	for range 3 {
		press(m, "S")
	}
	if got := m.s.tick(); got != 30 {
		t.Fatalf("after three S presses tick = %d, want 30", got)
	}
	want := m.snap
	press(m, "b") // back 10: tick 20
	if got := m.s.tick(); got != 20 {
		t.Fatalf("after b tick = %d, want 20", got)
	}
	press(m, "S") // forward 10: tick 30
	if !reflect.DeepEqual(m.snap, want) {
		t.Errorf("snapshot at tick 30 differs after going back and forward:\ngot  %+v\nwant %+v", m.snap, want)
	}
}

// TestTUI_RunStepsBySpeedAndPauses verifies that space starts a run, each
// frame advances by the chosen speed, and a second space stops it.
//
// Guard: the frame loop is the only place time passes in the UI; a frame that
// kept scheduling itself after a pause would keep stepping a paused world.
func TestTUI_RunStepsBySpeedAndPauses(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	m := testModel(t)
	if cmd := press(m, " "); cmd == nil || !m.running {
		t.Fatalf("space did not start a run (running %v, cmd nil %v)", m.running, cmd == nil)
	}
	_, cmd := m.Update(frameMsg{})
	if got, want := m.s.tick(), speeds[m.speed]; got != want {
		t.Errorf("one frame stepped to tick %d, want %d", got, want)
	}
	if cmd == nil {
		t.Errorf("a running frame returned no next frame")
	}
	press(m, " ")
	before := m.s.tick()
	if _, cmd := m.Update(frameMsg{}); cmd != nil || m.s.tick() != before {
		t.Errorf("a frame after pause stepped (tick %d, was %d) or scheduled another", m.s.tick(), before)
	}
}

// TestTUI_FreezeAndSeedKeepTheTick verifies that toggling freeze or changing
// the seed rebuilds the world and returns to the tick it was at, and that the
// header shows the change.
//
// Guard: a restart that went back to tick 0 would lose the reader's place on
// every what-if.
func TestTUI_FreezeAndSeedKeepTheTick(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	m := testModel(t)
	press(m, "S")
	press(m, "f")
	if m.s.tick() != 10 || !strings.Contains(m.View(), "FROZEN") {
		t.Errorf("after f: tick %d, view lacks FROZEN:\n%s", m.s.tick(), m.View())
	}
	press(m, "]")
	if m.s.tick() != 10 || !strings.Contains(m.View(), "seed 2") {
		t.Errorf("after ]: tick %d, view lacks seed 2", m.s.tick())
	}
	press(m, "[")
	press(m, "[") // the seed does not go below 1
	if !strings.Contains(m.View(), "seed 1") {
		t.Errorf("after [ [ the view lacks seed 1")
	}
}

// TestTUI_ViewsShowTheirContent verifies the three views: the overview has the
// role table and the price history, the events view the counters of the last
// window, and the market view the best bid and ask of the selected market.
//
// Guard: each view reads a different part of the session; one that read a
// stale or empty source would render an empty screen without an error.
func TestTUI_ViewsShowTheirContent(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	m := testModel(t)
	press(m, "B") // back 100 at tick 0 stays at 0
	for range 5 {
		press(m, "S")
	}
	tests := []struct {
		key  string
		want []string
	}{
		{"1", []string{"cook", "grower", "price history", "staple"}},
		{"2", []string{"events of the window before tick 50", "exec."}},
		{"3", []string{"best bid", "price, last"}},
	}
	for _, tt := range tests {
		press(m, tt.key)
		v := m.View()
		for _, w := range tt.want {
			if !strings.Contains(v, w) {
				t.Errorf("view %s lacks %q:\n%s", tt.key, w, v)
			}
		}
	}
	press(m, "2")
	press(m, "e") // the "deaths and moves" filter hides the production counters
	if strings.Contains(m.View(), "exec.") {
		t.Errorf("filtered events view still shows exec counters")
	}
}

// TestTUI_NewModelFailsOnABadPack verifies that a pack that cannot be loaded
// is an error from newModel and, through the command line, exit code 1.
//
// Guard: the terminal must not be taken over before the world exists.
func TestTUI_NewModelFailsOnABadPack(t *testing.T) {
	if _, err := newModel(options{chains: "testdata/chains", pack: "absent", win: 10}); err == nil {
		t.Errorf("newModel(absent pack) error = nil, want an error")
	}
	if code, _, _ := runArgs(t, "-pack", "absent", "-tui"); code != 1 {
		t.Errorf("run(-tui, absent pack) = %d, want 1", code)
	}
}

// TestSparkAndBar verifies the two small drawing helpers at their edges.
func TestSparkAndBar(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"empty spark", spark(nil, 5), ""},
		{"flat spark", spark([]int64{7, 7, 7}, 5), "▁▁▁"},
		{"rising spark", spark([]int64{0, 7}, 5), "▁█"},
		{"spark keeps the last width values", spark([]int64{9, 0, 7}, 2), "▁█"},
		{"bar with no money", bar(0, 100, 4), "····   0%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
	if got := bar(50, 100, 4); !strings.Contains(got, " 50%") {
		t.Errorf("bar(50, 100, 4) = %q, want it to end with 50%%", got)
	}
}

// TestTUI_HelpScreenAndTabBar verifies that the first screen names the three
// views with their keys and points to the help, that ? opens a help screen
// listing the keys, that the key that closes it does nothing else, and that q
// quits from it.
//
// Guard: the keys are not guessable (a reader met "press 2" only in the
// README). If the tab bar or the help were dropped, the program would have no
// on-screen way to learn them.
func TestTUI_HelpScreenAndTabBar(t *testing.T) {
	m := testModel(t)
	for _, w := range []string{"1 overview", "2 events", "3 market", "? help"} {
		if !strings.Contains(m.View(), w) {
			t.Errorf("first screen lacks %q:\n%s", w, m.View())
		}
	}
	for _, k := range []string{"?", "h"} {
		press(m, k)
		v := m.View()
		for _, w := range []string{"inspector help", "space", "freeze", "1 overview", "any key closes"} {
			if !strings.Contains(v, w) {
				t.Errorf("after %q the help lacks %q:\n%s", k, w, v)
			}
		}
		if cmd := press(m, "S"); cmd != nil || m.help || m.s.tick() != 0 {
			t.Errorf("the key that closes the help acted: help %v, tick %d", m.help, m.s.tick())
		}
	}
	press(m, "?")
	if cmd := press(m, "q"); cmd == nil {
		t.Errorf("q from the help screen did not quit")
	}
}
