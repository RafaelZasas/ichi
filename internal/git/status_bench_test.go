package git

import (
	"fmt"
	"strings"
	"testing"
)

// makeStatusV2 builds synthetic `git status --porcelain=v2` output mixing
// ordinary, renamed, untracked, and conflict entries.
func makeStatusV2(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		path := fmt.Sprintf("pkg/module%d/file%d.go", i%8, i)
		switch i % 5 {
		case 0:
			sb.WriteString(fmt.Sprintf("1 M. N... 100644 100644 100644 1111111 2222222 %s\n", path))
		case 1:
			sb.WriteString(fmt.Sprintf("1 .M N... 100644 100644 100644 1111111 2222222 %s\n", path))
		case 2:
			sb.WriteString(fmt.Sprintf("2 R. N... 100644 100644 100644 1111111 2222222 R100 %s\told/%s\n", path, path))
		case 3:
			sb.WriteString(fmt.Sprintf("? %s\n", path))
		case 4:
			sb.WriteString(fmt.Sprintf("u UU N... 100644 100644 100644 100644 1111111 2222222 3333333 %s\n", path))
		}
	}
	return sb.String()
}

func BenchmarkParseStatusV2(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		out := makeStatusV2(n)
		b.Run(fmt.Sprintf("entries_%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(out)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = parseStatusV2(out)
			}
		})
	}
}
