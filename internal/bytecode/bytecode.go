// Package bytecode defines the compiled form of a LogQL query. Filter
// predicates compile to a stack-machine instruction stream (Expr); the overall
// query compiles to an ordered list of pipeline operators (Program), each a
// structured op that the VM executes as a streaming or buffering stage.
package bytecode

import "github.com/kumar-ayush101/new--experiment/internal/record"

// Op is a predicate-expression opcode.
type Op uint8

const (
	OpConst     Op = iota // push constant Const[arg]
	OpLoadField           // push value of field Fields[arg]
	OpEq
	OpNeq
	OpLt
	OpLte
	OpGt
	OpGte
	OpMatch  // regex match, pattern index in Regexps[arg]
	OpNMatch // regex not-match
	OpAnd
	OpOr
	OpNot
)

// Instr is one predicate instruction: an opcode plus an integer argument whose
// meaning depends on the opcode (constant index, field index, regex index).
type Instr struct {
	Op  Op
	Arg int
}

// Expr is a compiled predicate: a flat instruction stream plus its pools. It
// is evaluated against a record on a small value stack, leaving one boolean
// result on top.
type Expr struct {
	Code    []Instr
	Consts  []record.Value
	Fields  []string         // field names referenced by OpLoadField
	Regexps []*CompiledRegex // regex programs referenced by OpMatch/OpNMatch
}

// StageKind identifies a pipeline operator.
type StageKind uint8

const (
	StageWhere StageKind = iota
	StageFields
	StageStats
	StageSort
	StageLimit
)

// Stage is one compiled pipeline operator.
type Stage struct {
	Kind StageKind

	// StageWhere
	Predicate *Expr

	// StageFields
	Fields []FieldProj

	// StageStats
	Aggs []Agg
	By   []string

	// StageSort
	SortKeys []SortKey

	// StageLimit
	N int
}

// FieldProj is a projected field with an optional output name.
type FieldProj struct {
	Name  string
	Alias string
}

// AggFunc is an aggregate function id.
type AggFunc uint8

const (
	AggCount AggFunc = iota
	AggSum
	AggAvg
	AggMin
	AggMax
)

// Agg is a compiled aggregation.
type Agg struct {
	Func  AggFunc
	Field string
	Alias string
}

// SortKey is a compiled sort key.
type SortKey struct {
	Field string
	Desc  bool
}

// Program is a fully compiled query: the ordered pipeline operators.
type Program struct {
	Stages []Stage
}
