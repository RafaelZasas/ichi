package views

import (
	"fmt"
	"testing"

	"github.com/atterpac/dado/components"
	"github.com/atterpac/ichi/internal/git"
)

// makeStatusEntries builds n synthetic working-tree entries spread across a
// directory hierarchy `dirs` levels deep with `breadth` siblings per level,
// approximating a real repo's path distribution.
func makeStatusEntries(n, dirs, breadth int) []git.StatusEntry {
	entries := make([]git.StatusEntry, n)
	for i := 0; i < n; i++ {
		path := ""
		d := i
		for l := 0; l < dirs; l++ {
			path += fmt.Sprintf("dir%d/", d%breadth)
			d /= breadth
		}
		path += fmt.Sprintf("file%d.go", i)
		entries[i] = git.StatusEntry{
			Path:        path,
			WorkStatus:  git.FileModified,
			IndexStatus: git.FileUnchanged,
		}
	}
	return entries
}

// makeLeaves builds throwaway leaf nodes for the entries so the benchmark
// measures only the directory-grouping structural work (groupFilesIntoTree),
// not git I/O or ParseDiff.
func makeLeaves(entries []git.StatusEntry) []*components.TreeNode {
	leaves := make([]*components.TreeNode, len(entries))
	for i := range entries {
		leaves[i] = &components.TreeNode{ID: entries[i].Path, Label: entries[i].Path}
	}
	return leaves
}

func BenchmarkGroupFilesIntoTree(b *testing.B) {
	cases := []struct {
		name             string
		n, dirs, breadth int
		flat             bool
	}{
		{"flat_100", 100, 0, 1, true},
		{"nested_100_shallow", 100, 2, 4, false},
		{"nested_500_deep", 500, 4, 5, false},
		{"nested_2000_deep", 2000, 5, 6, false},
	}
	for _, c := range cases {
		entries := makeStatusEntries(c.n, c.dirs, c.breadth)
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Leaves are rebuilt each iter: a real staging refresh allocates
				// fresh nodes too, and reusing them would let AddChild corrupt
				// sibling links across iterations.
				leaves := makeLeaves(entries)
				_ = groupFilesIntoTree("root", "Root", entries, leaves, false, c.flat)
			}
		})
	}
}

// BenchmarkGroupFilesIntoTreeNoLeaves isolates the directory-grouping cost from
// leaf allocation by reusing a single leaf set (single-shot, not looped).
func BenchmarkGroupFilesIntoTreeStructureOnly(b *testing.B) {
	entries := makeStatusEntries(2000, 5, 6)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		leaves := makeLeaves(entries)
		b.StartTimer()
		_ = groupFilesIntoTree("root", "Root", entries, leaves, false, false)
	}
}
