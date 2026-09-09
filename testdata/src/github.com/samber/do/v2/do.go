// Package do is a signature-faithful stub of github.com/samber/do/v2 used by
// the analysistest fixtures (GOPATH mode under testdata/src). Only the surface
// the analyzer resolves is present: the six lifecycle interfaces and the
// registration functions with their exact generic shapes.
//
// Fidelity to the REAL samber/do source is enforced separately by the drift
// matrix test, which parses the actual v2.0.0/v2.1.0 sources from the module
// cache and asserts the same signatures these fixtures rely on.
package do

import "context"

// Injector is the container handle passed to providers.
type Injector interface {
	InjectorOf() interface{}
}

type Healthchecker interface {
	HealthCheck() error
}

type HealthcheckerWithContext interface {
	HealthCheck(context.Context) error
}

type Shutdowner interface {
	Shutdown()
}

type ShutdownerWithError interface {
	Shutdown() error
}

type ShutdownerWithContext interface {
	Shutdown(context.Context)
}

type ShutdownerWithContextAndError interface {
	Shutdown(context.Context) error
}

type Provider[T any] func(Injector) (T, error)

func Provide[T any](i Injector, provider Provider[T]) {}

func ProvideNamed[T any](i Injector, name string, provider Provider[T]) {}

func ProvideValue[T any](i Injector, value T) {}

func ProvideNamedValue[T any](i Injector, name string, value T) {}

func ProvideTransient[T any](i Injector, provider Provider[T]) {}

func ProvideNamedTransient[T any](i Injector, name string, provider Provider[T]) {}

func Override[T any](i Injector, provider Provider[T]) {}

func OverrideNamed[T any](i Injector, name string, provider Provider[T]) {}

func OverrideValue[T any](i Injector, value T) {}

func OverrideNamedValue[T any](i Injector, name string, value T) {}

func OverrideTransient[T any](i Injector, provider Provider[T]) {}

func OverrideNamedTransient[T any](i Injector, name string, provider Provider[T]) {}

func As[Initial any, Alias any](i Injector) error { return nil }

func AsNamed[Initial any, Alias any](i Injector, initial string, alias string) error {
	return nil
}

func Invoke[T any](i Injector) (T, error) {
	var zero T
	return zero, nil
}
