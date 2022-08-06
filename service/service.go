package service

import (
	"context"
	"sync"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/xerrors"
)

type Service interface {
	Name() string

	// Run executes the service and blocks until the context gets cancelled
	// or an error occurs.
	Run(context.Context) error
}

type ServiceGroup []Service


// Run executes all Service instances in the group using the provided context.
// Calls to Run block until all services have completed executing either because
// the context was cancelled or any of the services reported an error.
func (g ServiceGroup) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(g))
	wg.Add(len(g))
	for _, s := range g {
		go func(s Service) {
			defer wg.Done()	
			if err := s.Run(runCtx); err != nil {
				errCh <- xerrors.Errorf("%s: %w", s.Name(), err)
				cancel()
			}
		}(s)
	}
	
	<-runCtx.Done()
	wg.Wait()
	
	var err error
	close(errCh)
	for svcErr := range errCh {
		err = multierror.Append(err, svcErr)
	}
	return err
}
