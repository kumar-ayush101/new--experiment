// Package compiler lowers an ast.Query into a bytecode.Program. Pipeline
// stages become structured operators; filter predicates are compiled to a
// post-order stack-machine instruction stream so the VM can evaluate them per
// record without re-walking the AST.
package compiler

import (
	"fmt"

	"github.com/kumar-ayush101/new--experiment/internal/ast"
	"github.com/kumar-ayush101/new--experiment/internal/bytecode"
	"github.com/kumar-ayush101/new--experiment/internal/record"
)

// Compile lowers a query AST into an executable program.
func Compile(q *ast.Query) (*bytecode.Program, error) {
	prog := &bytecode.Program{}
	for _, st := range q.Stages {
		cs, err := compileStage(st)
		if err != nil {
			return nil, err
		}
		prog.Stages = append(prog.Stages, cs)
	}
	return prog, nil
}

func compileStage(st ast.Stage) (bytecode.Stage, error) {
	switch s := st.(type) {
	case *ast.WhereStage:
		ex, err := compileExpr(s.Predicate)
		if err != nil {
			return bytecode.Stage{}, err
		}
		return bytecode.Stage{Kind: bytecode.StageWhere, Predicate: ex}, nil
	case *ast.FieldsStage:
		fields := make([]bytecode.FieldProj, len(s.Fields))
		for i, f := range s.Fields {
			fields[i] = bytecode.FieldProj{Name: f.Name, Alias: f.Alias}
		}
		return bytecode.Stage{Kind: bytecode.StageFields, Fields: fields}, nil
	case *ast.StatsStage:
		aggs := make([]bytecode.Agg, len(s.Aggs))
		for i, a := range s.Aggs {
			fn, err := aggFunc(a.Func)
			if err != nil {
				return bytecode.Stage{}, err
			}
			aggs[i] = bytecode.Agg{Func: fn, Field: a.Field, Alias: a.Alias}
		}
		return bytecode.Stage{Kind: bytecode.StageStats, Aggs: aggs, By: s.By}, nil
	case *ast.SortStage:
		keys := make([]bytecode.SortKey, len(s.Keys))
		for i, k := range s.Keys {
			keys[i] = bytecode.SortKey{Field: k.Field, Desc: k.Desc}
		}
		return bytecode.Stage{Kind: bytecode.StageSort, SortKeys: keys}, nil
	case *ast.LimitStage:
		return bytecode.Stage{Kind: bytecode.StageLimit, N: s.N}, nil
	default:
		return bytecode.Stage{}, fmt.Errorf("compiler: unknown stage %T", st)
	}
}

func aggFunc(name string) (bytecode.AggFunc, error) {
	switch name {
	case "count":
		return bytecode.AggCount, nil
	case "sum":
		return bytecode.AggSum, nil
	case "avg":
		return bytecode.AggAvg, nil
	case "min":
		return bytecode.AggMin, nil
	case "max":
		return bytecode.AggMax, nil
	default:
		return 0, fmt.Errorf("compiler: unknown aggregate %q", name)
	}
}

// exprCompiler accumulates a predicate's instruction stream and pools.
type exprCompiler struct {
	ex *bytecode.Expr
}

func compileExpr(e ast.Expr) (*bytecode.Expr, error) {
	c := &exprCompiler{ex: &bytecode.Expr{}}
	if err := c.emit(e); err != nil {
		return nil, err
	}
	return c.ex, nil
}

func (c *exprCompiler) emit(e ast.Expr) error {
	switch n := e.(type) {
	case *ast.NumberLit:
		idx := c.addConst(record.Numv(n.Value))
		c.add(bytecode.OpConst, idx)
	case *ast.StringLit:
		idx := c.addConst(record.RawStr(n.Value))
		c.add(bytecode.OpConst, idx)
	case *ast.FieldExpr:
		idx := c.addField(n.Name)
		c.add(bytecode.OpLoadField, idx)
	case *ast.UnaryExpr:
		if n.Op != "not" {
			return fmt.Errorf("compiler: unknown unary op %q", n.Op)
		}
		if err := c.emit(n.Operand); err != nil {
			return err
		}
		c.add(bytecode.OpNot, 0)
	case *ast.BinaryExpr:
		return c.emitBinary(n)
	default:
		return fmt.Errorf("compiler: unknown expr %T", e)
	}
	return nil
}

func (c *exprCompiler) emitBinary(n *ast.BinaryExpr) error {
	// Regex ops need the right-hand side as a compiled pattern, not a value.
	if n.Op == "=~" || n.Op == "!~" {
		lit, ok := n.Right.(*ast.StringLit)
		if !ok {
			return fmt.Errorf("compiler: %s requires a string pattern on the right", n.Op)
		}
		if err := c.emit(n.Left); err != nil {
			return err
		}
		cr, err := bytecode.CompileRegex(lit.Value)
		if err != nil {
			return fmt.Errorf("compiler: invalid regex %q: %w", lit.Value, err)
		}
		idx := len(c.ex.Regexps)
		c.ex.Regexps = append(c.ex.Regexps, cr)
		if n.Op == "=~" {
			c.add(bytecode.OpMatch, idx)
		} else {
			c.add(bytecode.OpNMatch, idx)
		}
		return nil
	}

	if err := c.emit(n.Left); err != nil {
		return err
	}
	if err := c.emit(n.Right); err != nil {
		return err
	}
	switch n.Op {
	case "==":
		c.add(bytecode.OpEq, 0)
	case "!=":
		c.add(bytecode.OpNeq, 0)
	case "<":
		c.add(bytecode.OpLt, 0)
	case "<=":
		c.add(bytecode.OpLte, 0)
	case ">":
		c.add(bytecode.OpGt, 0)
	case ">=":
		c.add(bytecode.OpGte, 0)
	case "and":
		c.add(bytecode.OpAnd, 0)
	case "or":
		c.add(bytecode.OpOr, 0)
	default:
		return fmt.Errorf("compiler: unknown binary op %q", n.Op)
	}
	return nil
}

func (c *exprCompiler) add(op bytecode.Op, arg int) {
	c.ex.Code = append(c.ex.Code, bytecode.Instr{Op: op, Arg: arg})
}

func (c *exprCompiler) addConst(v record.Value) int {
	c.ex.Consts = append(c.ex.Consts, v)
	return len(c.ex.Consts) - 1
}

func (c *exprCompiler) addField(name string) int {
	for i, f := range c.ex.Fields {
		if f == name {
			return i
		}
	}
	c.ex.Fields = append(c.ex.Fields, name)
	return len(c.ex.Fields) - 1
}
