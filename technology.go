package economysim

// This file is the technology module (spec-technology.md; architecture.md
// 2.10.1). In this change it holds the part the configuration needs: the
// recipe-to-field map that validation, the warnings and (later) the world share.
// The state, the step and the call follow in the later changes of the plan; the
// validation and the warnings are in technology_validate.go.

// techFieldOfRecipe maps each recipe ID of a technology module to the index of its
// field in the order of Fields, the first field that names it. It reads the
// configuration as written, so it is safe on one that does not validate. It is nil
// when the module is off.
func techFieldOfRecipe(m *Technology) map[string]int {
	if m == nil {
		return nil
	}
	out := make(map[string]int)
	for i := range m.Fields {
		for _, id := range m.Fields[i].Recipes {
			if _, taken := out[id]; !taken {
				out[id] = i
			}
		}
	}
	return out
}
