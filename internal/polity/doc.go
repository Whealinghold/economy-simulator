// Package polity is the arithmetic of the polity module (spec-polity.md): the
// tables of a tree of jurisdictions over the regions, the queries on them (the
// chain of a region, whether a jurisdiction covers it, the common jurisdiction of
// two regions, the regions a jurisdiction covers, the crossing of an ordered pair
// of regions and the edge-pair test of Between) and the three ways a table keyed
// by jurisdiction combines along a chain (innermost, added, scaled).
//
// It works on primitive slices and knows no region, commodity or edge: a
// jurisdiction and a region are indexes from 0, and -1 means none. The root
// converts at the call (architecture.md 2.9). It takes no random source, reads no
// clock and keeps no state beyond the Tables it builds. Every list is ascending by
// index except a chain, which is innermost first.
package polity
