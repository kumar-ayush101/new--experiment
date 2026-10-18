// Package logql is the top-level facade: compile a query string and run it
// over a log source in one call.
package logql

import (
	"io"
	"strings"

	"github.com/kumar-ayush101/new--experiment/internal/compiler"
	"github.com/kumar-ayush101/new--experiment/internal/parser"
	"github.com/kumar-ayush101/new--experiment/internal/source"
	"github.com/kumar-ayush101/new--experiment/internal/vm"
)

// Query is a compiled, reusable query.
type Query struct {
	vmProg *vm.VM
}

// Compile parses and compiles a LogQL query string, returning a runnable Query
// bound to its input parser (determined by the `parse <format>` source).
func Compile(src string) (*Query, error) {
	q, err := parser.Parse(src)
	if err != nil {
		return nil, err
	}
	prog, err := compiler.Compile(q)
	if err != nil {
		return nil, err
	}
	lp, err := source.New(q.Source)
	if err != nil {
		return nil, err
	}
	return &Query{vmProg: vm.New(prog, lp)}, nil
}

// Run executes the query over r.
func (q *Query) Run(r io.Reader) (*vm.Result, error) {
	return q.vmProg.Run(r)
}

// RunString is a convenience that runs the query over an in-memory log string.
func (q *Query) RunString(logs string) (*vm.Result, error) {
	return q.Run(strings.NewReader(logs))
}
