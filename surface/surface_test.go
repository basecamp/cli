package surface

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func newTestRoot() *cobra.Command {
	root := &cobra.Command{
		Use: "mycli",
	}
	root.PersistentFlags().Bool("json", false, "JSON output")
	root.PersistentFlags().Bool("verbose", false, "Verbose output")

	projects := &cobra.Command{Use: "projects", Short: "Manage projects"}
	projects.Flags().Int("limit", 50, "Limit results")

	list := &cobra.Command{Use: "list", Short: "List projects"}
	list.Flags().Bool("all", false, "Show all")

	show := &cobra.Command{Use: "show", Short: "Show project"}
	show.Flags().String("format", "table", "Output format")

	projects.AddCommand(list, show)
	root.AddCommand(projects)

	hidden := &cobra.Command{Use: "internal", Hidden: true}
	root.AddCommand(hidden)

	return root
}

func TestSnapshot(t *testing.T) {
	root := newTestRoot()
	entries := Snapshot(root)

	// Should have entries for all visible commands
	var cmds []string
	for _, e := range entries {
		if e.Kind == KindCmd {
			cmds = append(cmds, e.Path)
		}
	}

	assert.Contains(t, cmds, "mycli")
	assert.Contains(t, cmds, "mycli projects")
	assert.Contains(t, cmds, "mycli projects list")
	assert.Contains(t, cmds, "mycli projects show")
}

func TestSnapshotFlags(t *testing.T) {
	root := newTestRoot()
	entries := Snapshot(root)

	var flags []string
	for _, e := range entries {
		if e.Kind == KindFlag {
			flags = append(flags, e.String())
		}
	}

	assert.Contains(t, flags, "FLAG mycli --json type=bool")
	assert.Contains(t, flags, "FLAG mycli --verbose type=bool")
	assert.Contains(t, flags, "FLAG mycli projects --limit type=int")
	assert.Contains(t, flags, "FLAG mycli projects list --all type=bool")
}

func TestSnapshotSubcommands(t *testing.T) {
	root := newTestRoot()
	entries := Snapshot(root)

	var subs []string
	for _, e := range entries {
		if e.Kind == KindSub {
			subs = append(subs, e.String())
		}
	}

	assert.Contains(t, subs, "SUB mycli projects")
	assert.Contains(t, subs, "SUB mycli projects list")
	assert.Contains(t, subs, "SUB mycli projects show")
}

func TestSnapshotHiddenExcluded(t *testing.T) {
	root := newTestRoot()
	entries := Snapshot(root)

	for _, e := range entries {
		assert.NotContains(t, e.Path, "internal", "hidden commands should be excluded")
	}
}

func TestSnapshotString(t *testing.T) {
	root := newTestRoot()
	s := SnapshotString(root)

	assert.NotEmpty(t, s)

	// Should be sorted
	lines := splitLines(s)
	for i := 1; i < len(lines); i++ {
		assert.True(t, lines[i-1] <= lines[i], "lines should be sorted: %q > %q", lines[i-1], lines[i])
	}
}

func TestDiffIdentical(t *testing.T) {
	root := newTestRoot()
	entries := Snapshot(root)

	result := Diff(entries, entries)
	assert.Empty(t, result.Added)
	assert.Empty(t, result.Removed)
	assert.False(t, result.HasBreakingChanges())
}

func TestDiffAdditions(t *testing.T) {
	root1 := newTestRoot()
	old := Snapshot(root1)

	root2 := newTestRoot()
	root2.AddCommand(&cobra.Command{Use: "newcmd", Short: "New command"})
	new := Snapshot(root2)

	result := Diff(old, new)
	assert.NotEmpty(t, result.Added)
	assert.Empty(t, result.Removed)
	assert.False(t, result.HasBreakingChanges())

	// Check specific addition
	var addedCmds []string
	for _, e := range result.Added {
		if e.Kind == KindCmd {
			addedCmds = append(addedCmds, e.Path)
		}
	}
	assert.Contains(t, addedCmds, "mycli newcmd")
}

func TestDiffRemovals(t *testing.T) {
	root1 := newTestRoot()
	root1.AddCommand(&cobra.Command{Use: "oldcmd"})
	old := Snapshot(root1)

	root2 := newTestRoot()
	new := Snapshot(root2)

	result := Diff(old, new)
	assert.NotEmpty(t, result.Removed)
	assert.True(t, result.HasBreakingChanges())
}

func TestEntryString(t *testing.T) {
	tests := []struct {
		entry    Entry
		expected string
	}{
		{Entry{Kind: KindCmd, Path: "mycli"}, "CMD mycli"},
		{Entry{Kind: KindFlag, Path: "mycli", Name: "json", FlagType: "bool"}, "FLAG mycli --json type=bool"},
		{Entry{Kind: KindSub, Path: "mycli", Name: "projects"}, "SUB mycli projects"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entry.String())
		})
	}
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
