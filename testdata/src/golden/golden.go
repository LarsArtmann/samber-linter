// Package golden is the frozen CV-incident corpus: each fixture mirrors a
// golden case from README §9. Expected findings are annotated inline; the
// analysistest run fails if any expectation is missed or any unexpected
// diagnostic appears (this doubles as the compile gate — fixtures must
// type-check cleanly).
package golden

import (
	"context"

	"database/sql"

	do "github.com/samber/do/v2"
)

// Store mirrors CV's graphrag.Store at incident HEAD: Shutdowner (ctx+error
// variant), no Healthchecker, registered lazily. HW-1 fires.
type Store struct {
	db *sql.DB
}

func NewStore(i do.Injector) (*Store, error) {
	return &Store{}, nil
}

func (s *Store) Shutdown(_ context.Context) error {
	return s.db.Close()
}

// want HW-1 on the Provide line below.
var _ = func() bool {
	do.Provide(nil, NewStore) // want `HW-1: \*golden\.Store implements do\.Shutdowner but no Healthchecker`
	return true
}()

// GroqChat mirrors a real CV checker: implements the context variant and is
// registered eagerly — clean by construction.
type GroqChat struct{}

func (g *GroqChat) HealthCheck(context.Context) error { return nil }

func NewGroqChat() *GroqChat { return &GroqChat{} }

var _ = func() bool {
	do.ProvideValue(nil, NewGroqChat())
	return true
}()

// Handler mirrors CV's DI-registered handlers: no lifecycle interfaces at
// all — rule precision requires this to stay clean.
type Handler struct{}

func NewHandler(i do.Injector) (*Handler, error) { return &Handler{}, nil }

var _ = func() bool {
	do.Provide(nil, NewHandler)
	return true
}()

// ConfigValue mirrors inert config registered by value — clean.
type ConfigValue struct {
	ListenAddr string
}

var _ = func() bool {
	do.ProvideValue(nil, ConfigValue{ListenAddr: ":8080"})
	return true
}()
