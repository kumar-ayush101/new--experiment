// Package vm executes a compiled bytecode.Program over a stream of records.
// This file implements the predicate evaluator: a tiny stack machine that runs
// a compiled filter expression against one record and returns a boolean.
package vm

import (
	"github.com/kumar-ayush101/new--experiment/internal/bytecode"
	"github.com/kumar-ayush101/new--experiment/internal/record"
)

// evalExpr runs a compiled predicate against rec, returning its truthiness.
// The stack is passed in so the caller can reuse one allocation across the
// whole record stream (hot path).
func evalExpr(ex *bytecode.Expr, rec record.Record, stack []record.Value) bool {
	stack = stack[:0]
	push := func(v record.Value) { stack = append(stack, v) }
	pop := func() record.Value {
		n := len(stack) - 1
		v := stack[n]
		stack = stack[:n]
		return v
	}

	for _, in := range ex.Code {
		switch in.Op {
		case bytecode.OpConst:
			push(ex.Consts[in.Arg])
		case bytecode.OpLoadField:
			name := ex.Fields[in.Arg]
			if v, ok := rec[name]; ok {
				push(v)
			} else {
				push(record.Nullv())
			}
		case bytecode.OpEq:
			b := pop()
			a := pop()
			push(record.Boolv(compare(a, b) == 0 && comparable(a, b)))
		case bytecode.OpNeq:
			b := pop()
			a := pop()
			push(record.Boolv(!(compare(a, b) == 0 && comparable(a, b))))
		case bytecode.OpLt:
			b := pop()
			a := pop()
			push(record.Boolv(compare(a, b) < 0))
		case bytecode.OpLte:
			b := pop()
			a := pop()
			push(record.Boolv(compare(a, b) <= 0))
		case bytecode.OpGt:
			b := pop()
			a := pop()
			push(record.Boolv(compare(a, b) > 0))
		case bytecode.OpGte:
			b := pop()
			a := pop()
			push(record.Boolv(compare(a, b) >= 0))
		case bytecode.OpMatch:
			a := pop()
			re := ex.Regexps[in.Arg].RE
			push(record.Boolv(re.MatchString(a.String())))
		case bytecode.OpNMatch:
			a := pop()
			re := ex.Regexps[in.Arg].RE
			push(record.Boolv(!re.MatchString(a.String())))
		case bytecode.OpAnd:
			b := pop()
			a := pop()
			push(record.Boolv(a.Truthy() && b.Truthy()))
		case bytecode.OpOr:
			b := pop()
			a := pop()
			push(record.Boolv(a.Truthy() || b.Truthy()))
		case bytecode.OpNot:
			a := pop()
			push(record.Boolv(!a.Truthy()))
		}
	}
	if len(stack) == 0 {
		return false
	}
	return stack[len(stack)-1].Truthy()
}

// comparable reports whether a and b can be meaningfully compared for equality
// (same broad category after numeric promotion).
func comparable(a, b record.Value) bool {
	_, an := a.AsNumber()
	_, bn := b.AsNumber()
	if an && bn {
		return true
	}
	return a.Kind != record.Null && b.Kind != record.Null
}

// compare returns -1, 0, or 1. Numeric when both sides are numeric, else
// lexical on their string forms. A null compares as "less than" anything.
func compare(a, b record.Value) int {
	af, an := a.AsNumber()
	bf, bn := b.AsNumber()
	if an && bn {
		switch {
		case af < bf:
			return -1
		case af > bf:
			return 1
		default:
			return 0
		}
	}
	as, bs := a.String(), b.String()
	switch {
	case as < bs:
		return -1
	case as > bs:
		return 1
	default:
		return 0
	}
}
