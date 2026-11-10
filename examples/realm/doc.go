// Command realm is the first consumer's example: the reference chain `five` in a
// fantasy realm's words, loaded from an embedded pack and driven through the public
// API of the economy-sim module alone (no internal/ import, architecture.md 1.3).
//
// The package has the pack (pack.go), the game that owns the world behind a mutex
// (game.go) and, in the next change, an HTTP/JSON server over the game. The setting
// words live here and in the pack only; the library stays setting-agnostic.
package main
