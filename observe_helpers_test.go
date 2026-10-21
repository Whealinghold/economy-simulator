package economysim_test

import (
	"context"
	"log/slog"
	"sync"

	economysim "github.com/Whealinghold/economy-sim"
)

// recorder is the recording Observer of the event tests (observability-and-logging.md
// 8.1): it keeps every event it is given, in order. It is a hand-written fake at
// an extension point (architecture.md 10.3).
type recorder struct {
	events []economysim.Event
}

// Observe keeps e.
func (r *recorder) Observe(e economysim.Event) { r.events = append(r.events, e) }

// forget drops the events kept so far. A test of the step calls it after
// NewWorld, whose events (WorldCreated and the AgentSpawned of each agent) have
// tests of their own.
func (r *recorder) forget() { r.events = nil }

// logLine is one record the captureHandler kept: the level, the message and the
// attributes, those of WithAttrs first.
type logLine struct {
	level slog.Level
	msg   string
	attrs []slog.Attr
}

// logSink is the list of records that a captureHandler and every handler
// derived from it write to.
type logSink struct {
	mu    sync.Mutex
	lines []logLine
}

// captureHandler is the capturing slog.Handler of the log tests: Enabled is
// true for every level, so a line behind logger.Enabled is built, and every
// record is kept in the order it was written.
type captureHandler struct {
	sink  *logSink
	attrs []slog.Attr
}

// newCaptureHandler returns an empty capturing handler.
func newCaptureHandler() *captureHandler { return &captureHandler{sink: &logSink{}} }

// Enabled is true for every level.
func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

// Handle keeps the record.
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	line := logLine{level: r.Level, msg: r.Message, attrs: append([]slog.Attr(nil), h.attrs...)}
	r.Attrs(func(a slog.Attr) bool {
		line.attrs = append(line.attrs, a)
		return true
	})
	h.sink.mu.Lock()
	h.sink.lines = append(h.sink.lines, line)
	h.sink.mu.Unlock()
	return nil
}

// WithAttrs returns a handler that adds attrs to every record.
func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &captureHandler{sink: h.sink, attrs: append(append([]slog.Attr(nil), h.attrs...), attrs...)}
}

// WithGroup returns h: the tests use no group.
func (h *captureHandler) WithGroup(string) slog.Handler { return h }

// forget drops the records kept so far (see recorder.forget).
func (h *captureHandler) forget() {
	h.sink.mu.Lock()
	h.sink.lines = nil
	h.sink.mu.Unlock()
}

// records returns a copy of the records kept so far.
func (h *captureHandler) records() []logLine {
	h.sink.mu.Lock()
	defer h.sink.mu.Unlock()
	return append([]logLine(nil), h.sink.lines...)
}

// attr returns the value of the attribute with the key, and false if the line
// has none.
func (l logLine) attr(key string) (slog.Value, bool) {
	for _, a := range l.attrs {
		if a.Key == key {
			return a.Value, true
		}
	}
	return slog.Value{}, false
}

// levelHandler is a slog.Handler that reports Enabled only for the levels at or
// above min and counts the records it is given: the logger of a consumer that
// leaves Debug off, to show that no Debug record reaches it.
type levelHandler struct {
	min     slog.Level
	handled int
}

// Enabled reports whether level is at or above min.
func (h *levelHandler) Enabled(_ context.Context, level slog.Level) bool { return level >= h.min }

// Handle counts the record.
func (h *levelHandler) Handle(context.Context, slog.Record) error { h.handled++; return nil }

// WithAttrs returns h: the tests use no attributes.
func (h *levelHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

// WithGroup returns h: the tests use no group.
func (h *levelHandler) WithGroup(string) slog.Handler { return h }
