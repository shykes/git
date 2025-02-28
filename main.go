// Git as a Dagger Module
package main

import (
	"context"
	"git/internal/dagger"
	"strings"
	"time"
)

const (
	gitStatePath    = "/git/state"
	gitWorktreePath = "/git/worktree"
	// The commit to use when download the git-filter-repo script
	gitFilterRepoCommit = "9da70bddfa491bc50fefc3c35fd5cec773182816"
	gitFilterRepoURL    = "https://raw.githubusercontent.com/newren/git-filter-repo/" + gitFilterRepoCommit + "/git-filter-repo"
)

type Git struct {
	// +private
	SSHKey *dagger.Secret
}

func New(
	// SSH key to use for git operations.
	//
	// +optional
	sshKey *dagger.Secret,
) *Git {
	return &Git{
		SSHKey: sshKey,
	}
}

// Load the contents of a git repository
func (g *Git) Load(
	ctx context.Context,
	// The source directory to load.
	// It must contain a `.git` directory, or be one.
	source *dagger.Directory,
	// A separate worktree, if needed.
	// +optional
	worktree *dagger.Directory,
) (*Repo, error) {
	var state *dagger.Directory
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
		WithEnvVariable("CACHE_BUSTER", time.Now().Format(time.RFC3339Nano)).
		WithExec([]string{"git", "clone", url, "src"}).
		Directory("src")

	return g.
		Init().
		WithState(clone.Directory(".git")).
		WithWorktree(clone.WithoutDirectory(".git"))
}

func (g *Git) container() *dagger.Container {
	sshArgs := []string{
		"ssh",
		"-o", "StrictHostKeyChecking=accept-new",
	}

	return dag.
		Wolfi().
		Container(dagger.WolfiContainerOpts{
			Packages: []string{"git", "openssh-client", "openssh-keyscan", "python3"},
		}).
		WithFile(
			"/bin/git-filter-repo",
			dag.HTTP(gitFilterRepoURL),
			dagger.ContainerWithFileOpts{
				Permissions: 0755,
			},
		).
		WithMountedCache("/root/.ssh", dag.CacheVolume("git-known-hosts")).
		With(func(c *dagger.Container) *dagger.Container {
			if g.SSHKey != nil {
				sshArgs = append(sshArgs, "-i", "/git/ssh-key")

				// This is an ugly hack until the following issue is resolved: https://github.com/dagger/dagger/issues/7220
				sshKeyContent, _ := g.SSHKey.Plaintext(context.TODO())
				sshKeyContent += "\n"

				sshKey := dag.SetSecret("ssh-key", sshKeyContent)

				c = c.
					WithMountedSecret("/git/ssh-key", sshKey)
			}

			return c
		}).
		WithEnvVariable("GIT_SSH_COMMAND", strings.Join(sshArgs, " "))
}
