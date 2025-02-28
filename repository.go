package main

import (
	"context"
	"git/internal/dagger"
	"strings"
)

// A git repository
type Repo struct {
	State    *dagger.Directory
	Worktree *dagger.Directory

	// +private
	Git *Git
}

// Combine the repository's worktree and state into a single directory.
//
//	The state is copied to `.git`
func (r *Repo) Directory() *dagger.Directory {
	return r.Worktree.WithDirectory(".git", r.State)
}

// Checkout the given ref into the worktree
func (r *Repo) Checkout(
	// The git ref to checkout
	ref string,
) *Repo {
	return r.WithCommand([]string{"checkout", ref})
}

// Set the git state directory
func (r *Repo) WithState(dir *dagger.Directory) *Repo {
	r.State = dir

	return r
}

// Set the git worktree
func (r *Repo) WithWorktree(dir *dagger.Directory) *Repo {
	r.Worktree = dir

	return r
}

// Filter the contents of the repository
func (r *Repo) FilterSubdirectory(path string) *Repo {
	return r.WithCommand([]string{
		"filter-repo", "--force", "--subdirectory-filter", path,
	})
}

// Filter history by moving the whole repo to a subdirectory
func (r *Repo) FilterToSubdirectory(path string) *Repo {
	return r.WithCommand([]string{
		"filter-repo", "--force", "--to-subdirectory-filter", path,
	})
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
		Git:   r.Git,
	}
}

// A Git command
type GitCommand struct {
	Args  []string
	Input *Repo

	// +private
	Git *Git
}

func (cmd *GitCommand) container() *dagger.Container {
	prefix := []string{"git", "--git-dir=" + gitStatePath, "--work-tree=" + gitWorktreePath}
	execArgs := append(prefix, cmd.Args...)
	return cmd.Git.container().
		WithDirectory(gitStatePath, cmd.Input.State).
		WithDirectory(gitWorktreePath, cmd.Input.Worktree).
		WithExec(execArgs)
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
		Git:      cmd.Git,
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

func (t *Tag) Tree() *dagger.Directory {
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

func (c *Commit) Tree() *dagger.Directory {
	return c.Repository.
		WithCommand([]string{"checkout", c.Digest}).
		Worktree
}
