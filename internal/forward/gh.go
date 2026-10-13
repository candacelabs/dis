package forward

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/mount"
)

// GHConfigTarget is where the host's GitHub CLI config is mounted in the
// container, independent of the container user's home directory.
const GHConfigTarget = "/dis/gh"

// ghHomeDir and ghGetenv resolve the host's gh config directory.
// Override in tests to use a temp directory.
var (
	ghHomeDir = os.UserHomeDir
	ghGetenv  = os.Getenv
)

// ghConfigDir is the host's GitHub CLI config directory, as gh resolves it:
// GH_CONFIG_DIR, then $XDG_CONFIG_HOME/gh, then ~/.config/gh.
func ghConfigDir() string {
	if dir := ghGetenv("GH_CONFIG_DIR"); dir != "" {
		return dir
	}
	if xdg := ghGetenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "gh")
	}
	home, err := ghHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "gh")
}

// GHMounts returns a Docker bind mount and env var forwarding the host's
// GitHub CLI login: the host's gh config directory, read-only, at
// GHConfigTarget, with GH_CONFIG_DIR pointing at it, so gh and anything
// that reads gh's config inside the container is logged in as the host user.
// Never returns an error for a missing config — just returns nothing.
func GHMounts() ([]mount.Mount, []string, error) {
	dir := ghConfigDir()
	if dir == "" {
		return nil, nil, nil
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, nil, nil
	}
	return []mount.Mount{{
		Type:     mount.TypeBind,
		Source:   dir,
		Target:   GHConfigTarget,
		ReadOnly: true,
	}}, []string{"GH_CONFIG_DIR=" + GHConfigTarget}, nil
}
