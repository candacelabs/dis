package forward

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/mount"
)

// HistoryVolume returns a named Docker volume mount for persisting shell history.
// The volume name is "dis-<projectName>-history".
// The target path depends on the shell:
//   - bash (or /bin/bash) → /root/.bash_history
//   - zsh  (or /bin/zsh)  → /root/.zsh_history
//   - anything else        → /root/.bash_history (default)
func HistoryVolume(projectName string, shell string) mount.Mount {
	return mount.Mount{
		Type:   mount.TypeVolume,
		Source: fmt.Sprintf("dis-%s-history", projectName),
		Target: historyTarget(shell),
	}
}

// historyTarget maps a shell name or path to the corresponding history file path.
func historyTarget(shell string) string {
	// Accept both "/bin/zsh" and "zsh" forms.
	base := strings.ToLower(filepath.Base(shell))
	switch base {
	case "zsh":
		return "/root/.zsh_history"
	default:
		return "/root/.bash_history"
	}
}
