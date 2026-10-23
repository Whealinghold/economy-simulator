package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// speeds are the ticks stepped per frame while running.
var speeds = []int64{1, 2, 5, 10, 25}

// eventFilters are the presets of the events view, cycled with "e". An empty
// prefix list shows every counter.
var eventFilters = []struct {
	name     string
	prefixes []string
}{
	{"all", nil},
	{"deaths and moves", []string{"dead", "cut", "move", "entry"}},
	{"shortfalls", []string{"short"}},
	{"production", []string{"exec", "aband", "refuse"}},
	{"trade", []string{"vol", "val", "rev", "spend"}},
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	headStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
	badStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	goodStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	tabOn      = lipgloss.NewStyle().Bold(true).Reverse(true).Padding(0, 1)
	tabOff     = lipgloss.NewStyle().Padding(0, 1)
)

// tabNames are the three views, in the order of the keys 1 2 3.
var tabNames = [...]string{"overview", "events", "market"}

type frameMsg struct{}

// model is the bubbletea model: the session it steps and what the screen
// shows. All stepping happens in Update, on the UI goroutine, so the world
// is never used from two goroutines.
type model struct {
	s       *session
	snap    snapOut
	running bool
	speed   int // index into speeds
	tab     int // 0 overview, 1 events, 2 market
	help    bool
	sel     int // selected market row
	filter  int // index into eventFilters
	err     error
	width   int
	height  int
}

func newModel(o options) (*model, error) {
	s, err := newSession(o)
	if err != nil {
		return nil, err
	}
	m := &model{s: s, speed: 2, width: 100, height: 40}
	return m, m.refresh()
}

// refresh reads the snapshot into the model.
func (m *model) refresh() error {
	_, so, err := m.s.view()
	if err != nil {
		return err
	}
	m.snap = so
	return nil
}

// Init implements tea.Model.
func (m *model) Init() tea.Cmd { return nil }

func frame() tea.Cmd {
	return tea.Tick(40*time.Millisecond, func(time.Time) tea.Msg { return frameMsg{} })
}

// move steps or seeks and refreshes; any error stops the run and is shown.
func (m *model) move(f func() error) {
	if err := f(); err != nil {
		m.err, m.running = err, false
		return
	}
	m.err = m.refresh()
}

// Update implements tea.Model.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case frameMsg:
		if !m.running {
			return m, nil
		}
		m.move(func() error { return m.s.advance(speeds[m.speed]) })
		return m, frame()
	case tea.KeyMsg:
		return m.key(msg.String())
	}
	return m, nil
}

func (m *model) key(k string) (tea.Model, tea.Cmd) {
	t := m.s.tick()
	if m.help { // any key closes the help; q still quits
		m.help = false
		if k != "q" && k != "ctrl+c" {
			return m, nil
		}
	}
	switch k {
	case "?", "h":
		m.help = true
	case "q", "ctrl+c":
		return m, tea.Quit
	case " ":
		m.running = !m.running
		if m.running {
			return m, frame()
		}
	case "s", "right":
		m.move(func() error { return m.s.advance(1) })
	case "S":
		m.move(func() error { return m.s.advance(10) })
	case "b", "left":
		m.move(func() error { return m.s.seek(t - 10) })
	case "B":
		m.move(func() error { return m.s.seek(t - 100) })
	case "+", "=":
		m.speed = min(m.speed+1, len(speeds)-1)
	case "-":
		m.speed = max(m.speed-1, 0)
	case "r":
		m.move(func() error { return m.s.seek(0) })
	case "]":
		m.s.o.seed++
		m.restart(t)
	case "[":
		m.s.o.seed = max(m.s.o.seed, 2) - 1
		m.restart(t)
	case "f":
		m.s.o.freeze = !m.s.o.freeze
		m.restart(t)
	case "tab":
		m.tab = (m.tab + 1) % 3
	case "1", "2", "3":
		m.tab = int(k[0] - '1')
	case "e":
		m.filter = (m.filter + 1) % len(eventFilters)
	case "up", "k":
		m.sel = max(m.sel-1, 0)
	case "down", "j":
		m.sel = min(m.sel+1, max(len(m.snap.Markets)-1, 0))
	}
	return m, nil
}

