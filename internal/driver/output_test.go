package driver

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
	output "github.com/larsartmann/go-output"
)

func sampleFinding() finding.Finding {
	return finding.NewBuilder(
		finding.RuleName("HW-1"),
		finding.ToolName(ToolName),
		"service Store lacks a health check but implements Shutdown",
		finding.SeverityWarning,
		finding.Position{File: finding.FilePath("main.go"), Line: 66, Column: 2},
	).WithConfidence(finding.ConfidenceFull).MustBuild()
}

func TestSupportedOutputFormats(t *testing.T) {
	t.Parallel()

	formats := SupportedOutputFormats()
	if len(formats) == 0 {
		t.Fatal("no output formats registered; the go-output submodule imports were lost")
	}

	for _, want := range []output.Format{output.FormatTable, output.FormatCSV, output.FormatTSV, output.FormatMarkdown} {
		if !slices.Contains(formats, want) {
			t.Errorf("supported formats missing %q: %v", want, formats)
		}
	}

	for _, banned := range []output.Format{
		output.FormatJSON, output.FormatJSONL, output.FormatYAML, output.FormatTOML,
		output.FormatTree, output.FormatD2, output.FormatMermaid, output.FormatDOT, output.FormatPlantUML,
	} {
		if slices.Contains(formats, banned) {
			t.Errorf("format %q must not be served by --output (machine formats belong to --json/--sarif)", banned)
		}
	}
}

func TestParseOutputFormat(t *testing.T) {
	t.Parallel()

	t.Run("accepts the empty value as the plain-text default", func(t *testing.T) {
		t.Parallel()

		for _, empty := range []string{"", "  ", "\t"} {
			got, err := ParseOutputFormat(empty)
			if err != nil {
				t.Fatalf("ParseOutputFormat(%q) = %v, want nil: the zero flag value must not break plain invocations", empty, err)
			}

			if got != "" {
				t.Errorf("ParseOutputFormat(%q) = %q, want empty (driver falls back to plain text)", empty, got)
			}
		}
	})

	t.Run("accepts a supported format", func(t *testing.T) {
		t.Parallel()

		got, err := ParseOutputFormat("markdown")
		if err != nil {
			t.Fatalf("ParseOutputFormat(markdown) = %v, want nil", err)
		}

		if got != output.FormatMarkdown {
			t.Errorf("ParseOutputFormat(markdown) = %q, want %q", got, output.FormatMarkdown)
		}
	})

	t.Run("rejects machine formats with a tool hint", func(t *testing.T) {
		t.Parallel()

		_, err := ParseOutputFormat("json")
		if err == nil {
			t.Fatal("ParseOutputFormat(json) = nil, want error")
		}

		for _, want := range []string{"json", "--json", "supported formats"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error missing %q: %v", want, err)
			}
		}
	})

	t.Run("rejects unknown formats", func(t *testing.T) {
		t.Parallel()

		if _, err := ParseOutputFormat("nope"); err == nil {
			t.Fatal("ParseOutputFormat(nope) = nil, want error")
		}
	})
}

func TestRenderFindings(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{sampleFinding()}

	t.Run("markdown contains rule, location and message", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		if err := renderFindings(&out, findings, output.FormatMarkdown); err != nil {
			t.Fatalf("renderFindings(markdown) = %v, want nil", err)
		}

		for _, want := range []string{"Rule", "HW-1", "main.go:66:2", "lacks a health check"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("markdown output missing %q:\n%s", want, out.String())
			}
		}
	})

	t.Run("csv emits header row", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		if err := renderFindings(&out, findings, output.FormatCSV); err != nil {
			t.Fatalf("renderFindings(csv) = %v, want nil", err)
		}

		for _, want := range []string{"Rule", "HW-1"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("csv output missing %q:\n%s", want, out.String())
			}
		}
	})

	t.Run("empty findings still render the header", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		if err := renderFindings(&out, nil, output.FormatMarkdown); err != nil {
			t.Fatalf("renderFindings(nil, markdown) = %v, want nil", err)
		}

		if !strings.Contains(out.String(), "Rule") {
			t.Errorf("empty markdown output missing header:\n%s", out.String())
		}
	})
}

func TestRunWithOutputFormatEndToEnd(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns:      []string{"./..."},
		Dir:           app,
		Env:           []string{"GOFLAGS=-mod=mod"},
		Version:       "test",
		Stdout:        &out,
		Stderr:        &errOut,
		CoverageMin:   -1,
		MinConfidence: finding.ConfidenceHigh,
		OutputFormat:  output.FormatMarkdown,
	})
	if code != 1 {
		t.Fatalf("exit = %d, want 1 (high-confidence HW-1); stderr: %s", code, errOut.String())
	}

	reported := out.String()
	if !strings.Contains(reported, "HW-1") {
		t.Errorf("markdown output missing HW-1 finding:\n%s", reported)
	}

	if !strings.Contains(reported, "health-coverage: 1/3 = 33%") {
		t.Errorf("coverage line must stay plain text in presentation formats:\n%s", reported)
	}
}

func TestRunWithUnsupportedMachineFormatFailsLoudly(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns:      []string{"./..."},
		Dir:           app,
		Env:           []string{"GOFLAGS=-mod=mod"},
		Version:       "test",
		Stdout:        &out,
		Stderr:        &errOut,
		CoverageMin:   -1,
		MinConfidence: finding.ConfidenceHigh,
		OutputFormat:  output.Format("json"),
	})
	if code != 1 {
		t.Fatalf("exit = %d, want 1 (unsupported format); stdout: %s", code, out.String())
	}

	if !strings.Contains(errOut.String(), "json") {
		t.Errorf("stderr must name the failing format:\n%s", errOut.String())
	}
}
