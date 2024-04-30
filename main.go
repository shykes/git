// Git as a Dagger Module
package main

import (
	"context"
)

const (
	gitStatePath    = "/git/state"
	gitWorktreePath = "/git/worktree"
	// The commit to use when download the git-filter-repo script
	gitFilterRepoCommit = "9da70bddfa491bc50fefc3c35fd5cec773182816"
	gitFilterRepoURL    = "https://raw.githubusercontent.com/newren/git-filter-repo/" + gitFilterRepoCommit + "/git-filter-repo"
)

type Git struct{}

// Load the contents of a git repository
func (g *Git) Load(
	ctx context.Context,
	// The source directory to load.
	// It must contain a `.git` directory, or be one.
	source *Directory,
	// A separate worktree, if needed.
	// +optional
	worktree *Directory,
) (*Repo, error) {
	var state *Directory
	if _, err := source.Directory(".git").Entries(ctx); err != nil {
		// If there is no .git, assume source *is* a .git
		state = source
		if worktree == nil {
			worktree = dag.Directory()
		}
	} else {
		// If there is a .git, split up state from worktree
		state = source.Directory(".git")
		if worktree == nil {
			worktree = source.WithoutDirectory(".git")
		}
	}
	return &Repo{
		State:    state,
		Worktree: worktree,
		Git:      g,
	}, nil
}

// Initialize a git repository
func (g *Git) Init() *Repo {
	return &Repo{
		State: g.container().
			WithDirectory(gitStatePath, dag.Directory()).
			WithExec([]string{
				"git", "--git-dir=" + gitStatePath,
				"init", "-q", "--bare",
			}).
			Directory(gitStatePath),
		Worktree: dag.Directory(),
		Git:      g,
	}
}

// Clone a remote git repository
func (g *Git) Clone(ctx context.Context, url string) *Repo {
	clone := g.container().
		WithWorkdir("/tmp").
		WithExec([]string{"git", "clone", url, "src"}).
		Directory("src")

	return g.
		Init().
		WithState(clone.Directory(".git")).
		WithWorktree(clone.WithoutDirectory(".git"))
}

func (g *Git) container() *Container {
	return dag.
		Wolfi().
		Container(WolfiContainerOpts{
			Packages: []string{"git", "openssh-client", "openssh-keyscan", "python3"},
		}).
		WithFile(
			"/bin/git-filter-repo",
			dag.HTTP(gitFilterRepoURL),
			ContainerWithFileOpts{
				Permissions: 0755,
			},
		).
		WithMountedCache("/root/.ssh", dag.CacheVolume("git-known-hosts")).
		WithEnvVariable("GIT_SSH_COMMAND", "ssh -o StrictHostKeyChecking=accept-new")
}
