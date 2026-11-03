package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/mgl"
)

func TestProbeReportAndRows(t *testing.T) {
	args := []string{"-card", "../../testdata/mgl/card", "-arcade-list", "../../testdata/mgl/arcade-pilot.json"}
	var out, stderr bytes.Buffer
	if err := run(args, &out, &stderr); err != nil {
		t.Fatal(err)
	}
	var report mgl.Result
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || len(report.Rows) != 5 {
		t.Fatalf("report: %v %s", err, out.String())
	}
	out.Reset()
	if err := run(append(args, "-rows"), &out, &stderr); err != nil {
		t.Fatal(err)
	}
	var rows []data.Row
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil || len(rows) != 5 {
		t.Fatalf("rows: %v %s", err, out.String())
	}
	if err := run([]string{"-card", "../../testdata/mgl/card", "-rows"}, &out, &stderr); err == nil {
		t.Fatal("allowed unclassified rows")
	}
	if err := run([]string{"-card", t.TempDir() + "/missing"}, &out, &stderr); err == nil {
		t.Fatal("missing card reported as success")
	}
}
