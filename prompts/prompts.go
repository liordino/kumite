// Package prompts makes the three immutable prompt artifacts available to the
// engine regardless of the process working directory.
//
// The .md files stay the single source of truth in the repository and are
// never modified during implementation. At runtime, os.ReadFile wins when the
// file exists (so prompts stay live-editable in a working checkout); the
// build-time embedded copy is the fallback for test binaries and installed
// builds where the file is not on disk.
package prompts

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed shisho.md
var shisho string

//go:embed uchikomi.md
var uchikomi string

//go:embed agent_template.md
var agentTemplate string

// Load returns a prompt artifact by its file name, preferring the on-disk
// file under prompts/ and falling back to the embedded copy.
func Load(name string, embedded string) (string, error) {
	if b, err := os.ReadFile(filepath.Join("prompts", name)); err == nil {
		return string(b), nil
	}
	if embedded == "" {
		return "", os.ErrNotExist
	}
	return embedded, nil
}

// Shisho returns the orchestrator prompt (Phase 0 + Phase 2 + handoff).
func Shisho() (string, error) { return Load("shisho.md", shisho) }

// Uchikomi returns the intake distillation prompt.
func Uchikomi() (string, error) { return Load("uchikomi.md", uchikomi) }

// AgentTemplate returns the specialist prompt envelope.
func AgentTemplate() (string, error) { return Load("agent_template.md", agentTemplate) }
