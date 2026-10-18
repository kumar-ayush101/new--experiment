package source

import (
	"testing"

	"github.com/kumar-ayush101/new--experiment/internal/ast"
	"github.com/kumar-ayush101/new--experiment/internal/record"
)

func TestLogfmt(t *testing.T) {
	p, _ := New(ast.Source{Format: ast.Logfmt})
	rec, ok := p.Parse(`level=error status=500 path=/api msg="db timeout" cached`)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if rec["level"].String() != "error" {
		t.Fatalf("level = %q", rec["level"].String())
	}
	// status should promote to a number
	if n, isNum := rec["status"].AsNumber(); !isNum || n != 500 {
		t.Fatalf("status = %+v", rec["status"])
	}
	if rec["msg"].String() != "db timeout" {
		t.Fatalf("msg = %q", rec["msg"].String())
	}
	if rec["cached"].Kind != record.Bool || !rec["cached"].B {
		t.Fatalf("cached bare flag = %+v", rec["cached"])
	}
	if _, ok := p.Parse("   "); ok {
		t.Fatal("blank line should be skipped")
	}
}

func TestJSON(t *testing.T) {
	p, _ := New(ast.Source{Format: ast.JSON})
	rec, ok := p.Parse(`{"status":404,"path":"/x","ok":false,"latency":12.5}`)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if n, _ := rec["status"].AsNumber(); n != 404 {
		t.Fatalf("status = %+v", rec["status"])
	}
	if rec["ok"].Kind != record.Bool || rec["ok"].B {
		t.Fatalf("ok = %+v", rec["ok"])
	}
	if n, _ := rec["latency"].AsNumber(); n != 12.5 {
		t.Fatalf("latency = %+v", rec["latency"])
	}
	if _, ok := p.Parse(`not json`); ok {
		t.Fatal("invalid json should be skipped")
	}
}

func TestRegex(t *testing.T) {
	p, err := New(ast.Source{Format: ast.Regex, Pattern: `(?P<ip>\S+) \S+ \S+ \[[^\]]+\] "(?P<method>\S+) (?P<path>\S+)[^"]*" (?P<status>\d+)`})
	if err != nil {
		t.Fatalf("New regex: %v", err)
	}
	line := `192.168.1.1 - - [10/Oct/2026:13:55:36] "GET /index.html HTTP/1.1" 200`
	rec, ok := p.Parse(line)
	if !ok {
		t.Fatal("expected regex match")
	}
	if rec["ip"].String() != "192.168.1.1" {
		t.Fatalf("ip = %q", rec["ip"].String())
	}
	if rec["method"].String() != "GET" {
		t.Fatalf("method = %q", rec["method"].String())
	}
	if n, _ := rec["status"].AsNumber(); n != 200 {
		t.Fatalf("status = %+v", rec["status"])
	}
	if _, ok := p.Parse("no match here"); ok {
		t.Fatal("non-matching line should be skipped")
	}
}

func TestRegexInvalid(t *testing.T) {
	if _, err := New(ast.Source{Format: ast.Regex, Pattern: `(?P<x>`}); err == nil {
		t.Fatal("expected error for invalid regex")
	}
}
