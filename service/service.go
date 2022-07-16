package service

import "context"

type Service interface {
	Name() string

	// Run executes the service and blocks until the context gets cancelled
	// or an error occurs.
	Run(context.Context) error
}
