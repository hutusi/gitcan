package main

import (
	"os"
	"fmt"
	"log"
	"errors"
	git "github.com/libgit2/git2go/v31"
)

func lookupCommit(repo *git.Repository, ref string) (*git.Commit, error) {
	object, err := repo.RevparseSingle(ref)
	if err != nil {
		return nil, fmt.Errorf("could not lookup reference %q: %w", ref, err)
	}

	peeled, err := object.Peel(git.ObjectCommit)
	if err != nil {
		return nil, fmt.Errorf("could not peel reference %q: %w", ref, err)
	}

	commit, err := peeled.AsCommit()
	if err != nil {
		return nil, fmt.Errorf("reference %q is not a commit: %w", ref, err)
	}

	return commit, nil
}

func merge(path string, ourCommit string, theirCommit string, message string) (string, error) {
	repo, err := git.OpenRepository(path)
	if err != nil {
		return "", fmt.Errorf("could not open repository: %w", err)
	}
	defer repo.Free()

	ours, err := lookupCommit(repo, ourCommit)
	if err != nil {
		return "", fmt.Errorf("could not lookup commit %q: %w", ourCommit, err)
	}

	theirs, err := lookupCommit(repo, theirCommit)
	if err != nil {
		return "", fmt.Errorf("could not lookup commit %q: %w", theirCommit, err)
	}

	mergeOpts, err := git.DefaultMergeOptions()
	if err != nil {
		return "", fmt.Errorf("could not create merge options: %w", err)
	}

	index, err := repo.MergeCommits(ours, theirs, &mergeOpts)
	if err != nil {
		return "", fmt.Errorf("could not merge commits: %w", err)
	}
	defer index.Free()

	if index.HasConflicts() {
		return "", errors.New("could not auto-merge due to conflicts")
	}

	tree, err := index.WriteTreeTo(repo)
	if err != nil {
		return "", fmt.Errorf("could not write tree: %w", err)
	}

	committer, err := repo.DefaultSignature()
	if err != nil {
		return "", fmt.Errorf("could not get default signature: %w", err)
	}

	commit, err := repo.CreateCommitFromIds("", committer, committer, message, tree, ours.Id(), theirs.Id())
	if err != nil {
		return "", fmt.Errorf("could not create merge commit: %w", err)
	}

	return commit.String(), nil
}

func main() {
	args := os.Args[1:]
	if len(args) < 4 {
		log.Fatal("missing args...")
	}
	commit, err := merge(args[0], args[1], args[2], args[3])
	if err != nil {
		log.Fatalf("fail to merge: %v\n", err)
	}

	log.Printf("merge %v and %v into new commit: %v\n", args[1], args[2], commit)
}
