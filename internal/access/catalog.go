package access

// NamedFeature couples documentation to the same thresholds used by the app.
type NamedFeature struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Feature
}

// Catalog returns fresh values so graduation and threshold changes cannot leave
// a separately maintained coverage list behind.
func Catalog() []NamedFeature {
	return []NamedFeature{
		{"themes", "Themes", Themes},
		{"group-tallies", "Group tallies", Tallies},
		{"rom-report", "ROM report", ROMReport},
		{"button-mapping", "Button mapping", Controls},
	}
}

func (f Feature) Category() string {
	if f.Fancy {
		if f.Beta {
			return "Supporter / beta"
		}
		return "Supporter"
	}
	if f.Beta {
		return "Free beta"
	}
	return "Free"
}
