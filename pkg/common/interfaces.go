package common

import "context"

// Typer allows retrieving a type pointer.
type Typer interface {
	// For type T, usually returns `(*T)(nil)`.
	Type() any
}

// Runner can be started by calling Run.
type Runner interface {
	// Run blocks during object lifecycle.
	Run(ctx context.Context) error
}

// Stopper can be stopped by calling Stop.
type Stopper interface {
	// Stop signals and waits for object to stop unless context is canceled.
	Stop(ctx context.Context) error
}

type RunnerStopper interface {
	Runner
	Stopper
}
