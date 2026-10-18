// Package parser builds an ast.Query from LogQL source using recursive
// descent, with precedence-climbing for filter expressions.
package parser

import (
	"fmt"
	"strconv"

	"github.com/kumar-ayush101/new--experiment/internal/ast"
	"github.com/kumar-ayush101/new--experiment/internal/lexer"
	"github.com/kumar-ayush101/new--experiment/internal/token"
)

// Parser holds the token stream and lookahead.
type Parser struct {
	lex  *lexer.Lexer
	cur  token.Token
	peek token.Token
}

// New builds a parser over src.
func New(src string) *Parser {
	p := &Parser{lex: lexer.New(src)}
	p.next()
	p.next()
	return p
}

func (p *Parser) next() {
	p.cur = p.peek
	p.peek = p.lex.Next()
}

func (p *Parser) errf(format string, args ...any) error {
	return fmt.Errorf("parse error at offset %d: %s", p.cur.Pos, fmt.Sprintf(format, args...))
}

// Parse parses a complete query.
func Parse(src string) (*ast.Query, error) {
	return New(src).ParseQuery()
}

// ParseQuery parses `source | stage | ...`.
func (p *Parser) ParseQuery() (*ast.Query, error) {
	src, err := p.parseSource()
	if err != nil {
		return nil, err
	}
	q := &ast.Query{Source: src}
	for p.cur.Type == token.PIPE {
		p.next() // consume |
		st, err := p.parseStage()
		if err != nil {
			return nil, err
		}
		q.Stages = append(q.Stages, st)
	}
	if p.cur.Type != token.EOF {
		return nil, p.errf("unexpected token %q after pipeline", p.cur.Literal)
	}
	return q, nil
}

// parseSource parses `parse logfmt|json|regex "pattern"`.
func (p *Parser) parseSource() (ast.Source, error) {
	if p.cur.Type != token.IDENT || p.cur.Literal != "parse" {
		return ast.Source{}, p.errf("query must start with `parse <format>`, got %q", p.cur.Literal)
	}
	p.next()
	if p.cur.Type != token.IDENT {
		return ast.Source{}, p.errf("expected a format after `parse`")
	}
	switch p.cur.Literal {
	case "logfmt":
		p.next()
		return ast.Source{Format: ast.Logfmt}, nil
	case "json":
		p.next()
		return ast.Source{Format: ast.JSON}, nil
	case "regex":
		p.next()
		if p.cur.Type != token.STRING {
			return ast.Source{}, p.errf("`parse regex` requires a quoted pattern")
		}
		pat := p.cur.Literal
		p.next()
		return ast.Source{Format: ast.Regex, Pattern: pat}, nil
	default:
		return ast.Source{}, p.errf("unknown format %q (want logfmt, json or regex)", p.cur.Literal)
	}
}

func (p *Parser) parseStage() (ast.Stage, error) {
	if p.cur.Type != token.IDENT {
		return nil, p.errf("expected a stage name, got %q", p.cur.Literal)
	}
	switch p.cur.Literal {
	case "where", "filter":
		return p.parseWhere()
	case "fields", "project":
		return p.parseFields()
	case "stats", "aggregate":
		return p.parseStats()
	case "sort", "order":
		return p.parseSort()
	case "limit", "head":
		return p.parseLimit()
	default:
		return nil, p.errf("unknown stage %q", p.cur.Literal)
	}
}

func (p *Parser) parseWhere() (ast.Stage, error) {
	p.next() // consume stage keyword
	expr, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	return &ast.WhereStage{Predicate: expr}, nil
}

func (p *Parser) parseFields() (ast.Stage, error) {
	p.next()
	var fields []ast.FieldRef
	for {
		if p.cur.Type != token.IDENT {
			return nil, p.errf("expected a field name")
		}
		fr := ast.FieldRef{Name: p.cur.Literal}
		p.next()
		if p.cur.Type == token.AS {
			p.next()
			if p.cur.Type != token.IDENT {
				return nil, p.errf("expected an alias after `as`")
			}
			fr.Alias = p.cur.Literal
			p.next()
		}
		fields = append(fields, fr)
		if p.cur.Type != token.COMMA {
			break
		}
		p.next()
	}
	return &ast.FieldsStage{Fields: fields}, nil
}

func (p *Parser) parseStats() (ast.Stage, error) {
	p.next()
	st := &ast.StatsStage{}
	for {
		agg, err := p.parseAggregation()
		if err != nil {
			return nil, err
		}
		st.Aggs = append(st.Aggs, agg)
		if p.cur.Type != token.COMMA {
			break
		}
		p.next()
	}
	if p.cur.Type == token.BY {
		p.next()
		for {
			if p.cur.Type != token.IDENT {
				return nil, p.errf("expected a field name after `by`")
			}
			st.By = append(st.By, p.cur.Literal)
			p.next()
			if p.cur.Type != token.COMMA {
				break
			}
			p.next()
		}
	}
	return st, nil
}

