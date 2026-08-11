package cmd

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/paul/gh-wiki/internal/wiki"
	"github.com/spf13/cobra"
)

var version = "dev"

type options struct {
	repository string
}

func Execute() error {
	opts := &options{}
	root := &cobra.Command{
		Use:           "gh wiki",
		Short:         "Work with a GitHub repository wiki from a managed local cache",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
		Long: `Work with a GitHub wiki as local files.

The wiki is cloned into a per-repository cache on first use. Commands use the
repository in the current directory unless --repo names another repository.`,
		Example: `  gh wiki edit
  gh wiki edit --code
  gh wiki new "Release process"
  gh wiki push -m "Document the release process"`,
	}
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.PersistentFlags().StringVarP(&opts.repository, "repo", "R", "", "Select another repository using [HOST/]OWNER/REPO")
	root.AddCommand(
		newSyncCommand(opts),
		newEditCommand(opts),
		newPageCommand(opts),
		newPushCommand(opts),
		newStatusCommand(opts),
		newPathCommand(opts),
	)
	root.CompletionOptions.DisableDefaultCmd = true
	return root.Execute()
}

func managerFor(opts *options) (*wiki.Manager, error) {
	var resolved repository.Repository
	var err error
	if opts.repository != "" {
		resolved, err = repository.Parse(opts.repository)
	} else {
		resolved, err = repository.Current()
	}
	if err != nil {
		return nil, fmt.Errorf("resolve repository: %w; run from a GitHub repository or pass --repo OWNER/REPO", err)
	}
	root, err := wiki.CacheRoot()
	if err != nil {
		return nil, err
	}
	return wiki.NewManager(wiki.Repo{Host: resolved.Host, Owner: resolved.Owner, Name: resolved.Name}, root, os.Stdout, os.Stderr)
}

func newSyncCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Fetch remote wiki changes and safely update the local cache",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := managerFor(opts)
			if err != nil {
				return err
			}
			_, err = manager.Sync()
			return err
		},
	}
}

func newEditCommand(opts *options) *cobra.Command {
	var editor string
	var code bool
	command := &cobra.Command{
		Use:   "edit",
		Short: "Sync the wiki cache and open it in your editor",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := managerFor(opts)
			if err != nil {
				return err
			}
			return manager.OpenEditor(editor, code)
		},
	}
	command.Flags().StringVarP(&editor, "editor", "e", "", "Editor command (default: $VISUAL, $EDITOR, code, then vi)")
	command.Flags().BoolVar(&code, "code", false, "Open in VS Code and wait for it to close")
	return command
}

func newPageCommand(opts *options) *cobra.Command {
	var editor string
	var code bool
	var noEdit bool
	command := &cobra.Command{
		Use:   "new TITLE",
		Short: "Create a Markdown wiki page and open it in your editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			manager, err := managerFor(opts)
			if err != nil {
				return err
			}
			if err := manager.Ensure(); err != nil {
				return err
			}
			if _, err := manager.Sync(); err != nil {
				fmt.Fprintln(os.Stderr, "warning:", err)
				fmt.Fprintln(os.Stderr, "warning: creating the page in the preserved local cache anyway")
			}
			path, err := manager.CreatePage(args[0])
			if err != nil {
				return err
			}
			fmt.Println("Created", path)
			if noEdit {
				return nil
			}
			return manager.OpenPath(path, editor, code)
		},
	}
	command.Flags().StringVarP(&editor, "editor", "e", "", "Editor command (default: $VISUAL, $EDITOR, code, then vi)")
	command.Flags().BoolVar(&code, "code", false, "Open in VS Code and wait for it to close")
	command.Flags().BoolVar(&noEdit, "no-edit", false, "Create the page without opening an editor")
	return command
}

func newPushCommand(opts *options) *cobra.Command {
	var message string
	command := &cobra.Command{
		Use:   "push",
		Short: "Commit local edits, reconcile remote changes, and push the wiki",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := managerFor(opts)
			if err != nil {
				return err
			}
			return manager.Push(message)
		},
	}
	command.Flags().StringVarP(&message, "message", "m", "", "Commit message (opens Git's editor when omitted)")
	return command
}

func newStatusCommand(opts *options) *cobra.Command {
	var noFetch bool
	command := &cobra.Command{
		Use:   "status",
		Short: "Show the cache path, edits, and local/remote divergence",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := managerFor(opts)
			if err != nil {
				return err
			}
			if err := manager.Ensure(); err != nil {
				return err
			}
			if !noFetch {
				if err := manager.Fetch(); err != nil {
					return err
				}
			}
			snapshot, err := manager.Snapshot()
			if err != nil {
				return err
			}
			state := "clean"
			if snapshot.Dirty {
				state = "modified"
			}
			fmt.Printf("Repository: %s/%s\nCache:      %s\nBranch:     %s\nFiles:      %s\nAhead:      %d\nBehind:     %d\n", manager.Repo.Owner, manager.Repo.Name, manager.CachePath, snapshot.Branch, state, snapshot.Ahead, snapshot.Behind)
			return nil
		},
	}
	command.Flags().BoolVar(&noFetch, "no-fetch", false, "Report cached state without contacting GitHub")
	return command
}

func newPathCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Ensure the wiki is cached and print its local path",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := managerFor(opts)
			if err != nil {
				return err
			}
			if err := manager.Ensure(); err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, manager.CachePath)
			return nil
		},
	}
}
