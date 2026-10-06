// Package controls defines the saved-profile contract for controller features.
package controls

type Profile struct {
	Reviewed bool   `json:"reviewed,omitempty"`
	Name     string `json:"name,omitempty"`
	Panel    bool   `json:"arcade_panel,omitempty"`
	// Missing actions inherit; zero explicitly clears an action after a swap.
	Bindings map[string]uint16 `json:"bindings,omitempty"`
	Style    string            `json:"label_style,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
	Names    map[uint16]string `json:"button_names,omitempty"`
	NameBase string            `json:"name_base,omitempty"`
}

type File struct {
	Schema   int                `json:"schema"`
	Profiles map[string]Profile `json:"profiles"`
}

func New() File { return File{Schema: 1, Profiles: map[string]Profile{}} }
func (p Profile) Clone() Profile {
	q := p
	q.Bindings = map[string]uint16{}
	for k, v := range p.Bindings {
		q.Bindings[k] = v
	}
	q.Labels = map[string]string{}
	for k, v := range p.Labels {
		q.Labels[k] = v
	}
	q.Names = map[uint16]string{}
	for k, v := range p.Names {
		q.Names[k] = v
	}
	return q
}
func (f File) Clone() File {
	g := New()
	for id, p := range f.Profiles {
		g.Profiles[id] = p.Clone()
	}
	return g
}
