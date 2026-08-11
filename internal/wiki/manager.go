package wiki

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/google/shlex"
)

type Repo struct {
	Host  string
	Owner string
	Name  string
}

type Manager struct {
	Repo      Repo
	CachePath string
	Out       io.Writer
	Err       io.Writer
	git       gitRunner
}

type Snapshot struct {
	Branch string
	Dirty  bool
	Ahead  int
	Behind int
}

type repositoryMetadata struct {
	HasWiki bool `json:"has_wiki"`
}

func NewManager(repo Repo, cacheRoot string, out, errOut io.Writer) (*Manager, error) {
	path, err := CachePath(cacheRoot, repo.Host, repo.Owner, repo.Name)
	if err != nil {
		return nil, err
	}
	m := &Manager{Repo: repo, CachePath: path, Out: out, Err: errOut}
	m.git = gitRunner{out: out, err: errOut}
	return m, nil
}

func (m *Manager) cloneURL() string {
	return fmt.Sprintf("https://%s/%s/%s.wiki.git", m.Repo.Host, m.Repo.Owner, m.Repo.Name)
}

func (m *Manager) Ensure() error {
	gitDir := filepath.Join(m.CachePath, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return nil
	}
	if _, err := os.Stat(m.CachePath); err == nil {
		return fmt.Errorf("cache path exists but is not a wiki clone: %s", m.CachePath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect cache path: %w", err)
	}

	client, err := api.NewRESTClient(api.ClientOptions{Host: m.Repo.Host, Timeout: 20 * time.Second})
	if err != nil {
		return fmt.Errorf("initialize GitHub API client for %s: %w; run 'gh auth login --hostname %s'", m.Repo.Host, err, m.Repo.Host)
	}
	var metadata repositoryMetadata
	endpoint := fmt.Sprintf("repos/%s/%s", m.Repo.Owner, m.Repo.Name)
	if err := client.Get(endpoint, &metadata); err != nil {
		return fmt.Errorf("read repository %s/%s from %s: %w; check 'gh auth status --hostname %s' and your network", m.Repo.Owner, m.Repo.Name, m.Repo.Host, err, m.Repo.Host)
	}
	if !metadata.HasWiki {
		return fmt.Errorf("the wiki is disabled for %s/%s; enable Wikis in the repository settings first", m.Repo.Owner, m.Repo.Name)
	}
	if err := os.MkdirAll(filepath.Dir(m.CachePath), 0o700); err != nil {
		return fmt.Errorf("create cache parent: %w", err)
	}
	if _, err := m.git.output("", true, m.Repo.Host, "clone", "--origin", "origin", m.cloneURL(), m.CachePath); err != nil {
		_ = os.RemoveAll(m.CachePath)
		message := strings.ToLower(err.Error())
		switch {
		case strings.Contains(message, "repository not found"):
			return fmt.Errorf("the wiki for %s/%s has no pages yet; create its first page on GitHub, then try again", m.Repo.Owner, m.Repo.Name)
		case strings.Contains(message, "authentication failed"), strings.Contains(message, "could not read username"):
			return fmt.Errorf("authenticate to clone the wiki: %w; run 'gh auth login --hostname %s'", err, m.Repo.Host)
		case strings.Contains(message, "could not resolve"), strings.Contains(message, "failed to connect"), strings.Contains(message, "timed out"):
			return fmt.Errorf("network error while cloning the wiki: %w", err)
		default:
			return fmt.Errorf("clone wiki (it may not have an initial page yet): %w", err)
		}
	}
	fmt.Fprintf(m.Out, "Cached %s/%s wiki at %s\n", m.Repo.Owner, m.Repo.Name, m.CachePath)
	return nil
}

func (m *Manager) Fetch() error {
	if err := m.Ensure(); err != nil {
		return err
	}
	if _, err := m.git.output(m.CachePath, true, m.Repo.Host, "fetch", "--prune", "origin"); err != nil {
		return fmt.Errorf("fetch wiki updates: %w", err)
	}
	return nil
}

func (m *Manager) Snapshot() (Snapshot, error) {
	branchOut, err := m.git.output(m.CachePath, false, "", "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return Snapshot{}, fmt.Errorf("wiki cache is not on a branch: %w", err)
	}
	branch := strings.TrimSpace(branchOut)
	if _, err := m.git.output(m.CachePath, false, "", "rev-parse", "--verify", "@{upstream}"); err != nil {
		return Snapshot{}, fmt.Errorf("wiki branch %s has no upstream: %w", branch, err)
	}
	statusOut, err := m.git.output(m.CachePath, false, "", "status", "--porcelain")
	if err != nil {
		return Snapshot{}, err
	}
	countsOut, err := m.git.output(m.CachePath, false, "", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return Snapshot{}, err
	}
	fields := strings.Fields(countsOut)
	if len(fields) != 2 {
		return Snapshot{}, fmt.Errorf("unexpected git divergence output %q", strings.TrimSpace(countsOut))
	}
	ahead, err := strconv.Atoi(fields[0])
	if err != nil {
		return Snapshot{}, fmt.Errorf("parse ahead count: %w", err)
	}
	behind, err := strconv.Atoi(fields[1])
	if err != nil {
		return Snapshot{}, fmt.Errorf("parse behind count: %w", err)
	}
	return Snapshot{Branch: branch, Dirty: statusOut != "", Ahead: ahead, Behind: behind}, nil
}

func (m *Manager) Sync() (Snapshot, error) {
	if err := m.Fetch(); err != nil {
		return Snapshot{}, err
	}
	snapshot, err := m.Snapshot()
	if err != nil {
		return Snapshot{}, err
	}
	switch DecideSync(snapshot.Dirty, snapshot.Ahead, snapshot.Behind) {
	case SyncFastForward:
		if _, err := m.git.output(m.CachePath, false, "", "merge", "--ff-only", "@{upstream}"); err != nil {
			return snapshot, fmt.Errorf("fast-forward wiki cache: %w", err)
		}
		fmt.Fprintf(m.Out, "Updated %s (%d new commit(s)).\n", m.CachePath, snapshot.Behind)
		snapshot.Behind = 0
	case SyncDirtyBehind:
		return snapshot, fmt.Errorf("remote has %d new commit(s), but the cache has uncommitted edits; your edits were left untouched - run 'gh wiki push' to commit and reconcile them", snapshot.Behind)
	case SyncDiverged:
		return snapshot, fmt.Errorf("local and remote wiki histories diverged (%d ahead, %d behind); nothing was changed - run 'gh wiki push' to rebase and publish local commits", snapshot.Ahead, snapshot.Behind)
	case SyncAhead:
		fmt.Fprintf(m.Out, "Cache is %d commit(s) ahead; run 'gh wiki push' to publish.\n", snapshot.Ahead)
	case SyncCurrent:
		fmt.Fprintln(m.Out, "Wiki cache is up to date.")
	}
	return snapshot, nil
}

func (m *Manager) Commit(message string) error {
	if _, err := os.Stat(filepath.Join(m.CachePath, ".git", "rebase-merge")); err == nil {
		return fmt.Errorf("a rebase is in progress; resolve conflicts and run 'git -C %q rebase --continue' before pushing again", m.CachePath)
	}
	if _, err := os.Stat(filepath.Join(m.CachePath, ".git", "rebase-apply")); err == nil {
		return fmt.Errorf("a rebase is in progress; resolve conflicts and run 'git -C %q rebase --continue' before pushing again", m.CachePath)
	}
	if _, err := m.git.output(m.CachePath, false, "", "add", "--all"); err != nil {
		return fmt.Errorf("stage wiki edits: %w", err)
	}
	staged, err := m.git.output(m.CachePath, false, "", "diff", "--cached", "--name-only")
	if err != nil {
		return fmt.Errorf("inspect staged wiki edits: %w", err)
	}
	if strings.TrimSpace(staged) == "" {
		return nil
	}
	if message != "" {
		if _, err := m.git.output(m.CachePath, false, "", "commit", "--message", message); err != nil {
			return fmt.Errorf("commit wiki edits: %w", err)
		}
		return nil
	}
	fmt.Fprintln(m.Out, "Enter a commit message for the wiki edits.")
	if err := m.git.stream(m.CachePath, false, "", "commit"); err != nil {
		return fmt.Errorf("commit wiki edits: %w", err)
	}
	return nil
}

func (m *Manager) Push(message string) error {
	if err := m.Ensure(); err != nil {
		return err
	}
	if err := m.Commit(message); err != nil {
		return err
	}
	if err := m.Fetch(); err != nil {
		return err
	}
	if err := m.reconcile(); err != nil {
		return err
	}
	snapshot, err := m.Snapshot()
	if err != nil {
		return err
	}
	if snapshot.Ahead == 0 {
		fmt.Fprintln(m.Out, "Nothing to push; wiki is up to date.")
		return nil
	}
	if err := m.pushOnce(snapshot.Branch); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "non-fast-forward") && !strings.Contains(strings.ToLower(err.Error()), "fetch first") {
			return err
		}
		fmt.Fprintln(m.Out, "Remote changed during push; fetching and replaying local commits once.")
		if err := m.Fetch(); err != nil {
			return err
		}
		if err := m.reconcile(); err != nil {
			return err
		}
		if err := m.pushOnce(snapshot.Branch); err != nil {
			return fmt.Errorf("push still failed after reconciling once; no local work was discarded: %w", err)
		}
	}
	fmt.Fprintf(m.Out, "Pushed wiki changes to %s/%s.\n", m.Repo.Owner, m.Repo.Name)
	return nil
}