func (p *Parser) parseAggregation() (ast.Aggregation, error) {
	if p.cur.Type != token.IDENT {
		return ast.Aggregation{}, p.errf("expected an aggregate function")
	}
	fn := p.cur.Literal
	agg := ast.Aggregation{Func: fn}
	p.next()
	// count takes no field; others take (field).
	if p.cur.Type == token.LPAREN {
		p.next()
		if p.cur.Type != token.IDENT {
			return ast.Aggregation{}, p.errf("expected a field inside %s(...)", fn)
		}
		agg.Field = p.cur.Literal
		p.next()
		if p.cur.Type != token.RPAREN {
			return ast.Aggregation{}, p.errf("expected `)` to close %s(", fn)
		}
		p.next()
	}
	agg.Alias = agg.Func
	if agg.Field != "" {
		agg.Alias = agg.Func + "_" + agg.Field
	}
	if p.cur.Type == token.AS {
		p.next()
		if p.cur.Type != token.IDENT {
			return ast.Aggregation{}, p.errf("expected an alias after `as`")
		}
		agg.Alias = p.cur.Literal
		p.next()
	}
	return agg, nil
}

func (p *Parser) parseSort() (ast.Stage, error) {
	p.next()
	s := &ast.SortStage{}
	for {
		if p.cur.Type != token.IDENT {
			return nil, p.errf("expected a field name to sort by")
		}
		key := ast.SortKey{Field: p.cur.Literal}
		p.next()
		if p.cur.Type == token.ASC {
			p.next()
		} else if p.cur.Type == token.DESC {
			key.Desc = true
			p.next()
		}
		s.Keys = append(s.Keys, key)
		if p.cur.Type != token.COMMA {
			break
		}
		p.next()
	}
	return s, nil
}

func (p *Parser) parseLimit() (ast.Stage, error) {
	p.next()
	if p.cur.Type != token.NUMBER {
		return nil, p.errf("limit requires a number")
	}
	n, err := strconv.Atoi(p.cur.Literal)
	if err != nil {
		return nil, p.errf("invalid limit %q", p.cur.Literal)
	}
	p.next()
	return &ast.LimitStage{N: n}, nil
}

// ---- Expression parsing (precedence climbing) ----

// precedence levels; higher binds tighter.
func precedence(t token.Type) int {
	switch t {
	case token.OR:
		return 1
	case token.AND:
		return 2
	case token.EQ, token.NEQ, token.LT, token.LTE, token.GT, token.GTE, token.MATCH, token.NMATCH:
		return 3
	default:
		return 0
	}
}

func (p *Parser) parseExpr(minPrec int) (ast.Expr, error) {
	// `not` binds looser than comparisons: `not a == b` is `not (a == b)`.
	if p.cur.Type == token.NOT {
		p.next()
		operand, err := p.parseExpr(precedence(token.AND)) // not applies to the following comparison/and-group
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Op: "not", Operand: operand}, nil
	}

	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		prec := precedence(p.cur.Type)
		if prec == 0 || prec < minPrec {
			return left, nil
		}
		op := opLiteral(p.cur.Type)
		p.next()
		right, err := p.parseExpr(prec + 1)
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: op, Left: left, Right: right}
	}
}

func (p *Parser) parsePrimary() (ast.Expr, error) {
	switch p.cur.Type {
	case token.LPAREN:
		p.next()
		expr, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		if p.cur.Type != token.RPAREN {
			return nil, p.errf("expected `)`")
		}
		p.next()
		return expr, nil
	case token.IDENT:
		name := p.cur.Literal
		p.next()
		return &ast.FieldExpr{Name: name}, nil
	case token.STRING:
		v := p.cur.Literal
		p.next()
		return &ast.StringLit{Value: v}, nil
	case token.NUMBER:
		f, err := strconv.ParseFloat(p.cur.Literal, 64)
		if err != nil {
			return nil, p.errf("invalid number %q", p.cur.Literal)
		}
		p.next()
		return &ast.NumberLit{Value: f}, nil
	default:
		return nil, p.errf("unexpected token %q in expression", p.cur.Literal)
	}
}

func opLiteral(t token.Type) string {
	switch t {
	case token.AND:
		return "and"
	case token.OR:
		return "or"
	default:
		return t.String()
	}
}
