package git

import (
	"fmt"
	"strings"
	"testing"
)

// makeDiff builds a synthetic unified-diff string with the given number of
// files, hunks per file, and lines per hunk. It mirrors the shape of real
// `git diff` output so ParseDiff exercises every branch.
func makeDiff(files, hunksPerFile, linesPerHunk int) string {
	var sb strings.Builder
	for f := 0; f < files; f++ {
		path := fmt.Sprintf("pkg/module%d/file%d.go", f%8, f)
		sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", path, path))
		sb.WriteString("index 1111111..2222222 100644\n")
		sb.WriteString(fmt.Sprintf("--- a/%s\n", path))
		sb.WriteString(fmt.Sprintf("+++ b/%s\n", path))
		for h := 0; h < hunksPerFile; h++ {
			start := 1 + h*64
			sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@ func Thing%d() {\n",
				start, linesPerHunk, start, linesPerHunk, h))
			for l := 0; l < linesPerHunk; l++ {
				switch l % 4 {
				case 0:
					sb.WriteString(fmt.Sprintf("-\told line %d here\n", l))
				case 1:
					sb.WriteString(fmt.Sprintf("+\tnew line %d here\n", l))
				default:
					sb.WriteString(fmt.Sprintf(" \tcontext line %d unchanged\n", l))
				}
			}
		}
	}
	return sb.String()
}

func BenchmarkParseDiff(b *testing.B) {
	cases := []struct {
		name                         string
		files, hunksPerFile, perHunk int
	}{
		{"single_small", 1, 1, 10},
		{"single_large", 1, 8, 120},
		{"many_files", 50, 2, 30},
		{"huge", 200, 4, 80},
	}
	for _, c := range cases {
		diff := makeDiff(c.files, c.hunksPerFile, c.perHunk)
		b.Run(c.name, func(b *testing.B) {
			b.SetBytes(int64(len(diff)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := ParseDiff(diff); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// benchHunk builds one parsed hunk of n lines for patch-generation benchmarks.
func benchHunk(n int) *DiffHunk {
	files, _ := ParseDiff(makeDiff(1, 1, n))
	return files[0].Hunks[0]
}

func BenchmarkGenerateHunkPatch(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		hunk := benchHunk(n)
		b.Run(fmt.Sprintf("lines_%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = GenerateHunkPatch("pkg/file.go", hunk)
			}
		})
	}
}

func BenchmarkGenerateLinesPatch(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		hunk := benchHunk(n)
		// Select roughly half the change lines.
		var sel []*DiffLine
		for _, l := range hunk.Lines {
			if l.Type == LineAdded || l.Type == LineRemoved {
				sel = append(sel, l)
			}
		}
		sel = sel[:len(sel)/2+1]
		b.Run(fmt.Sprintf("lines_%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = GenerateLinesPatch("pkg/file.go", hunk, sel)
			}
		})
	}
}

func BenchmarkComputeLineNumbers(b *testing.B) {
	hunk := benchHunk(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		computeLineNumbers(hunk)
	}
}
