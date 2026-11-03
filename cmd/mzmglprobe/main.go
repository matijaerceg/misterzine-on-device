// mzmglprobe reads menu MGLs without launching anything or changing the card.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/matijaerceg/misterzine-on-device/internal/mgl"
)

func run(args []string, out, stderr io.Writer) error {
	flags := flag.NewFlagSet("mzmglprobe", flag.ContinueOnError)
	flags.SetOutput(stderr)
	card := flags.String("card", "", "card root or copied menu folder tree (required)")
	list := flags.String("arcade-list", "", "reviewed Neo Geo arcade identities JSON; without it, report candidates only")
	rows := flags.Bool("rows", false, "emit matched rows for mzharness -local instead of the full report")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *card == "" || flags.NArg() != 0 {
		return fmt.Errorf("use -card PATH [-arcade-list FILE] [-rows]")
	}
	if *rows && *list == "" {
		return fmt.Errorf("-rows requires -arcade-list; cartridge XML alone does not establish arcade identity")
	}
	var catalogue *mgl.Catalogue
	if *list != "" {
		f, err := os.Open(*list)
		if err != nil {
			return err
		}
		catalogue, err = mgl.ReadCatalogue(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("arcade list: %w", err)
		}
	}
	result, scanErr := mgl.Scan(*card, catalogue)
	// A partial scan may be useful diagnostically, but must never masquerade
	// as a complete replacement of the user's game list.
	if *rows && (scanErr != nil || len(result.Problems) != 0) {
		return fmt.Errorf("incomplete scan: %v", result.Problems)
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	var value any = result
	if *rows {
		value = result.Rows
	}
	if err := enc.Encode(value); err != nil {
		return err
	}
	if scanErr != nil {
		return scanErr
	}
	if len(result.Problems) != 0 {
		return fmt.Errorf("scan had %d problems; see report", len(result.Problems))
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "mzmglprobe:", err)
		os.Exit(1)
	}
}
