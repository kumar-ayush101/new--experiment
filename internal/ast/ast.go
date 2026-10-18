// Package ast defines the abstract syntax tree for LogQL. A query is a source
// stage followed by zero or more transform stages, mirroring the pipeline
// syntax `source | stage | stage | ...`.
package ast

// Query is the root node: a source and a pipeline of stages.
type Query struct {
	Source Source
	Stages []Stage
}

// ---- Sources ----

// Format identifies how raw log lines are parsed into records.
type Format int

const (
	Logfmt Format = iota // key=value key2="value two"
	JSON                 // one JSON object per line
	Regex                // named-capture regex
)

// Source describes how to turn raw lines into records.
type Source struct {
	Format  Format
	Pattern string // regex pattern when Format == Regex (named groups become fields)
}

// ---- Stages ----

// Stage is one transform in the pipeline.
type Stage interface{ stageNode() }

// WhereStage keeps only records for which Predicate is true.
type WhereStage struct {
	Predicate Expr
}

// FieldsStage projects a subset of fields (optionally renaming via As).
type FieldsStage struct {
	Fields []FieldRef
}

// FieldRef is a field name with an optional alias (name AS alias).
type FieldRef struct {
	Name  string
	Alias string
}

// StatsStage groups records by zero or more fields and computes aggregates.
type StatsStage struct {
	Aggs []Aggregation
	By   []string
}

// Aggregation is one aggregate function, e.g. count, sum(bytes) as total.
type Aggregation struct {
	Func  string // count, sum, avg, min, max
	Field string // empty for count
	Alias string // output column name
}

// SortStage orders records by keys.
type SortStage struct {
	Keys []SortKey
}

// SortKey is a field and direction.
type SortKey struct {
	Field string
	Desc  bool
}

// LimitStage keeps at most N records.
type LimitStage struct {
	N int
}

func (*WhereStage) stageNode()  {}
func (*FieldsStage) stageNode() {}
func (*StatsStage) stageNode()  {}
func (*SortStage) stageNode()   {}
func (*LimitStage) stageNode()  {}

// ---- Expressions (filter predicates) ----

// Expr is a boolean/value expression used in where predicates.
type Expr interface{ exprNode() }

// BinaryExpr is a comparison or logical operation.
type BinaryExpr struct {
	Op    string // ==, !=, <, <=, >, >=, =~, !~, and, or
	Left  Expr
	Right Expr
}

// UnaryExpr is a prefix operation (currently only `not`).
type UnaryExpr struct {
	Op      string
	Operand Expr
}

// FieldExpr references a record field by name.
type FieldExpr struct {
	Name string
}

// StringLit is a string literal.
type StringLit struct {
	Value string
}

// NumberLit is a numeric literal.
type NumberLit struct {
	Value float64
}

func (*BinaryExpr) exprNode() {}
func (*UnaryExpr) exprNode()  {}
func (*FieldExpr) exprNode()  {}
func (*StringLit) exprNode()  {}
func (*NumberLit) exprNode()  {}
