package data

// SourceFile is the URL-free account of one Downloader INI check. Paths are
// relative to the card root. The UI and support report share this snapshot.
type SourceFile struct {
	Path     string          `json:"path"`
	State    string          `json:"state"`
	Note     string          `json:"note,omitempty"`
	Sections []SourceSection `json:"sections,omitempty"`
}

const (
	SourceFileUsed       = "used"
	SourceFileMissing    = "missing"
	SourceFileUnreadable = "unreadable"
	SourceFileSkipped    = "skipped"
)

// SourceSection records the section name and matching catalogue source IDs,
// never the database address (which can contain an access key).
type SourceSection struct {
	ID      string   `json:"id"`
	Sources []string `json:"sources,omitempty"`
	Default bool     `json:"default,omitempty"`
}
