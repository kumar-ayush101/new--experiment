# LogQL — a query language for log files

[![CI](https://github.com/kumar-ayush101/new--experiment/actions/workflows/ci.yml/badge.svg)](https://github.com/kumar-ayush101/new--experiment/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![deps](https://img.shields.io/badge/dependencies-none-success)
![License](https://img.shields.io/badge/License-MIT-green)

LogQL is a small, purpose-built programming language for querying log files. It
turns tangled shell pipelines like

```bash
grep ' 5[0-9][0-9] ' access.log | awk '{print $7}' | sort | uniq -c | sort -rn | head
```

into one readable query:

```
parse logfmt | where status >= 500 | stats count by path | sort count desc | limit 10
```

It is a **complete language implementation written from scratch in Go with zero
external dependencies** — a hand-written lexer, a recursive-descent parser, a
bytecode compiler, and a stack virtual machine that streams records through a
pipeline of operators. It processes roughly **1 million log lines per second**
end to end (see Benchmarks).

---

## Why a language instead of `grep | awk | jq`

General tools each do one thing; chaining them is verbose, positional, and
breaks the moment a log format changes. SQL is powerful but assumes tabular,
pre-loaded data. LogQL is designed for the actual shape of logs — semi-structured
lines (logfmt, JSON, or regex-extracted) streamed from a file — and makes the
operations you actually do (filter, group, aggregate, sort, top-N) first-class
and composable with a single `|` pipeline.

---

## Install / run

```bash
go build -o logql ./cmd/logql

# query a file
./logql -q 'parse logfmt | where status >= 500 | stats count by path | sort count desc | limit 10' app.log

# or read from stdin, output JSON
cat access.log | ./logql -q 'parse json | where level == "error" | limit 20' -format json
```

---

## Language

A query is a **source** followed by a pipeline of **stages**, separated by `|`:

```
parse <format> | <stage> | <stage> | ...
```

### Sources

| Source | Parses |
| ------ | ------ |
| `parse logfmt` | `key=value key2="value with spaces"` lines |
| `parse json` | one JSON object per line |
| `parse regex "<pattern>"` | named captures `(?P<name>...)` become fields |

Numeric-looking fields are automatically typed as numbers, so `status >= 500`
compares numerically while `path == "/api"` compares as text.

### Stages

| Stage | Meaning | Aliases |
| ----- | ------- | ------- |
| `where <predicate>` | keep matching records | `filter` |
| `fields a, b as c` | project / rename columns | `project` |
| `stats <agg>[, <agg>] [by <f>...]` | group and aggregate | `aggregate` |
| `sort <f> [asc\|desc][, ...]` | order rows | `order` |
| `limit <N>` | keep first N rows | `head` |

Aggregates: `count`, `sum(field)`, `avg(field)`, `min(field)`, `max(field)`,
each with an optional `as alias`.

### Predicates

Comparisons `== != < <= > >=`, regex `=~` / `!~`, boolean `and` / `or` / `not`,
and parentheses. `not` binds looser than comparisons, so `not status == 200`
means `not (status == 200)`.

### Grammar (EBNF)

```
query      = "parse" format { "|" stage } ;
format     = "logfmt" | "json" | "regex" string ;
stage      = where | fields | stats | sort | limit ;
where      = ("where"|"filter") expr ;
fields     = ("fields"|"project") fieldref { "," fieldref } ;
fieldref   = ident [ "as" ident ] ;
stats      = ("stats"|"aggregate") agg { "," agg } [ "by" ident { "," ident } ] ;
agg        = ident [ "(" ident ")" ] [ "as" ident ] ;
sort       = ("sort"|"order") sortkey { "," sortkey } ;
sortkey    = ident [ "asc" | "desc" ] ;
limit      = ("limit"|"head") number ;
expr       = orExpr ;
orExpr     = andExpr { "or" andExpr } ;
andExpr    = cmpExpr { "and" cmpExpr } ;
cmpExpr    = primary [ cmpOp primary ] | "not" expr ;
primary    = ident | string | number | "(" expr ")" ;
```

---

## Examples

```
# Top 10 endpoints returning server errors
parse logfmt | where status >= 500 | stats count by path | sort count desc | limit 10

# Average and total latency per status code
parse logfmt | stats avg(ms), sum(ms) as total_ms by status

# Errors on the API, excluding health checks
parse json | where level == "error" and path =~ "/api" and not path =~ "/health"

# Parse Apache/nginx access logs with a regex, count by client IP
parse regex "(?P<ip>\S+) \S+ \S+ \[[^\]]+\] \"\S+ (?P<path>\S+)[^\"]*\" (?P<status>\d+)" | stats count by ip | sort count desc | limit 20
```

---

## Architecture

A classic language pipeline, each stage a separate package:

```
source text
   │  internal/lexer      → tokens          (hand-written scanner)
   ▼
tokens
   │  internal/parser     → AST             (recursive descent + precedence climbing)
   ▼
ast.Query
   │  internal/compiler   → bytecode.Program
   ▼                        - pipeline stages become structured operators
bytecode                    - filter predicates become stack-machine code
   │  internal/vm         → Result
   ▼                        - streams lines through operators
rows                        - predicates run on a reusable value stack
```

Design notes:

- **Predicates are compiled, not re-walked.** Each `where` expression lowers to
  a flat instruction stream (`OpLoadField`, `OpConst`, `OpGte`, `OpAnd`, ...),
  evaluated on a small reusable stack per record. Regexes are compiled once at
  compile time, not per line.
- **Streaming where possible.** `where`, `fields` and `limit` run during the
  file scan with a flat memory profile and an early exit when a `limit` is
  satisfied. Only `stats` and `sort` buffer, because they must see all rows.
- **Typed values.** A field is `string`, `number`, or `bool`; numeric text is
  promoted on parse so numeric and lexical comparisons both behave correctly.

### Project layout

```text
.
├── cmd/
│   ├── logql/        # CLI: run a query over a file or stdin
│   └── logqlbench/   # throughput benchmark harness
├── internal/
│   ├── token/        # token definitions
│   ├── lexer/        # source text → tokens
│   ├── ast/          # abstract syntax tree
│   ├── parser/       # tokens → AST
│   ├── record/       # typed value + record model
│   ├── source/       # logfmt / json / regex line parsers
│   ├── bytecode/     # compiled program + predicate opcodes
│   ├── compiler/     # AST → bytecode
│   └── vm/           # streaming executor + predicate stack machine + aggregation
├── logql.go          # top-level facade: Compile + Run
└── testdata/         # sample logs
```

---

## Benchmarks

`go run ./cmd/logqlbench -n 1000000` generates 1,000,000 logfmt lines (~63 MB)
and times representative queries end to end (parse + filter + aggregate).
Representative run (Go 1.25, single machine — numbers are relative):

| query | records/sec |
|---|---|
| filter only | ~1.0M |
| filter + regex | ~1.0M |
| group + count | ~1.0M |
| group + sum/avg | ~0.8M |
| top-N pipeline | ~1.0M |

Throughput is roughly **one million log records per second**, dominated by line
parsing — the compiled-predicate VM keeps filtering and aggregation cheap.

---

## Testing

```bash
go test -race ./...        # lexer, parser, source, vm, and end-to-end tests
go run ./cmd/logqlbench     # throughput numbers
```

The suite covers tokenization, parsing (including operator precedence and
errors), all three input formats, the predicate stack machine, group-by
aggregation, sort/limit, projection, and full `parse → compile → run` queries
over sample data.

---

## Roadmap

| Stage | Component | State |
|---|---|---|
| 1 | Lexer | **Done** |
| 2 | Parser + AST | **Done** |
| 3 | Record model + logfmt/json/regex sources | **Done** |
| 4 | Bytecode compiler | **Done** |
| 5 | Streaming VM + predicate stack machine | **Done** |
| 6 | Aggregation (count/sum/avg/min/max, group by) | **Done** |
| 7 | CLI (table/json) + benchmark harness | **Done** |
| — | Time-window aggregation (`by bucket(ts, 1m)`) | Planned |
| — | More functions (percentiles, distinct count) | Planned |
| — | Compile predicates to Go closures for extra speed | Stretch |

## License

MIT — see [LICENSE](./LICENSE).
