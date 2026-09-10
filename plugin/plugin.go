// Package plugin exposes samber-linter as a golangci-lint v2 custom linter
// module plugin. The wiring follows the released, dogfooded pattern from
// go-humanize-linter/plugin (verified 2026-09-09).
//
// # Integration (golangci-lint v2)
//
// 1. Create a .custom-gcl.yml build config (see this repo's .custom-gcl.yml).
//
// 2. Build a custom golangci-lint binary:
//
//	golangci-lint custom   # produces ./custom-gcl
//
// 3. Register the linter in your .golangci.yml (CRITICAL — without this
// section, golangci-lint reports "unknown linters: healthwash"):
//
//	linters:
//	  enable:
//	    - healthwash
//	  settings:
//	    custom:
//	      healthwash:
//	        type: "module"
//	        description: "Detect health-washing in samber/do v2 containers"
//	        settings:
//	          strict: true          # optional: report HW-unresolved
//	          disable: "HW-4"       # optional: skip these rules
package plugin

import (
	"fmt"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"github.com/larsartmann/samber-linter/pkg/healthwash"
	"golang.org/x/tools/go/analysis"
)

// analyzerName is the linter name registered with golangci-lint. It matches
// the tool's namespace in suppressions and configs.
const analyzerName = "healthwash"

func init() { //nolint:gochecknoinits // required by golangci-lint plugin register API
	register.Plugin(analyzerName, newPlugin)
}

// pluginSettings holds optional configuration passed via .golangci.yml.
type pluginSettings struct {
	Strict  string `json:"strict"`
	Disable string `json:"disable"`
}

// healthwashPlugin implements register.LinterPlugin.
type healthwashPlugin struct {
	settings pluginSettings
}

func newPlugin(settings any) (register.LinterPlugin, error) { //nolint:ireturn // required by register.NewSettingsPlugin
	s, err := register.DecodeSettings[pluginSettings](settings)
	if err != nil {
		return nil, err
	}

	return &healthwashPlugin{settings: s}, nil
}

// BuildAnalyzers returns the healthwash analyzer configured from settings.
func (p *healthwashPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	a := healthwash.New()
	if strings.EqualFold(strings.TrimSpace(p.settings.Strict), "true") {
		if err := a.Flags.Set("strict", "true"); err != nil {
			return nil, fmt.Errorf("set strict flag: %w", err)
		}
	}

	if d := strings.TrimSpace(p.settings.Disable); d != "" {
		if err := a.Flags.Set("disable", d); err != nil {
			return nil, fmt.Errorf("set disable flag: %w", err)
		}
	}

	return []*analysis.Analyzer{a}, nil
}

// BuildDocs is required by the register API.
func (p *healthwashPlugin) BuildDocs() string {
	return "Detects health-washing in samber/do v2 DI containers: services that " +
		"render green `pass` on health dashboards but cannot actually fail. " +
		"Rules: HW-1 unchecked-resource-holder, HW-2 contextless-check, " +
		"HW-3 transient-health-washing, HW-4 lazy-never-built-pass, " +
		"HW-5 pointer-receiver-value-registration. See " +
		"https://github.com/LarsArtmann/samber-linter"
}

// GetName is required by the register API.
func (p *healthwashPlugin) GetName() string { return analyzerName }

// GetLoadMode is required by the register API: module plugins run per-package
// with full type information.
func (p *healthwashPlugin) GetLoadMode() string { return register.LoadModeSyntax }
