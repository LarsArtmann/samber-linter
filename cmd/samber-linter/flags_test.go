package main

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	v4 "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/larsartmann/samber-linter/internal/driver"
)

// TestFlagSurfaceMatchesLegacyContract pins the cmdguard flag struct to the
// stdlib-flag CLI it replaced: the same twelve flag names, each with help
// text, and the defaults the README documents.
func TestFlagSurfaceMatchesLegacyContract(t *testing.T) {
	t.Parallel()

	want := map[string]string{ // flag name -> default tag
		"json":           "",
		"sarif":          "",
		"output":         "",
		"strict":         "",
		"disable":        "",
		"check":          "",
		"coverage-min":   "-1",
		"set-baseline":   "",
		"baseline":       "", // runLinter falls back to driver.DefaultBaselinePath
		"config":         "",
		"min-confidence": strconv.FormatFloat(defaultMinConfidence, 'f', -1, 64),
		"version":        "",
	}

	got := make(map[string]string)

	for field := range reflect.TypeFor[linterFlags]().Fields() {
		name := field.Tag.Get("flag")
		if name == "" {
			t.Errorf("field %s has no flag tag", field.Name)

			continue
		}

		if field.Tag.Get("help") == "" {
			t.Errorf("flag %s (field %s) has no help text", name, field.Name)
		}

		got[name] = field.Tag.Get("default")
	}

	for name, def := range want {
		gotDef, ok := got[name]
		if !ok {
			t.Errorf("flag %s is missing from linterFlags", name)

			continue
		}

		if gotDef != def {
			t.Errorf("flag %s default = %q, want %q", name, gotDef, def)
		}

		delete(got, name)
	}

	for name := range got {
		t.Errorf("unexpected extra flag %s (not part of the legacy contract)", name)
	}
}

// TestBaselineFallsBackToDefault pins the empty --baseline fallback to the
// driver's default: the flag tag cannot carry the driver constant (tag values
// are compile-time strings), so runLinter owns the default.
func TestBaselineFallsBackToDefault(t *testing.T) {
	t.Parallel()

	if got := driver.DefaultBaselinePath; got != ".samber-linter-baseline.json" {
		t.Fatalf("driver.DefaultBaselinePath = %q, want the documented .samber-linter-baseline.json", got)
	}
}

// TestOutputHelpListsSupportedFormats proves the --output help names exactly
// the formats the driver accepts — the static tag cannot (the list is
// dynamic), so dynamicOutputHelp owns it and this test guards the wiring
// against the real CLI construction.
func TestOutputHelpListsSupportedFormats(t *testing.T) {
	t.Parallel()

	cli, err := v4.NewCLI[linterFlags]("samber-linter", "test", linterFlags{},
		v4.WithHelpTransform(dynamicOutputHelp))
	if err != nil {
		t.Fatalf("building CLI: %v", err)
	}

	root := cli.RootCommand()
	dynamicOutputHelp(root)

	outputFlag := root.PersistentFlags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("output flag not registered")
	}

	for _, name := range driver.SupportedOutputFormatNames() {
		if !strings.Contains(outputFlag.Usage, name) {
			t.Errorf("--output help %q does not mention supported format %q", outputFlag.Usage, name)
		}
	}
}
