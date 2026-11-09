package economysim

import "testing"

// BenchmarkRumours_ReferenceView reports the cost of the rumours of one informed
// view (error 500, so every candidate takes a draw) on the fixture of three
// regions, whose miller has 3 candidates, and the cost per candidate, which
// scales to the 1 280 candidates of the reference world of NFR-INF-002 (40
// informed agents of 8 commodities and 4 other regions). It reports; the figure
// is a placeholder and no limit is asserted (testing.md 8.2).
//
// Requirement: NFR-INF-002
func BenchmarkRumours_ReferenceView(b *testing.B) {
	c := infoConfig()
	c.Modules.Information.RumourErrorBP = 500
	w, err := NewWorld(c, WithStrategy("script", &infoScript{}))
	if err != nil {
		b.Fatalf("NewWorld() = %v, want nil", err)
	}
	for range 6 {
		if err := w.Step(); err != nil {
			b.Fatalf("Step() = %v, want nil", err)
		}
	}
	a, v := &w.agents[3], AgentView{Region: infoA}
	b.ReportAllocs()
	for b.Loop() {
		v.Rumours = nil
		w.fillRumours(&v, a)
	}
	if n := len(v.Rumours); n > 0 {
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(n), "ns/candidate")
	}
}
