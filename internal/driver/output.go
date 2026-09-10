package driver

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"
	output "github.com/larsartmann/go-output"
	_ "github.com/larsartmann/go-output/delimited" // csv, tsv table marshalers
	_ "github.com/larsartmann/go-output/markdown"  // markdown table marshaler
	_ "github.com/larsartmann/go-output/markup"    // html, xml, asciidoc table marshalers
	_ "github.com/larsartmann/go-output/table"     // terminal table marshaler
)

// ErrUnsupportedOutputFormat reports a --output value that is a valid
// go-output format but cannot render findings. Callers can match it with
// errors.Is; the message names the supported formats and the machine-readable
// flags.
var ErrUnsupportedOutputFormat = errors.New("unsupported --output format")

// SupportedOutputFormats lists the presentation formats renderFindings can
// emit. Machine-readable formats are deliberately absent: --json and --sarif
// own that contract via go-finding, and a second JSON flavor would invite
// drift between the two shapes.
func SupportedOutputFormats() []output.Format {
	formats := output.RegisteredTableMarshalFormats()
	slices.SortFunc(formats, func(a, b output.Format) int {
		return strings.Compare(string(a), string(b))
	})

	return formats
}

// SupportedOutputFormatNames returns the supported formats as strings for CLI
// help text.
func SupportedOutputFormatNames() []string {
	return formatNames(SupportedOutputFormats())
}

// ParseOutputFormat validates a --output flag value. Any parseable format
// outside SupportedOutputFormats is rejected with a hint toward --json and
// --sarif so the failure names the right tool instead of listing enum values
// that exist but cannot render findings.
func ParseOutputFormat(value string) (output.Format, error) {
	format, err := output.ParseFormat(value)
	if err != nil {
		return "", fmt.Errorf("invalid --output format: %w", err)
	}

	if !slices.Contains(SupportedOutputFormats(), format) {
		return "", fmt.Errorf(
			"%w: %q cannot render findings; supported formats: %s (machine-readable output: --json, --sarif)",
			ErrUnsupportedOutputFormat,
			value,
			strings.Join(SupportedOutputFormatNames(), ", "),
		)
	}

	return format, nil
}

// renderFindings writes the findings table in the requested presentation
// format. The go-finding report (--json) and SARIF export are unaffected.
func renderFindings(out io.Writer, findings []finding.Finding, format output.Format) error {
	tbl, err := findingsTable(findings)
	if err != nil {
		return fmt.Errorf("building findings table: %w", err)
	}

	if err := output.RenderTable(tbl, format, output.RenderOptions{Writer: out}); err != nil {
		return fmt.Errorf("rendering findings as %s: %w", format, err)
	}

	return nil
}

// findingsTable maps findings to go-output rows. The column count is fixed at
// the header arity, so AddRowChecked can only fail on a programming error.
func findingsTable(findings []finding.Finding) (*output.Table, error) {
	tbl := output.NewTable([]string{"Rule", "Severity", "Confidence", "Location", "Message"})

	for _, record := range findings {
		row := []string{
			string(record.Rule),
			record.Severity.String(),
			record.Confidence.String(),
			fmt.Sprintf("%s:%d:%d", record.Position.File, record.Position.Line, record.Position.Column),
			record.Message,
		}

		if err := tbl.AddRowChecked(row); err != nil {
			return nil, fmt.Errorf("adding finding row: %w", err)
		}
	}

	return tbl, nil
}

func formatNames(formats []output.Format) []string {
	names := make([]string, 0, len(formats))
	for _, f := range formats {
		names = append(names, string(f))
	}

	return names
}
