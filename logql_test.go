package logql

import (
	"testing"

	"github.com/kumar-ayush101/new--experiment/internal/record"
)

const sampleLogs = `level=info status=200 path=/home bytes=120
level=error status=500 path=/api bytes=0
level=info status=200 path=/api bytes=340
level=warn status=404 path=/missing bytes=0
level=error status=503 path=/api bytes=0
level=info status=200 path=/home bytes=90`

func run(t *testing.T, query string) *struct {
	Columns []string
	Rows    []record.Record
} {
	t.Helper()
	q, err := Compile(query)
	if err != nil {
		t.Fatalf("Compile(%q): %v", query, err)
	}
	res, err := q.RunString(sampleLogs)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return &struct {
		Columns []string
		Rows    []record.Record
	}{res.Columns, res.Rows}
}

func TestWhereNumericFilter(t *testing.T) {
	res := run(t, `parse logfmt | where status >= 500`)
	if len(res.Rows) != 2 {
		t.Fatalf("rows = %d, want 2 (two 5xx)", len(res.Rows))
	}
	for _, r := range res.Rows {
		if n, _ := r["status"].AsNumber(); n < 500 {
			t.Fatalf("got status %v, want >= 500", n)
		}
	}
}

func TestWhereRegexAndLogic(t *testing.T) {
	res := run(t, `parse logfmt | where path =~ "/api" and status >= 500`)
	if len(res.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(res.Rows))
	}
}

func TestStatsCountBy(t *testing.T) {
	res := run(t, `parse logfmt | stats count by path`)
	// paths: /home x2, /api x3, /missing x1 -> 3 groups
	if len(res.Rows) != 3 {
		t.Fatalf("groups = %d, want 3", len(res.Rows))
	}
	counts := map[string]float64{}
	for _, r := range res.Rows {
		n, _ := r["count"].AsNumber()
		counts[r["path"].String()] = n
	}
	if counts["/api"] != 3 || counts["/home"] != 2 || counts["/missing"] != 1 {
		t.Fatalf("counts = %+v", counts)
	}
}

func TestStatsSumAvg(t *testing.T) {
	res := run(t, `parse logfmt | stats sum(bytes) as total, avg(bytes) by level`)
	totals := map[string]float64{}
	for _, r := range res.Rows {
		n, _ := r["total"].AsNumber()
		totals[r["level"].String()] = n
	}
	// info bytes: 120+340+90 = 550
	if totals["info"] != 550 {
		t.Fatalf("info total = %v, want 550", totals["info"])
	}
}

func TestSortAndLimitTopN(t *testing.T) {
	res := run(t, `parse logfmt | stats count by path | sort count desc | limit 1`)
	if len(res.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(res.Rows))
	}
	if res.Rows[0]["path"].String() != "/api" {
		t.Fatalf("top path = %q, want /api", res.Rows[0]["path"].String())
	}
}

func TestFieldsProjection(t *testing.T) {
	res := run(t, `parse logfmt | where status == 404 | fields path, status as code`)
	if len(res.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(res.Rows))
	}
	r := res.Rows[0]
	if _, ok := r["code"]; !ok {
		t.Fatalf("expected aliased column 'code', got %+v", r)
	}
	if _, ok := r["level"]; ok {
		t.Fatalf("projection should have dropped 'level'")
	}
}

func TestJSONSource(t *testing.T) {
	logs := `{"status":200,"path":"/a"}
{"status":500,"path":"/b"}
{"status":500,"path":"/b"}`
	q, err := Compile(`parse json | where status >= 500 | stats count by path`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	res, err := q.RunString(logs)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Rows) != 1 || res.Rows[0]["path"].String() != "/b" {
		t.Fatalf("json result = %+v", res.Rows)
	}
	if n, _ := res.Rows[0]["count"].AsNumber(); n != 2 {
		t.Fatalf("count = %v, want 2", n)
	}
}

func TestNotOperator(t *testing.T) {
	res := run(t, `parse logfmt | where not status == 200`)
	// non-200: 500, 404, 503 -> 3 rows
	if len(res.Rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(res.Rows))
	}
}
