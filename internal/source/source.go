// Package source turns raw log lines into records according to the chosen
// input format. Each parser implements LineParser so the VM can stream a file
// one line at a time without loading it all into memory.
package source

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kumar-ayush101/new--experiment/internal/ast"
	"github.com/kumar-ayush101/new--experiment/internal/record"
)

// LineParser converts one raw line into a Record. ok is false when the line
// should be skipped (blank or unparseable), without being a fatal error.
type LineParser interface {
	Parse(line string) (rec record.Record, ok bool)
}

// New returns the parser for a source definition.
func New(s ast.Source) (LineParser, error) {
	switch s.Format {
	case ast.Logfmt:
		return &logfmtParser{}, nil
	case ast.JSON:
		return &jsonParser{}, nil
	case ast.Regex:
		re, err := regexp.Compile(s.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return &regexParser{re: re, names: re.SubexpNames()}, nil
	default:
		return nil, fmt.Errorf("unknown source format")
	}
}

// ---- logfmt: key=value key2="value with spaces" ----

type logfmtParser struct{}

func (p *logfmtParser) Parse(line string) (record.Record, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, false
	}
	rec := make(record.Record)
	i := 0
	n := len(line)
	for i < n {
		// skip spaces
		for i < n && line[i] == ' ' {
			i++
		}
		if i >= n {
			break
		}
		// key
		keyStart := i
		for i < n && line[i] != '=' && line[i] != ' ' {
			i++
		}
		key := line[keyStart:i]
		if i >= n || line[i] != '=' {
			// bare token without a value; store as present=true
			if key != "" {
				rec[key] = record.Boolv(true)
			}
			continue
		}
		i++ // consume '='
		// value: quoted or bare
		if i < n && (line[i] == '"' || line[i] == '\'') {
			quote := line[i]
			i++
			var b strings.Builder
			for i < n && line[i] != quote {
				if line[i] == '\\' && i+1 < n {
					i++
				}
				b.WriteByte(line[i])
				i++
			}
			if i < n {
				i++ // closing quote
			}
			rec[key] = record.Str(b.String())
		} else {
			valStart := i
			for i < n && line[i] != ' ' {
				i++
			}
			rec[key] = record.Str(line[valStart:i])
		}
	}
	if len(rec) == 0 {
		return nil, false
	}
	return rec, true
}

// ---- JSON lines: one object per line ----

type jsonParser struct{}

func (p *jsonParser) Parse(line string) (record.Record, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, false
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, false
	}
	rec := make(record.Record, len(raw))
	for k, v := range raw {
		rec[k] = fromAny(v)
	}
	return rec, true
}

func fromAny(v any) record.Value {
	switch t := v.(type) {
	case string:
		return record.RawStr(t)
	case float64:
		return record.Numv(t)
	case bool:
		return record.Boolv(t)
	case nil:
		return record.Nullv()
	default:
		// nested objects/arrays: render as their JSON text
		b, _ := json.Marshal(t)
		return record.RawStr(string(b))
	}
}

// ---- regex with named captures ----

type regexParser struct {
	re    *regexp.Regexp
	names []string
}

func (p *regexParser) Parse(line string) (record.Record, bool) {
	m := p.re.FindStringSubmatch(line)
	if m == nil {
		return nil, false
	}
	rec := make(record.Record)
	for i, name := range p.names {
		if i == 0 || name == "" {
			continue
		}
		rec[name] = record.Str(m[i])
	}
	if len(rec) == 0 {
		return nil, false
	}
	return rec, true
}

// quoteIfNeeded is used by output formatting elsewhere; kept here to centralise
// logfmt escaping rules.
func quoteIfNeeded(s string) string {
	if strings.ContainsAny(s, " \"") {
		return strconv.Quote(s)
	}
	return s
}

var _ = quoteIfNeeded
