package main

const (
	gitStatePath    = "/git/state"
	gitWorktreePath = "/git/worktree"
)

type Git struct{}

func (s *Git) Container() *Container {
	return container()
}

func container() *Container {
	return dag.
		Container().
		From("cgr.dev/chainguard/wolfi-base").
		WithExec([]string{"apk", "add", "git"})
}
