package economysim

import (
	"fmt"
	"log/slog"
	"slices"
)

// Option configures a world. Every option validates its argument and returns
// an error rather than panics; NewWorld applies them in the order given and
// returns the first error before any world state is built (spec-world.md 4).
type Option func(*options) error

// options is what the options of NewWorld set. It is unexported
// (architecture.md 6.2). The zero value is not the default: build it with
// defaultOptions.
type options struct {
	logger            *slog.Logger
	observers         observers
	audit             bool
	targetSpread      int64
	targetSpreadGiven bool
	seed              uint64
	clearings         registry[Clearing]
}

// defaultOptions returns the options of a world built with no option: the
// discard logger, no observer, no audit, no target spread given, seed 0.
func defaultOptions() options {
	return options{logger: discardLogger()}
}

// discardLogger is the default logger, slog.New(slog.DiscardHandler)
// (architecture.md 11.2).
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// applyOptions applies opts in order to the default options and returns the
// first error. A nil Option is skipped.
func applyOptions(opts []Option) (options, error) {
	o := defaultOptions()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&o); err != nil {
			return options{}, err
		}
	}
	return o, nil
}

// WithLogger sets the logger the world logs through. A nil logger means the
// default, which discards every record; it is not an error. The library never
// falls back to slog.Default.
func WithLogger(l *slog.Logger) Option {
	return func(o *options) error {
		if l == nil {
			l = discardLogger()
		}
		o.logger = l
		return nil
	}
}

// WithObserver adds an observer. It is additive: it may be given more than
// once, and every event reaches each observer in the order the options were
// given. A nil observer is ignored. With no observer the world builds no
// event.
func WithObserver(ob Observer) Option {
	return func(o *options) error {
		o.observers = newObservers(append(slices.Clone(o.observers.list), ob))
		return nil
	}
}

// WithAudit turns the audit on: a violated invariant becomes an error of the
// step where there would be a panic. It changes no result of a world that
// holds its invariants and is not part of the configuration hash.
func WithAudit() Option {
	return func(o *options) error {
		o.audit = true
		return nil
	}
}

// WithTargetSpread overrides the TargetSpread of every role with bp, in basis
// points from 0 to 9 999; any other value is an error that wraps
// ErrOutOfBounds. A given 0 is not the same as not given: the override then
// sets every spread to 0. The last one given wins. The value is part of the
// effective configuration and of its hash.
func WithTargetSpread(bp int64) Option {
	return func(o *options) error {
		// The bound of Role.TargetSpread: 0 to 9 999 (spec-welfare-and-wages.md 12).
		if bp < 0 || bp > wholeBP-1 {
			return fmt.Errorf("target spread %d outside 0 to %d: %w", bp, wholeBP-1, ErrOutOfBounds)
		}
		o.targetSpread = bp
		o.targetSpreadGiven = true
		return nil
	}
}

// WithSeed sets the seed of the world's random streams. The default is 0, so a
// world built with no option is still reproducible; the last one given wins.
// Every draw is a function of the seed and of nothing else, so a seed is a
// reproducible stream by design: it is not for secrets, and a consumer must
// not use it where unpredictability matters.
func WithSeed(seed uint64) Option {
	return func(o *options) error {
		o.seed = seed
		return nil
	}
}

// registry is the list of the extensions registered under one kind of name, in
// the order given. A slice searched linearly keeps that order and needs no map
// (architecture.md 7.5). The strategy, spawner and policy options reuse it.
type registry[T any] struct {
	names []string
	vals  []T
}

// add registers v under name. The checks run in the order of spec-world.md
// 4.1.1: a name that is not an ID is ErrInvalidID, a name already given or in
// builtin is ErrDuplicateID, and isNil (the caller's nil test of v, which a
// type parameter cannot make) is ErrNilExtension.
func (r *registry[T]) add(name string, v T, isNil bool, builtin ...string) error {
	if !validID(name) {
		return fmt.Errorf("registration name %q: %w", name, ErrInvalidID)
	}
	if slices.Contains(builtin, name) || slices.Contains(r.names, name) {
		return fmt.Errorf("registration name %q: %w", name, ErrDuplicateID)
	}
	if isNil {
		return fmt.Errorf("registration %q: %w", name, ErrNilExtension)
	}
	r.names = append(r.names, name)
	r.vals = append(r.vals, v)
	return nil
}

// find returns the extension registered under name.
func (r *registry[T]) find(name string) (v T, ok bool) {
	if i := slices.Index(r.names, name); i >= 0 {
		return r.vals[i], true
	}
	return v, false
}

// averageName is the name of the built-in clearing, AverageClearing.
const averageName = "average"

// WithClearing registers c under name, so that World.Clearing may name it. It
// is additive. A name that is not an ID is ErrInvalidID, a name given twice or
// "average" (the built-in AverageClearing) is ErrDuplicateID and a nil c is
// ErrNilExtension. The name is part of the configuration hash and the code
// behind it is not.
func WithClearing(name string, c Clearing) Option {
	return func(o *options) error {
		return o.clearings.add(name, c, c == nil, averageName)
	}
}

// resolveClearing returns the clearing a configured name selects: the built-in
// for "average", else the registered one. An empty or unknown name is a
// *ValidationError at path with rule ErrUnknownClearing and the name as value
// (spec-config.md 7.1.4). NewWorld calls it once, after Validate succeeds.
//
// It returns the Clearing interface (architecture.md 4.4) because the result is
// a consumer-registered extension: the concrete type is unknowable here.
func resolveClearing(o *options, name string, path []string) (Clearing, error) {
	if name == averageName {
		return AverageClearing{}, nil
	}
	if c, ok := o.clearings.find(name); ok {
		return c, nil
	}
	return nil, &ValidationError{Path: path, Rule: ErrUnknownClearing, Value: name}
}