func (m *Manager) reconcile() error {
	snapshot, err := m.Snapshot()
	if err != nil {
		return err
	}
	if snapshot.Dirty {
		return fmt.Errorf("wiki cache still has uncommitted changes after commit; nothing was rebased or pushed")
	}
	if snapshot.Behind == 0 {
		return nil
	}
	if snapshot.Ahead == 0 {
		if _, err := m.git.output(m.CachePath, false, "", "merge", "--ff-only", "@{upstream}"); err != nil {
			return fmt.Errorf("fast-forward wiki before push: %w", err)
		}
		return nil
	}
	fmt.Fprintf(m.Out, "Remote has %d new commit(s); replaying %d local commit(s) on top.\n", snapshot.Behind, snapshot.Ahead)
	if err := m.git.stream(m.CachePath, false, "", "rebase", "@{upstream}"); err != nil {
		return fmt.Errorf("rebase stopped because changes conflict; your work is preserved in %s - resolve files, run 'git -C %q rebase --continue', then run 'gh wiki push' again", m.CachePath, m.CachePath)
	}
	return nil
}

func (m *Manager) pushOnce(branch string) error {
	if _, err := m.git.output(m.CachePath, true, m.Repo.Host, "push", "origin", "HEAD:"+branch); err != nil {
		return fmt.Errorf("push wiki: %w", err)
	}
	return nil
}

