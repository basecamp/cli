package surface

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// EntryKind identifies the type of surface entry.
type EntryKind string

const (
	KindCmd  EntryKind = "CMD"
	KindFlag EntryKind = "FLAG"
	KindSub  EntryKind = "SUB"
)

// Entry represents a single element in the CLI surface.
type Entry struct {
	Kind     EntryKind
	Path     string // Full command path (e.g., "basecamp projects list")
	Name     string // Flag or subcommand name
	FlagType string // Flag type (e.g., "string", "bool") — only for FLAG entries
}

// String returns the canonical string representation of the entry.
// Format: "CMD path", "FLAG path --name type=flagtype", "SUB path name"
func (e Entry) String() string {
	switch e.Kind {
	case KindCmd:
		return fmt.Sprintf("CMD %s", e.Path)
	case KindFlag:
		return fmt.Sprintf("FLAG %s --%s type=%s", e.Path, e.Name, e.FlagType)
	case KindSub:
		return fmt.Sprintf("SUB %s %s", e.Path, e.Name)
	default:
		return fmt.Sprintf("%s %s %s", e.Kind, e.Path, e.Name)
	}
}

// Snapshot walks a Cobra command tree and returns all surface entries.
func Snapshot(cmd *cobra.Command) []Entry {
	var entries []Entry
	walkCommand(cmd, cmd.Name(), &entries)
	return entries
}

// SnapshotString returns a sorted, newline-joined string of all surface entries.
func SnapshotString(cmd *cobra.Command) string {
	entries := Snapshot(cmd)
	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = e.String()
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func walkCommand(cmd *cobra.Command, path string, entries *[]Entry) {
	// Emit CMD entry
	*entries = append(*entries, Entry{Kind: KindCmd, Path: path})

	// Collect and sort all flags visible at this command level:
	// local flags, persistent flags on this command, and inherited persistent flags.
	var flags []Entry
	seen := make(map[string]bool)
	addFlag := func(f *pflag.Flag) {
		if seen[f.Name] || f.Hidden {
			return
		}
		seen[f.Name] = true
		flags = append(flags, Entry{
			Kind:     KindFlag,
			Path:     path,
			Name:     f.Name,
			FlagType: f.Value.Type(),
		})
	}
	cmd.Flags().VisitAll(addFlag)
	cmd.PersistentFlags().VisitAll(addFlag)
	if cmd.HasParent() {
		cmd.InheritedFlags().VisitAll(addFlag)
	}
	sort.Slice(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	*entries = append(*entries, flags...)

	// Collect and sort subcommands
	subs := cmd.Commands()
	sort.Slice(subs, func(i, j int) bool { return subs[i].Name() < subs[j].Name() })

	for _, sub := range subs {
		if sub.Hidden {
			continue
		}
		*entries = append(*entries, Entry{
			Kind: KindSub,
			Path: path,
			Name: sub.Name(),
		})
		walkCommand(sub, path+" "+sub.Name(), entries)
	}
}
