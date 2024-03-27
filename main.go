package main

const (
	gitStatePath    = "/git/state"
	gitWorktreePath = "/git/worktree"
	// The commit to use when download the git-filter-repo script
	gitFilterRepoCommit = "9da70bddfa491bc50fefc3c35fd5cec773182816"
	gitFilterRepoURL    = "https://raw.githubusercontent.com/newren/git-filter-repo/" + gitFilterRepoCommit + "/git-filter-repo"
)

type Git struct{}

func (s *Git) Container() *Container {
	return container()
}

func container() *Container {
	return dag.
		Wolfi().
		Container(WolfiContainerOpts{
			Packages: []string{"git", "python3"},
		}).
		WithFile(
			"/bin/git-filter-repo",
			dag.HTTP(gitFilterRepoURL),
			ContainerWithFileOpts{
				Permissions: 0755,
			},
		)
}
