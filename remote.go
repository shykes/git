package main

import (
	"context"
	"regexp"
	"strings"
)

// Initialize a reference to a git remote
func (r *Git) Remote(url string) *Remote {
	return &Remote{
		URL: url,
		Git: r,
	}
}

// A git remote
type Remote struct {
	URL string

	// +private
	Git *Git
}

// Lookup a tag in the remote
func (r *Remote) Tag(ctx context.Context, name string) (*RemoteTag, error) {
	output, err := r.Git.container().WithExec([]string{"git", "ls-remote", "--tags", r.URL, name}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	line, _, _ := strings.Cut(output, "\n")
	commit, name := tagSplit(line)
	return &RemoteTag{
		CommitID: commit,
		Name:     name,
		URL:      r.URL,
		Git:      r.Git,
	}, nil
}

// Query the remote for its tags.
//
//	If `filter` is set, only tag matching that regular expression will be included.
func (r *Remote) Tags(
	ctx context.Context,
	// A regular expression to filter tag names. Only matching tag names will be included.
	// +optional
	filter string,
) ([]*RemoteTag, error) {
	var (
		filterRE *regexp.Regexp
		err      error
	)
	if filter != "" {
		filterRE, err = regexp.Compile(filter)
		if err != nil {
			return nil, err
		}
	}
	output, err := r.Git.container().WithExec([]string{"git", "ls-remote", "--tags", r.URL}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(output, "\n")
	tags := make([]*RemoteTag, 0, len(lines))
	for i := range lines {
		commit, name := tagSplit(lines[i])
		if name == "" {
			continue
		}
		if filterRE != nil {
			if !filterRE.MatchString(name) {
				continue
			}
		}
		tags = append(tags, &RemoteTag{
			Name:     name,
			CommitID: commit,
			URL:      r.URL,
			Git:      r.Git,
		})
	}
	return tags, nil
}

// A git tag
type RemoteTag struct {
	Name     string
	CommitID string
	URL      string

	// +private
	Git *Git
}

// Return the commit referenced by the remote tag
func (t *RemoteTag) Commit() *Commit {
	return t.Git.Init().
		WithCommand([]string{"fetch", t.URL, t.Name}, false).
		Commit(t.CommitID)
}

// Lookup a branch in the remote
func (r *Remote) Branch(ctx context.Context, name string) (*RemoteBranch, error) {
	output, err := r.Git.container().WithExec([]string{"git", "ls-remote", r.URL, name}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	line, _, _ := strings.Cut(output, "\n")
	commit, name := branchSplit(line)
	return &RemoteBranch{
		URL:      r.URL,
		CommitID: commit,
		Name:     name,
		Git:      r.Git,
	}, nil
}

// List available branches in the remote
func (r *Remote) Branches(
	ctx context.Context,
	// A regular expression to filter branch names. Only matching names are included.
	// +optional
	filter string,
) ([]*RemoteBranch, error) {
	var (
		filterRE *regexp.Regexp
		err      error
	)
	if filter != "" {
		filterRE, err = regexp.Compile(filter)
		if err != nil {
			return nil, err
		}
	}
	output, err := r.Git.container().WithExec([]string{"git", "ls-remote", "--heads", r.URL}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(output, "\n")
	branches := make([]*RemoteBranch, 0, len(lines))
	for i := range lines {
		commit, name := branchSplit(lines[i])
		if name == "" {
			continue
		}
		if filterRE != nil {
			if !filterRE.MatchString(name) {
				continue
			}
		}
		branches = append(branches, &RemoteBranch{
			Name:     name,
			CommitID: commit,
			URL:      r.URL,
			Git:      r.Git,
		})
	}
	return branches, nil
}

// A git branch
type RemoteBranch struct {
	Name     string
	CommitID string
	URL      string

	// +private
	Git *Git
}

// Return the commit referenced by the remote branch
func (b *RemoteBranch) Commit() *Commit {
	return b.Git.Init().
		WithCommand([]string{"fetch", b.URL, b.Name}, false).
		Commit(b.CommitID)
}

func refSplit(line, trimPrefix string) (string, string) {
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) == 0 {
		return "", ""
	}
	commit := parts[0]
	if len(parts) == 1 {
		return commit, ""
	}
	name := parts[1]
	if trimPrefix != "" {
		name = strings.TrimPrefix(parts[1], trimPrefix)
	}
	return commit, name
}

func tagSplit(line string) (string, string) {
	return refSplit(line, "refs/tags/")
}

func branchSplit(line string) (string, string) {
	return refSplit(line, "refs/heads/")
}
