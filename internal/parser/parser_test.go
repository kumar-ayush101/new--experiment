package parser

import (
	"testing"

	"github.com/kumar-ayush101/new--experiment/internal/ast"
)

func TestParseFullPipeline(t *testing.T) {
	q, err := Parse(`parse logfmt | where status >= 500 | stats count by path | sort count desc | limit 10`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if q.Source.Format != ast.Logfmt {
		t.Fatalf("source format = %v, want Logfmt", q.Source.Format)
	}
	if len(q.Stages) != 4 {
		t.Fatalf("stage count = %d, want 4", len(q.Stages))
	}
	if _, ok := q.Stages[0].(*ast.WhereStage); !ok {
		t.Fatalf("stage 0 = %T, want *WhereStage", q.Stages[0])
	}
	stats, ok := q.Stages[1].(*ast.StatsStage)
	if !ok {
		t.Fatalf("stage 1 = %T, want *StatsStage", q.Stages[1])
	}
	if len(stats.Aggs) != 1 || stats.Aggs[0].Func != "count" {
		t.Fatalf("stats aggs = %+v", stats.Aggs)
	}
	if len(stats.By) != 1 || stats.By[0] != "path" {
		t.Fatalf("stats by = %+v", stats.By)
	}
	sort, ok := q.Stages[2].(*ast.SortStage)
	if !ok || !sort.Keys[0].Desc {
		t.Fatalf("sort stage = %+v", q.Stages[2])
	}
	lim, ok := q.Stages[3].(*ast.LimitStage)
	if !ok || lim.N != 10 {
		t.Fatalf("limit stage = %+v", q.Stages[3])
	}
}

func TestParsePrecedence(t *testing.T) {
	// a or b and c  ==>  a or (b and c)
	q, err := Parse(`parse json | where a == 1 or b == 2 and c == 3`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	w := q.Stages[0].(*ast.WhereStage)
	top, ok := w.Predicate.(*ast.BinaryExpr)
	if !ok || top.Op != "or" {
		t.Fatalf("top expr = %+v, want or at root", w.Predicate)
	}
	right, ok := top.Right.(*ast.BinaryExpr)
	if !ok || right.Op != "and" {
		t.Fatalf("right expr = %+v, want and", top.Right)
	}
}

func TestParseStatsFunctions(t *testing.T) {
	q, err := Parse(`parse logfmt | stats sum(bytes) as total, avg(ms), count by host`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	st := q.Stages[0].(*ast.StatsStage)
	if len(st.Aggs) != 3 {
		t.Fatalf("agg count = %d, want 3", len(st.Aggs))
	}
	if st.Aggs[0].Func != "sum" || st.Aggs[0].Field != "bytes" || st.Aggs[0].Alias != "total" {
		t.Fatalf("agg0 = %+v", st.Aggs[0])
	}
	if st.Aggs[1].Func != "avg" || st.Aggs[1].Field != "ms" || st.Aggs[1].Alias != "avg_ms" {
		t.Fatalf("agg1 = %+v", st.Aggs[1])
	}
	if st.Aggs[2].Func != "count" || st.Aggs[2].Alias != "count" {
		t.Fatalf("agg2 = %+v", st.Aggs[2])
	}
}

func TestParseRegexSourceAndFields(t *testing.T) {
	q, err := Parse(`parse regex "(?P<ip>\S+) (?P<code>\d+)" | fields ip, code as status`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if q.Source.Format != ast.Regex || q.Source.Pattern == "" {
		t.Fatalf("regex source = %+v", q.Source)
	}
	f := q.Stages[0].(*ast.FieldsStage)
	if len(f.Fields) != 2 || f.Fields[1].Alias != "status" {
		t.Fatalf("fields = %+v", f.Fields)
	}
}

func TestParseErrors(t *testing.T) {
	bad := []string{
		``,
		`where status > 1`,              // missing source
		`parse logfmt | bogus x`,        // unknown stage
		`parse logfmt | limit abc`,      // non-numeric limit
		`parse logfmt | where (a == 1`,  // unbalanced paren
	}
	for _, src := range bad {
		if _, err := Parse(src); err == nil {
			t.Fatalf("expected error for %q", src)
		}
	}
}
