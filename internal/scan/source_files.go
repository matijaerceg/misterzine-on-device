package scan

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// DownloaderScan keeps the filtering inputs and their provenance from one
// scan. Only Files, which contains no database URLs, goes to the UI/report.
type DownloaderScan struct {
	DBs   []data.DB
	Found bool
	Files []data.SourceFile
}

// ScanDownloader follows the first readable main INI and its drop-ins. Other
// locations are inventoried too, but their entries never affect filtering.
func ScanDownloader(card string) DownloaderScan {
	var result DownloaderScan
	selected := ""
	for _, rel := range []string{"downloader.ini", "Scripts/downloader.ini"} {
		f := data.SourceFile{Path: rel}
		path := filepath.Join(card, filepath.FromSlash(rel))
		if selected != "" {
			f = skippedSourceFile(card, rel, "Using "+selected)
		} else if raw, err := os.ReadFile(path); err != nil {
			f.State = sourceReadState(err)
		} else {
			selected, result.Found = rel, true
			f.State = data.SourceFileUsed
			f.Note = "First readable main INI"
			sections := parseINISections(string(raw))
			defaultDB := len(sections) == 0
			if defaultDB {
				sections = []data.DB{{ID: "distribution_mister"}}
			}
			result.DBs = append(result.DBs, sections...)
			f.Sections = sourceSections(sections, defaultDB)
		}
		result.Files = append(result.Files, f)
	}
	for _, dir := range []string{".", "Scripts"} {
		use := selected != "" && filepath.Dir(filepath.FromSlash(selected)) == dir
		note := "No readable main INI"
		if selected != "" {
			note = "Drop-ins belong beside " + selected
		}
		for _, pattern := range []string{"downloader/*.ini", "downloader_*.ini"} {
			relPattern := filepath.Join(dir, filepath.FromSlash(pattern))
			paths, listErr := sourceDropIns(card, dir, pattern)
			listed := false
			for _, path := range paths {
				if strings.HasPrefix(filepath.Base(path), ".") {
					continue
				}
				listed = true
				rel, _ := filepath.Rel(card, path)
				rel = filepath.ToSlash(rel)
				f := data.SourceFile{Path: rel}
				if !use {
					f = skippedSourceFile(card, rel, note)
				} else if raw, err := os.ReadFile(path); err != nil {
					f.State = sourceReadState(err)
				} else {
					f.State = data.SourceFileUsed
					sections := parseINISections(string(raw))
					result.DBs = append(result.DBs, sections...)
					f.Sections = sourceSections(sections, false)
					if len(sections) == 0 {
						f.Note = "No database sections"
					}
				}
				result.Files = append(result.Files, f)
			}
			if !listed {
				f := data.SourceFile{Path: filepath.ToSlash(relPattern), State: data.SourceFileMissing, Note: "No matching files"}
				if listErr != nil && !os.IsNotExist(listErr) {
					f.State, f.Note = data.SourceFileUnreadable, "Could not list this location"
				}
				result.Files = append(result.Files, f)
			}
		}
	}
	return result
}

// Unlike Glob, ReadDir retains I/O errors so they cannot look like an empty
// directory. ReadDir's sorted names preserve Downloader's drop-in order.
func sourceDropIns(card, dir, pattern string) ([]string, error) {
	base := filepath.Join(card, dir)
	namePattern := pattern
	if strings.HasPrefix(pattern, "downloader/") {
		base = filepath.Join(base, "downloader")
		namePattern = "*.ini"
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		match, _ := filepath.Match(namePattern, entry.Name())
		if match && !strings.HasPrefix(entry.Name(), ".") {
			paths = append(paths, filepath.Join(base, entry.Name()))
		}
	}
	return paths, nil
}

func sourceReadState(err error) string {
	if os.IsNotExist(err) {
		return data.SourceFileMissing
	}
	return data.SourceFileUnreadable
}

func skippedSourceFile(card, rel, note string) data.SourceFile {
	f := data.SourceFile{Path: rel, State: data.SourceFileSkipped, Note: note}
	if info, err := os.Stat(filepath.Join(card, filepath.FromSlash(rel))); err != nil {
		f.State, f.Note = sourceReadState(err), ""
	} else if !info.Mode().IsRegular() {
		f.State, f.Note = data.SourceFileUnreadable, "Not a regular file"
	}
	return f
}

func sourceSections(dbs []data.DB, defaultDB bool) []data.SourceSection {
	var sections []data.SourceSection
	for _, db := range dbs {
		sections = append(sections, data.SourceSection{ID: db.ID, Sources: data.SourcesForDB(db), Default: defaultDB})
	}
	return sections
}
