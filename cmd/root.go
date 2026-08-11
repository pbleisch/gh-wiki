package cmd

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/pbleisch/gh-wiki/internal/wiki"
	"github.com/spf13/cobra"
)

type options struct {
	repository string
}

func Execute() error {
	opts := &options{}
	buildInfo, hasBuildInfo := debug.ReadBuildInfo()
	root := &cobra.Command{
		Use:   "gh-wiki",
		Short: "Work with a GitHub repository wiki from a managed local cache",
		Annotations: map[string]string{
			cobra.CommandDisplayNameAnnotation: "gh wiki",
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       resolveVersion(version, buildInfo, hasBuildInfo),
		Long: `Work with a GitHub wiki as local files.

A GitHub wiki is a separate Git repository. This extension keeps one clone of it
per repository in your user cache directory, opens that clone in your editor,
and reconciles remote changes before pushing so concurrent edits are never lost.

Commands use the repository in the current directory unless --repo names another
repository. Progress messages go to standard error, so 'list', 'path', and
'status' can be read by other programs.`,
		Example: `  gh wiki list
  gh wiki edit
  gh wiki edit --code
  gh wiki new "Release process"
  gh wiki push -m "Document the release process"
  gh wiki status --repo cli/cli`,
	}
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.PersistentFlags().StringVarP(&opts.repository, "repo", "R", "", "Select another repository using [HOST/]OWNER/REPO")
	root.AddCommand(
		newListCommand(opts),
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

func newListCommand(opts *options) *cobra.Command {
	var noFetch bool
	var showFiles bool
	var all bool
	command := &cobra.Command{
		Use:   "list",
		Short: "List the wiki's pages",
		Long: `List the wiki's pages, one per line and sorted by title.

GitHub identifies a wiki page by its filename alone. A page committed inside a
directory is therefore listed under its own title with no directory prefix, and
two files whose names differ only by directory, capitalization, or markup
extension name the same GitHub page; those are reported as warnings on standard
error because GitHub serves only one of them.`,
		Example: `  gh wiki list
  gh wiki list --files
  gh wiki list --no-fetch --repo cli/cli`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, err := managerFor(opts)
			if err != nil {
				return err
			}
			if err := manager.Ensure(); err != nil {
				return err
			}
			if !noFetch {
				if _, err := manager.Sync(); err != nil {
					fmt.Fprintln(os.Stderr, "warning:", err)
					fmt.Fprintln(os.Stderr, "warning: listing the preserved local cache anyway")
				}
			}
			pages, err := manager.Pages()
			if err != nil {
				return err
			}
			for _, group := range wiki.Collisions(pages) {
				paths := make([]string, 0, len(group))
				for _, page := range group {
					paths = append(paths, page.Path)
				}
				fmt.Fprintf(os.Stderr, "warning: %s name the same wiki page %q; GitHub serves only one of them\n", strings.Join(paths, " and "), group[0].Title)
			}
			for _, page := range pages {
				if page.Special != "" && !all {
					continue
				}
				if showFiles {
					fmt.Fprintln(os.Stdout, page.Path)
					continue
				}
				fmt.Fprintln(os.Stdout, page.Title)
			}
			return nil
		},
	}
	command.Flags().BoolVar(&showFiles, "files", false, "Print each page's file path instead of its title")
	command.Flags().BoolVar(&all, "all", false, "Include GitHub's _Sidebar and _Footer layout files")
	command.Flags().BoolVar(&noFetch, "no-fetch", false, "List the cached pages without contacting GitHub")
	return command
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
		Long: `Create a Markdown wiki page and open it in your editor.

TITLE is converted to the filename GitHub expects: spaces become hyphens, so
"Release process" is created as Release-process.md. GitHub does not allow
\ / : * ? " < > | in a wiki title and its own editor silently strips them, so
this command reports the problem instead of renaming the page for you.`,
		Example: `  gh wiki new "Release process"
  gh wiki new Home --no-edit`,
		Args: cobra.ExactArgs(1),
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
