// want package:"healthwash: registration records"

// Package golden is the frozen CV-incident corpus (README §9): each fixture
// mirrors a golden case. Expectations are annotated inline; the analysistest
// run fails on any miss (this doubles as the compile gate).
package golden

import (
	"context"
	"database/sql"

	do "github.com/samber/do/v2"
)

// Store mirrors CV's graphrag.Store at incident HEAD: Shutdowner (ctx+error),
// no Healthchecker, registered lazily. HW-1 fires.
type Store struct {
	db *sql.DB
}

func NewStore(i do.Injector) (*Store, error) {
	return &Store{}, nil
}

func (s *Store) Shutdown(_ context.Context) error {
	return s.db.Close()
}

var _ = func() bool {
	do.Provide(nil, NewStore) // want `HW-1: \*Store implements do\.Shutdowner but no Healthchecker`
	return true
}()

// GroqChat mirrors a real CV checker: context variant, eager — clean.
type GroqChat struct{}

func (g *GroqChat) HealthCheck(context.Context) error { return nil }

func NewGroqChat() *GroqChat { return &GroqChat{} }

var _ = func() bool {
	do.ProvideValue(nil, NewGroqChat())
	return true
}()

// Handler mirrors DI-registered handlers: no lifecycle interfaces — rule
// precision requires clean.
type Handler struct{}

func NewHandler(i do.Injector) (*Handler, error) { return &Handler{}, nil }

var _ = func() bool {
	do.Provide(nil, NewHandler)
	return true
}()

// ConfigValue mirrors inert config by value — clean.
type ConfigValue struct {
	ListenAddr string
}

var _ = func() bool {
	do.ProvideValue(nil, ConfigValue{ListenAddr: ":8080"})
	return true
}()
