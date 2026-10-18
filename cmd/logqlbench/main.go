// Command logqlbench measures LogQL throughput. It generates N synthetic
// logfmt lines in memory, then runs several representative queries and reports
// how many records per second each processes end to end (parse + filter +
// aggregate). Numbers are machine-dependent; treat them as relative.
//
// Usage: go run ./cmd/logqlbench [-n 1000000]
package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/kumar-ayush101/new--experiment"
)

func main() {
	n := flag.Int("n", 1_000_000, "number of log lines to generate")
	flag.Parse()

	fmt.Printf("Generating %d logfmt lines...\n", *n)
	data := generate(*n)
	fmt.Printf("Input size: %.1f MB\n\n", float64(len(data))/(1024*1024))

	queries := []struct{ name, q string }{
		{"filter only", `parse logfmt | where status >= 500`},
		{"filter + regex", `parse logfmt | where path =~ "/api" and status >= 500`},
		{"group + count", `parse logfmt | stats count by path`},
		{"group + sum/avg", `parse logfmt | stats sum(bytes) as total, avg(ms) by status`},
		{"top-N pipeline", `parse logfmt | where status >= 400 | stats count by path | sort count desc | limit 5`},
	}

	fmt.Printf("| query | time | records/sec |\n")
	fmt.Printf("|---|---|---|\n")
	for _, c := range queries {
		q, err := logql.Compile(c.q)
		if err != nil {
			panic(err)
		}
		start := time.Now()
		if _, err := q.RunString(data); err != nil {
			panic(err)
		}
		elapsed := time.Since(start)
		rps := float64(*n) / elapsed.Seconds()
		fmt.Printf("| %s | %s | %s |\n", c.name, elapsed.Truncate(time.Millisecond), humanRate(rps))
	}
}

func generate(n int) string {
	levels := []string{"info", "info", "info", "warn", "error"}
	statuses := []int{200, 200, 200, 404, 500, 503}
	paths := []string{"/", "/login", "/api/products", "/api/checkout", "/favicon.ico"}
	var b strings.Builder
	b.Grow(n * 64)
	for i := 0; i < n; i++ {
		lvl := levels[i%len(levels)]
		st := statuses[i%len(statuses)]
		path := paths[i%len(paths)]
		fmt.Fprintf(&b, "level=%s status=%d method=GET path=%s bytes=%d ms=%d\n",
			lvl, st, path, (i%5000)*8, i%300)
	}
	return b.String()
}

func humanRate(r float64) string {
	switch {
	case r >= 1e6:
		return fmt.Sprintf("%.1fM", r/1e6)
	case r >= 1e3:
		return fmt.Sprintf("%.0fK", r/1e3)
	default:
		return fmt.Sprintf("%.0f", r)
	}
}
