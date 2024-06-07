package apprunner

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Runner interface {
	Start() error
	Stop()
}

type RunnerWithName struct {
	Runner
	name string
}

func NewRunner(name string, runner Runner) RunnerWithName {
	return RunnerWithName{
		Runner: runner,
		name:   name,
	}
}

func StartApp(ctx context.Context, runners ...RunnerWithName) error {
	wg := sync.WaitGroup{}

	errs := []error{}
	errCh := make(chan error, len(runners))

	start := func(runner RunnerWithName) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runner.Start(); err != nil {
				errCh <- fmt.Errorf("%s: %w", runner.name, err)
			}
		}()
	}

	for _, runner := range runners {
		start(runner)
	}

	done := make(chan any, 1)

	go func() {
		wg.Wait()
		done <- nil
	}()

	select {
	case <-ctx.Done():
		errs = append(errs, ctx.Err())
		for _, runner := range runners {
			runner.Stop()
		}
	case err := <-errCh:
		errs = append(errs, err)
	case <-done:
	}

	wg.Wait()
	lenErrCh := len(errCh)
	for i := 0; i < lenErrCh; i++ {
		errs = append(errs, <-errCh)
	}
	return errors.Join(errs...)
}
