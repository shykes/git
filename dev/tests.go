package main

import (
	"context"

	"github.com/sourcegraph/conc/pool"
)

// Run tests for the module.
func (m *Dev) Test() *Test {
	return &Test{}
}

type Test struct{}

// All executes all tests.
func (m *Test) All(ctx context.Context) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(m.CloneHttp)

	return p.Wait()
}

func (m *Test) CloneHttp(ctx context.Context) error {
	_, err := dag.Git().Clone("https://github.com/shykes/git.git").Directory().Sync(ctx)

	return err
}
