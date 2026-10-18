package vm

import (
	"bufio"
	"io"
	"math"
	"sort"

	"github.com/kumar-ayush101/new--experiment/internal/bytecode"
	"github.com/kumar-ayush101/new--experiment/internal/record"
	"github.com/kumar-ayush101/new--experiment/internal/source"
)

// Result is the output of a query: an ordered column list plus rows.
type Result struct {
	Columns []string
	Rows    []record.Record
}

// VM executes a compiled program against a line source.
type VM struct {
	prog   *bytecode.Program
	parser source.LineParser
}

// New builds a VM for a program and input parser.
func New(prog *bytecode.Program, parser source.LineParser) *VM {
	return &VM{prog: prog, parser: parser}
}

// Run streams lines from r through the pipeline and returns the result.
//
// Streaming stages (where, fields, limit) are applied line-by-line to keep
// memory flat. Buffering stages (stats, sort) necessarily collect their input
// first; the executor splits the pipeline at the first buffering stage.
func (vm *VM) Run(r io.Reader) (*Result, error) {
	// Split stages into a streaming prefix (applied during the scan) and the
	// remaining stages (applied to the materialised rows).
	streamPrefix, rest := splitStreaming(vm.prog.Stages)

	stack := make([]record.Value, 0, 16)
	var rows []record.Record

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	limit, hasLimit := streamingLimit(streamPrefix)

	for sc.Scan() {
		rec, ok := vm.parser.Parse(sc.Text())
		if !ok {
			continue
		}
		rec, keep := applyStreaming(streamPrefix, rec, stack)
		if !keep {
			continue
		}
		rows = append(rows, rec)
		if hasLimit && len(rows) >= limit && len(rest) == 0 {
			break // early exit when only streaming stages + a limit remain
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	res := &Result{Rows: rows}
	for _, st := range rest {
		if err := vm.applyBuffered(res, st); err != nil {
			return nil, err
		}
	}
	if res.Columns == nil {
		res.Columns = inferColumns(res.Rows)
	}
	return res, nil
}

// splitStreaming returns the leading run of streaming stages and the rest.
func splitStreaming(stages []bytecode.Stage) (prefix, rest []bytecode.Stage) {
	i := 0
	for i < len(stages) {
		k := stages[i].Kind
		if k == bytecode.StageWhere || k == bytecode.StageFields || k == bytecode.StageLimit {
			i++
			continue
		}
		break
	}
	return stages[:i], stages[i:]
}

func streamingLimit(prefix []bytecode.Stage) (int, bool) {
	for _, s := range prefix {
		if s.Kind == bytecode.StageLimit {
			return s.N, true
		}
	}
	return 0, false
}

// applyStreaming runs where/fields/limit stages on a single record.
func applyStreaming(prefix []bytecode.Stage, rec record.Record, stack []record.Value) (record.Record, bool) {
	for _, s := range prefix {
		switch s.Kind {
		case bytecode.StageWhere:
			if !evalExpr(s.Predicate, rec, stack) {
				return nil, false
			}
		case bytecode.StageFields:
			rec = project(rec, s.Fields)
		case bytecode.StageLimit:
			// limit is handled by the caller's row count
		}
	}
	return rec, true
}

func project(rec record.Record, fields []bytecode.FieldProj) record.Record {
	out := make(record.Record, len(fields))
	for _, f := range fields {
		name := f.Name
		out[outName(f)] = rec[name]
		_ = name
	}
	return out
}

func outName(f bytecode.FieldProj) string {
	if f.Alias != "" {
		return f.Alias
	}
	return f.Name
}

// applyBuffered runs a stage that needs all rows materialised.
func (vm *VM) applyBuffered(res *Result, st bytecode.Stage) error {
	switch st.Kind {
	case bytecode.StageWhere:
		stack := make([]record.Value, 0, 16)
		filtered := res.Rows[:0]
		for _, rec := range res.Rows {
			if evalExpr(st.Predicate, rec, stack) {
				filtered = append(filtered, rec)
			}
		}
		res.Rows = filtered
	case bytecode.StageFields:
		cols := make([]string, len(st.Fields))
		for i, f := range st.Fields {
			cols[i] = outName(f)
		}
		for i, rec := range res.Rows {
			res.Rows[i] = project(rec, st.Fields)
		}
		res.Columns = cols
	case bytecode.StageLimit:
		if st.N < len(res.Rows) {
			res.Rows = res.Rows[:st.N]
		}
	case bytecode.StageSort:
		applySort(res.Rows, st.SortKeys)
	case bytecode.StageStats:
		runStats(res, st)
	}
	return nil
}

func applySort(rows []record.Record, keys []bytecode.SortKey) {
	sort.SliceStable(rows, func(i, j int) bool {
		for _, k := range keys {
			c := compare(rows[i][k.Field], rows[j][k.Field])
			if c == 0 {
				continue
			}
			if k.Desc {
				return c > 0
			}
			return c < 0
		}
		return false
	})
}

func inferColumns(rows []record.Record) []string {
	seen := map[string]bool{}
	var cols []string
	for _, rec := range rows {
		for k := range rec {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, k)
			}
		}
	}
	sort.Strings(cols)
	return cols
}

var _ = math.Inf