// restart builds the world again after a seed or freeze change and returns
// to tick t.
func (m *model) restart(t int64) {
	m.move(func() error {
		if err := m.s.reset(); err != nil {
			return err
		}
		return m.s.advance(t)
	})
}

// View implements tea.Model.
func (m *model) View() string {
	if m.help {
		return helpText()
	}
	var b strings.Builder
	state := "paused"
	if m.running {
		state = fmt.Sprintf("running x%d", speeds[m.speed])
	}
	frozen := ""
	if m.s.o.freeze {
		frozen = "  FROZEN"
	}
	so := m.snap
	unfed := fmt.Sprintf("%d unfed", so.Unfed)
	if so.Unfed > 0 {
		unfed = badStyle.Render(unfed)
	}
	fmt.Fprintf(&b, "%s  tick %d  seed %d%s  %s\n", titleStyle.Render("inspector"), so.Tick, m.s.o.seed, frozen, state)
	fmt.Fprintf(&b, "households %d  %s  money %d (estates %d)\n", so.Households, unfed, so.Money, so.Pool)
	b.WriteString(m.tabBar() + "\n\n")
	switch m.tab {
	case 0:
		m.overview(&b)
	case 1:
		m.events(&b)
	default:
		m.market(&b)
	}
	if m.err != nil {
		fmt.Fprintf(&b, "\n%s\n", badStyle.Render("error: "+m.err.Error()))
	}
	fmt.Fprintf(&b, "\n%s\n", dimStyle.Render("space run  s step  b back  1 2 3 views  f freeze  ? help  q quit"))
	return b.String()
}

// tabBar draws the three views with their keys and marks the open one.
func (m *model) tabBar() string {
	parts := make([]string, 0, len(tabNames)+1)
	for i, n := range tabNames {
		label := fmt.Sprintf("%d %s", i+1, n)
		if i == m.tab {
			parts = append(parts, tabOn.Render(label))
		} else {
			parts = append(parts, tabOff.Render(label))
		}
	}
	parts = append(parts, tabOff.Render(dimStyle.Render("? help")))
	return strings.Join(parts, " ")
}

// helpText is the help screen: every key, and what each view shows.
func helpText() string {
	return titleStyle.Render("inspector help") + `

Keys
  space        run or pause           + / -     ticks stepped per frame
  s / S        step 1 / 10 ticks      b / B     go back 10 / 100 ticks
  r            back to tick 0         [ / ]     previous / next seed
  f            freeze or unfreeze role change and deaths (same tick)
  1 2 3, tab   overview, events, market view
  up / down    pick the market (market view, overview cursor)
  e            cycle the event filter (events view)
  ? or h       this help (any key closes it)       q   quit

Views
  1 overview   one row per role: households, unfed, welfare ratio, money and its
               share of all money; one row per market: price, volume, stock,
               unmet asks, cost anchor and a price-history sparkline
  2 events     the counters of the last complete window (deaths, role moves,
               upkeep shortfalls, recipe runs, trade); e narrows them
  3 market     the selected market: best bid and ask, unmet bid and ask, stock
               held by agents and by estates, and the price history

Tips
  - Going back builds the world again from the seed and steps to the tick, so it
    takes as long as running there again.
  - Freeze (f) shows whether the markets fail on their own: with no role change
    or deaths, unfed households point at production and trade, not the spawner.
  - Changing the seed or freezing returns to the tick you were on.

` + dimStyle.Render("press any key to go back")
}

func (m *model) overview(b *strings.Builder) {
	fmt.Fprintln(b, headStyle.Render(fmt.Sprintf("%-10s %6s %6s %9s %9s %12s %10s  %s", "role", "hh", "unfed", "ratioBP", "expectBP", "money", "money/hh", "share of money")))
	for _, r := range m.snap.Roles {
		if r.Households == 0 && r.Money == 0 {
			continue
		}
		unfed := fmt.Sprintf("%6d", r.Unfed)
		if r.Unfed > 0 {
			unfed = badStyle.Render(unfed)
		}
		fmt.Fprintf(b, "%-10s %6d %s %9d %9d %12d %10d  %s\n", r.Role, r.Households, unfed, r.MeanRatioBP, r.ExpectedRatioBP, r.Money, r.MoneyPerHousehold, bar(r.Money, m.snap.Money, 24))
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, headStyle.Render(fmt.Sprintf("  %-10s %7s %7s %9s %9s %7s  %s", "commodity", "price", "volume", "stock", "unmetAsk", "anchor", "price history")))
	for i, mk := range m.snap.Markets {
		cur := "  "
		if i == m.sel {
			cur = "> "
		}
		fmt.Fprintf(b, "%s%-10s %7d %7d %9d %9d %7d  %s\n", cur, mk.Commodity, mk.Price, mk.Volume, mk.HeldByAgents, mk.UnmatchedAsk, mk.CostAnchor, spark(m.history(i), 40))
	}
}

