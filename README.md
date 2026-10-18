# LogQL — a query language for log files

LogQL is a small, purpose-built programming language for querying log files. It
replaces tangled `grep | awk | sort | uniq -c | sort -rn` pipelines with one
readable query:

```
parse logfmt | where status >= 500 | stats count by path | sort count desc | limit 10
```

It is a complete language implementation written from scratch in Go, with no
external dependencies — a lexer, a parser, a bytecode compiler and a stack
virtual machine that streams records through a pipeline of stages.

> Work in progress. See the roadmap in this README.

## Why a language instead of a library

General tools (`grep`, `awk`, `jq`) each do one thing; chaining them is verbose,
error-prone and hard to read. SQL is powerful but assumes tabular, pre-loaded
data. LogQL is designed for the specific shape of log data — semi-structured
lines (logfmt, JSON, or regex-extracted) streamed from a file — and makes the
common operations (filter, group, aggregate, sort, top-N) first-class.

## License

MIT — see [LICENSE](./LICENSE).
