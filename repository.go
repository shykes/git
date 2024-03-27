package main

import (
	"context"
	"strings"
)

// A new git repository
func (r *Git) Repository() *Repo {
	// We need to initialize these fields in a constructor,
	// because we can't hide them behind an accessor
	// (private fields are not persisted in between dagger function calls)
	return &Repo{
		State: container().
			WithDirectory(gitStatePath, dag.Directory()).
			WithExec([]string{
				"git", "--git-dir=" + gitStatePath,
				"init", "-q", "--bare",
			}).
			Directory(gitStatePath),
		Worktree: dag.Directory(),
	}
}

// A git repository
type Repo struct {
	State    *Directory
	Worktree *Directory
}

// Execute a git command in the repository
func (r *Repo) WithCommand(args []string) *Repo {
	return r.Command(args).Output()
}

// A Git command executed from the current repository state
func (r *Repo) Command(args []string) *GitCommand {
	return &GitCommand{
		Args:  args,
		Input: r,
	}
}

// A Git command
type GitCommand struct {
	Args  []string
	Input *Repo
}

func (cmd *GitCommand) container() *Container {
	prefix := []string{"git", "--git-dir=" + gitStatePath, "--work-tree=" + gitWorktreePath}
	execArgs := append(prefix, cmd.Args...)
	return container().
		WithDirectory(gitStatePath, cmd.Input.State).
		WithDirectory(gitWorktreePath, cmd.Input.Worktree).
		WithExec(execArgs)
}

func (cmd *GitCommand) Debug() *Terminal {
	return cmd.container().WithWorkdir(gitWorktreePath).Terminal()
}

func (cmd *GitCommand) Stdout(ctx context.Context) (string, error) {
	return cmd.container().Stdout(ctx)
}

func (cmd *GitCommand) Stderr(ctx context.Context) (string, error) {
	return cmd.container().Stderr(ctx)
}

func (cmd *GitCommand) Sync(ctx context.Context) (*GitCommand, error) {
	_, err := cmd.container().Sync(ctx)
	return cmd, err
}

func (cmd *GitCommand) Output() *Repo {
	container := cmd.container()
	return &Repo{
		State:    container.Directory(gitStatePath),
		Worktree: container.Directory(gitWorktreePath),
	}
}

func (r *Repo) WithRemote(name, url string) *Repo {
	return r.WithCommand([]string{"remote", "add", name, url})
}

func (r *Repo) Tag(name string) *Tag {
	return &Tag{
		Repository: r,
		Name:       name,
	}
}

func (t *Tag) FullName() string {
	if strings.HasPrefix(t.Name, "refs/tags/") {
		return t.Name
	}
	if strings.HasPrefix(t.Name, "tags/") {
		return "refs/" + t.Name
	}
	return "refs/tags/" + t.Name
}

type Tag struct {
	Repository *Repo
	Name       string
}

func (t *Tag) Tree() *Directory {
	return t.Repository.WithCommand([]string{"checkout", t.Name}).Worktree
}

func (r *Repo) Commit(digest string) *Commit {
	return &Commit{
		Repository: r,
		Digest:     digest,
	}
}

type Commit struct {
	Digest     string
	Repository *Repo
}

func (c *Commit) Tree() *Directory {
	return c.Repository.
		WithCommand([]string{"checkout", c.Digest}).
		Worktree
}
