package config

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"

	economysim "github.com/Whealinghold/economy-sim"
)

// guard is the guard pass of spec-config.md 6.3.1 and ADR 0009: it checks one
// document's line length, the nesting depth and the keys of each table before
// the document is decoded. It parses once, only to count, and builds no tree.
//
// Depth is the segments of the table header, plus the segments of the key,
// plus the nesting of the value, where an array adds one level and an inline
// table adds the segments of its own keys. A document that reaches its end
// without error has every recursion over its tree at most MaxNestingDepth
// deep.
func guard(doc []byte) error {
	for line, rest := 1, doc; len(rest) > 0; line++ {
		n := bytes.IndexByte(rest, '\n')
		last := n < 0
		if last {
			n = len(rest)
		}
		if n > economysim.MaxLineBytes {
			return fmt.Errorf("line %d: %w", line, limitError("MaxLineBytes", economysim.MaxLineBytes, n))
		}
		if last {
			break
		}
		rest = rest[n+1:]
	}

	var p unstable.Parser
	p.Reset(doc)
	g := &guarder{p: &p, ids: map[child]int{}, kids: map[int]int{}}
	// header is the node of the current table header and depth its segments.
	var header, depth int
	for p.NextExpression() {
		e := p.Expression()
		switch e.Kind {
		case unstable.Table, unstable.ArrayTable:
			segs, err := g.key(e, 0)
			if err != nil {
				return err
			}
			if header, err = g.register(0, segs, e); err != nil {
				return err
			}
			depth = len(segs)
		case unstable.KeyValue:
			segs, err := g.key(e, depth)
			if err != nil {
				return err
			}
			id, err := g.register(header, segs, e)
			if err != nil {
				return err
			}
			if err := g.value(e.Value(), id, depth+len(segs)); err != nil {
				return err
			}
		}
	}
	if err := p.Error(); err != nil {
		// The library bounds nested arrays and inline tables itself, at 10 000
		// levels, and says so as a parse error. That is this loader's depth
		// limit, only larger, so it is reported as one. This reads the
		// library's message, one fixed phrase, and nothing of the document;
		// if the phrase changes the document is still rejected, as a syntax
		// error (ADR 0009).
		var perr *unstable.ParserError
		if errors.As(err, &perr) && strings.Contains(perr.Message, "nested more than") {
			return limitError("MaxNestingDepth", economysim.MaxNestingDepth, economysim.MaxNestingDepth+1)
		}
		// Decode the document again, only to take the library's line and
		// column; its text is never used.
		var m map[string]any
		return syntaxError(toml.Unmarshal(doc, &m))
	}
	return nil
}

// child names a node of the key set: the segment seg under the node parent.
type child struct {
	parent int
	seg    string
}

// guarder holds the sets of one guard pass: the node of each distinct path
// seen, and the number of children of each table. A path is a node id, so a
// segment is hashed once however long the path above it (spec-config.md 6.3.1).
type guarder struct {
	p    *unstable.Parser
	ids  map[child]int
	kids map[int]int
}

// where names the line of a node for an error, when the parser has it.
func (g *guarder) where(n *unstable.Node) string {
	if n == nil || n.Raw.Length == 0 {
		return ""
	}
	return " at line " + strconv.Itoa(g.p.Shape(n.Raw).Start.Line)
}

// key returns the segments of the key of a table header or key-value, and
// refuses one that, after base segments above it, is deeper than
// MaxNestingDepth.
func (g *guarder) key(e *unstable.Node, base int) ([]string, error) {
	var segs []string
	for it := e.Key(); it.Next(); {
		segs = append(segs, string(it.Node().Data))
		if base+len(segs) > economysim.MaxNestingDepth {
			return nil, fmt.Errorf("%w%s", limitError("MaxNestingDepth", economysim.MaxNestingDepth, base+len(segs)), g.where(it.Node()))
		}
	}
	return segs, nil
}

// register counts each of segs as a child of the table below parent, once per
// distinct path, and refuses a table of more than MaxTableKeys children. It
// returns the node of the last segment. Node 0 is the document root.
func (g *guarder) register(parent int, segs []string, at *unstable.Node) (int, error) {
	for _, s := range segs {
		c := child{parent, s}
		id, ok := g.ids[c]
		if !ok {
			id = len(g.ids) + 1
			g.ids[c] = id
			g.kids[parent]++
			if n := g.kids[parent]; n > economysim.MaxTableKeys {
				return 0, fmt.Errorf("%w%s", limitError("MaxTableKeys", economysim.MaxTableKeys, n), g.where(at))
			}
		}
		parent = id
	}
	return parent, nil
}

// value walks the nesting of a value at depth levels from the document root,
// whose node in the key set is id.
func (g *guarder) value(n *unstable.Node, id, depth int) error {
	switch n.Kind {
	case unstable.Array:
		if depth+1 > economysim.MaxNestingDepth {
			return fmt.Errorf("%w%s", limitError("MaxNestingDepth", economysim.MaxNestingDepth, depth+1), g.where(n))
		}
		i := 0
		for it := n.Children(); it.Next(); i++ {
			elem, err := g.register(id, []string{"\x00#" + strconv.Itoa(i)}, n)
			if err != nil {
				return err
			}
			if err := g.value(it.Node(), elem, depth+1); err != nil {
				return err
			}
		}
	case unstable.InlineTable:
		for it := n.Children(); it.Next(); {
			kv := it.Node()
			segs, err := g.key(kv, depth)
			if err != nil {
				return err
			}
			kid, err := g.register(id, segs, kv)
			if err != nil {
				return err
			}
			if err := g.value(kv.Value(), kid, depth+len(segs)); err != nil {
				return err
			}
		}
	}
	return nil
}
