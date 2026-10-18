package vm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kumar-ayush101/new--experiment/internal/ast"
	"github.com/kumar-ayush101/new--experiment/internal/compiler"
	"github.com/kumar-ayush101/new--experiment/internal/parser"
	"github.com/kumar-ayush101/new--experiment/internal/source"
)

func runFile(t *testing.T, query, file string) *Result {
	t.Helper()
	q, err := parser.Parse(query)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	prog, err := compiler.Compile(q)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	lp, err := source.New(q.Source)
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	f, err := os.Open(file)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	res, err := New(prog, lp).Run(f)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return res
}

func TestVMOverSampleFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.log")

	// Top error paths by count.
	res := runFile(t, `parse logfmt | where status >= 500 | stats count by path | sort count desc`, path)
	if len(res.Rows) == 0 {
		t.Fatal("expected error-path groups")
	}
	// /api/checkout has 2 5xx, /api/products has 1 -> checkout first.
	if res.Rows[0]["path"].String() != "/api/checkout" {
		t.Fatalf("top error path = %q, want /api/checkout", res.Rows[0]["path"].String())
	}

	// Total bytes by status for 200s.
	res2 := runFile(t, `parse logfmt | where status == 200 | stats sum(bytes) as total`, path)
	if len(res2.Rows) != 1 {
		t.Fatalf("expected single aggregate row, got %d", len(res2.Rows))
	}
	// 512+1024+4096+4096+512 = 10240
	if n, _ := res2.Rows[0]["total"].AsNumber(); n != 10240 {
		t.Fatalf("sum bytes = %v, want 10240", n)
	}
}

func TestVMLimitEarlyExit(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.log")
	res := runFile(t, `parse logfmt | where status == 200 | limit 2`, path)
	if len(res.Rows) != 2 {
		t.Fatalf("limit rows = %d, want 2", len(res.Rows))
	}
}

func TestVMEmptyStatsHasZeroRow(t *testing.T) {
	// Build a program with a stats over an empty filtered stream.
	q, _ := parser.Parse(`parse logfmt | where status == 999 | stats count`)
	prog, _ := compiler.Compile(q)
	lp, _ := source.New(ast.Source{Format: ast.Logfmt})
	res, err := New(prog, lp).RunString("level=info status=200\n")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("empty stats rows = %d, want 1 (zero row)", len(res.Rows))
	}
	if n, _ := res.Rows[0]["count"].AsNumber(); n != 0 {
		t.Fatalf("count = %v, want 0", n)
	}
}
