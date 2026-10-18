package vm

import (
	"math"
	"sort"
	"strings"

	"github.com/kumar-ayush101/new--experiment/internal/bytecode"
	"github.com/kumar-ayush101/new--experiment/internal/record"
)

// aggState accumulates one aggregate's running value within a group.
type aggState struct {
	count int64
	sum   float64
	min   float64
	max   float64
	seen  bool
}

func (a *aggState) add(v record.Value, hasVal bool) {
	a.count++
	if !hasVal {
		return
	}
	f, ok := v.AsNumber()
	if !ok {
		return
	}
	a.sum += f
	if !a.seen {
		a.min, a.max = f, f
		a.seen = true
	} else {
		a.min = math.Min(a.min, f)
		a.max = math.Max(a.max, f)
	}
}

func (a *aggState) result(fn bytecode.AggFunc) record.Value {
	switch fn {
	case bytecode.AggCount:
		return record.Numv(float64(a.count))
	case bytecode.AggSum:
		return record.Numv(a.sum)
	case bytecode.AggAvg:
		if a.count == 0 {
			return record.Numv(0)
		}
		return record.Numv(a.sum / float64(a.count))
	case bytecode.AggMin:
		if !a.seen {
			return record.Nullv()
		}
		return record.Numv(a.min)
	case bytecode.AggMax:
		if !a.seen {
			return record.Nullv()
		}
		return record.Numv(a.max)
	default:
		return record.Nullv()
	}
}

// group holds the by-field values and one aggState per aggregation.
type group struct {
	keyVals []record.Value
	states  []aggState
}

// runStats performs group-by aggregation over res.Rows, replacing them with
// one output row per group. Output columns are the by-fields followed by the
// aggregate aliases.
func runStats(res *Result, st bytecode.Stage) {
	groups := make(map[string]*group)
	var order []string // preserve first-seen group order for stable output

	var sb strings.Builder
	for _, rec := range res.Rows {
		sb.Reset()
		for i, f := range st.By {
			if i > 0 {
				sb.WriteByte(0)
			}
			sb.WriteString(rec[f].String())
		}
		key := sb.String()
		g := groups[key]
		if g == nil {
			g = &group{
				keyVals: make([]record.Value, len(st.By)),
				states:  make([]aggState, len(st.Aggs)),
			}
			for i, f := range st.By {
				g.keyVals[i] = rec[f]
			}
			groups[key] = g
			order = append(order, key)
		}
		for i, a := range st.Aggs {
			if a.Func == bytecode.AggCount {
				g.states[i].add(record.Nullv(), false)
				continue
			}
			v, ok := rec[a.Field]
			g.states[i].add(v, ok)
		}
	}

	// Build output rows.
	cols := make([]string, 0, len(st.By)+len(st.Aggs))
	cols = append(cols, st.By...)
	for _, a := range st.Aggs {
		cols = append(cols, a.Alias)
	}

	rows := make([]record.Record, 0, len(order))
	for _, key := range order {
		g := groups[key]
		out := make(record.Record, len(cols))
		for i, f := range st.By {
			out[f] = g.keyVals[i]
		}
		for i, a := range st.Aggs {
			out[a.Alias] = g.states[i].result(a.Func)
		}
		rows = append(rows, out)
	}

	// When there are no group-by fields, aggregates over an empty input still
	// produce a single zero row (e.g. count of nothing = 0).
	if len(st.By) == 0 && len(rows) == 0 {
		out := make(record.Record, len(st.Aggs))
		for _, a := range st.Aggs {
			var empty aggState
			out[a.Alias] = empty.result(a.Func)
		}
		rows = append(rows, out)
	}

	res.Rows = rows
	res.Columns = cols
	_ = sort.Strings
}