func (m *Manager) OpenEditor(editor string, forceCode bool) error {
	if err := m.Ensure(); err != nil {
		return err
	}
	if _, err := m.Sync(); err != nil {
		fmt.Fprintln(m.Err, "warning:", err)
		fmt.Fprintln(m.Err, "warning: opening the preserved local cache anyway")
	}
	return m.OpenPath(m.CachePath, editor, forceCode)
}

func (m *Manager) OpenPath(path, editor string, forceCode bool) error {
	command, args, err := editorCommand(editor, forceCode)
	if err != nil {
		return err
	}
	args = append(args, path)
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = m.Out
	cmd.Stderr = m.Err
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run editor %q: %w", command, err)
	}
	return nil
}

func editorCommand(explicit string, forceCode bool) (string, []string, error) {
	if forceCode {
		if _, err := exec.LookPath("code"); err != nil {
			return "", nil, fmt.Errorf("VS Code command 'code' was not found; install it from VS Code's Command Palette or use --editor")
		}
		return "code", []string{"--wait"}, nil
	}
	value := explicit
	if value == "" {
		value = os.Getenv("VISUAL")
	}
	if value == "" {
		value = os.Getenv("EDITOR")
	}
	if value == "" {
		if _, err := exec.LookPath("code"); err == nil {
			return "code", []string{"--wait"}, nil
		}
		value = "vi"
	}
	fields, err := shlex.Split(value)
	if err != nil {
		return "", nil, fmt.Errorf("parse editor command %q: %w", value, err)
	}
	if len(fields) == 0 {
		return "", nil, fmt.Errorf("editor command is empty")
	}
	return fields[0], fields[1:], nil
}