// history is the price series of the i-th market of the snapshot.
func (m *model) history(i int) []int64 {
	if i < 0 || i >= len(m.snap.Markets) {
		return nil
	}
	for k, series := range m.s.tr.prices {
		mk := m.snap.Markets[i]
		if m.s.tr.names.Commodities[k.commodity] == mk.Commodity && m.s.tr.names.Regions[k.region] == mk.Region {
			return series
		}
	}
	return nil
}

func (m *model) events(b *strings.Builder) {
	f := eventFilters[m.filter]
	fmt.Fprintf(b, "%s\n", headStyle.Render(fmt.Sprintf("events of the window before tick %d (filter: %s)", m.s.lastAt, f.name)))
	n := 0
	for _, kv := range m.s.last {
		key := kv[:strings.LastIndexByte(kv, '=')]
		if !keep(key, f.prefixes) {
			continue
		}
		n++
		fmt.Fprintf(b, "  %s\n", kv)
		if n >= max(m.height-10, 5) {
			fmt.Fprintf(b, "  %s\n", dimStyle.Render("... more; press e for a narrower filter"))
			break
		}
	}
	if n == 0 {
		fmt.Fprintln(b, dimStyle.Render("  nothing yet: a window closes every"), m.s.win, "ticks")
	}
}

func (m *model) market(b *strings.Builder) {
	if m.sel >= len(m.snap.Markets) {
		return
	}
	mk := m.snap.Markets[m.sel]
	fmt.Fprintf(b, "%s\n", headStyle.Render(fmt.Sprintf("%s in %s", mk.Commodity, mk.Region)))
	fmt.Fprintf(b, "  price %d   anchor %d   volume %d\n", mk.Price, mk.CostAnchor, mk.Volume)
	fmt.Fprintf(b, "  best bid %d   best ask %d   unmet bid %d   unmet ask %d\n", mk.BestBid, mk.BestAsk, mk.UnmatchedBid, mk.UnmatchedAsk)
	fmt.Fprintf(b, "  stock: agents %d   estate %d\n\n", mk.HeldByAgents, mk.HeldByEstate)
	h := m.history(m.sel)
	w := max(m.width-4, 20)
	if len(h) > w { // the range is of the part drawn
		h = h[len(h)-w:]
	}
	fmt.Fprintf(b, "  price, last %d ticks\n  %s\n", len(h), spark(h, w))
	if len(h) > 0 {
		lo, hi := h[0], h[0]
		for _, v := range h {
			lo, hi = min(lo, v), max(hi, v)
		}
		fmt.Fprintf(b, "  low %d  high %d\n", lo, hi)
	}
}

// spark draws the last width values of s as a line of block characters.
func spark(s []int64, width int) string {
	if len(s) > width {
		s = s[len(s)-width:]
	}
	if len(s) == 0 {
		return ""
	}
	lo, hi := s[0], s[0]
	for _, v := range s {
		lo, hi = min(lo, v), max(hi, v)
	}
	const blocks = "▁▂▃▄▅▆▇█"
	r := []rune(blocks)
	var b strings.Builder
	for _, v := range s {
		i := 0
		if hi > lo {
			i = int((v - lo) * int64(len(r)-1) / (hi - lo))
		}
		b.WriteRune(r[i])
	}
	return b.String()
}

// bar draws part of whole as width cells, with the share written after it.
func bar(part, whole int64, width int) string {
	if whole <= 0 || part <= 0 {
		return strings.Repeat("·", width) + "   0%"
	}
	n := int(part * int64(width) / whole)
	return goodStyle.Render(strings.Repeat("█", n)) + strings.Repeat("·", width-n) + fmt.Sprintf(" %3d%%", part*100/whole)
}

// runTUI shows the model on the terminal until the user quits.
func runTUI(o options) error {
	m, err := newModel(o)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
