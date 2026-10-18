// Command logql runs a LogQL query against log data read from a file or stdin
// and prints the result as an aligned table (default) or JSON.
//
// Examples:
//
//	logql -q 'parse logfmt | where status >= 500 | stats count by path | sort count desc' app.log
//	cat app.log | logql -q 'parse json | where level == "error" | limit 20'
//	logql -q '...' -format json access.log
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kumar-ayush101/new--experiment"
	"github.com/kumar-ayush101/new--experiment/internal/vm"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "logql: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	var query, format, file string
	format = "table"

	// Minimal flag parsing so -q can hold spaces without quoting surprises.
	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "-q" || a == "--query":
			i++
			if i >= len(args) {
				return fmt.Errorf("-q requires a query string")
			}
			query = args[i]
		case a == "-format" || a == "--format":
			i++
			if i >= len(args) {
				return fmt.Errorf("-format requires a value (table|json)")
			}
			format = args[i]
		case a == "-h" || a == "--help":
			printUsage()
			return nil
		case strings.HasPrefix(a, "-"):
			return fmt.Errorf("unknown flag %q", a)
		default:
			file = a // positional: input file
		}
		i++
	}

	if query == "" {
		printUsage()
		return fmt.Errorf("missing -q <query>")
	}

	q, err := logql.Compile(query)
	if err != nil {
		return err
	}

	var in io.Reader = os.Stdin
	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}

	res, err := q.Run(in)
	if err != nil {
		return err
	}

	switch format {
	case "json":
		return printJSON(os.Stdout, res)
	case "table":
		return printTable(os.Stdout, res)
	default:
		return fmt.Errorf("unknown format %q (want table or json)", format)
	}
}

func printTable(w io.Writer, res *vm.Result) error {
	if len(res.Rows) == 0 {
		fmt.Fprintln(w, "(no rows)")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(res.Columns, "\t"))
	// underline header
	ul := make([]string, len(res.Columns))
	for i, c := range res.Columns {
		ul[i] = strings.Repeat("-", len(c))
	}
	fmt.Fprintln(tw, strings.Join(ul, "\t"))
	for _, rec := range res.Rows {
		cells := make([]string, len(res.Columns))
		for i, c := range res.Columns {
			cells[i] = rec[c].String()
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	return tw.Flush()
}

func printJSON(w io.Writer, res *vm.Result) error {
	out := make([]map[string]string, 0, len(res.Rows))
	for _, rec := range res.Rows {
		m := make(map[string]string, len(res.Columns))
		for _, c := range res.Columns {
			m[c] = rec[c].String()
		}
		out = append(out, m)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printUsage() {
	fmt.Fprint(os.Stderr, `LogQL — a query language for log files

Usage:
  logql -q '<query>' [-format table|json] [file]

If no file is given, logql reads from stdin.

Query shape:
  parse <logfmt|json|regex "pat"> | <stage> | <stage> ...

Stages:
  where <predicate>              keep matching records (alias: filter)
  fields a, b as c               project/rename columns (alias: project)
  stats <agg>[, <agg>] [by f]    aggregate: count, sum(x), avg(x), min(x), max(x)
  sort f [asc|desc][, ...]       order rows (alias: order)
  limit N                        keep first N rows (alias: head)

Examples:
  logql -q 'parse logfmt | where status >= 500 | stats count by path | sort count desc | limit 10' app.log
  cat access.log | logql -q 'parse regex "(?P<ip>\S+).*\" (?P<status>\d+)" | stats count by ip | sort count desc'
`)
}
