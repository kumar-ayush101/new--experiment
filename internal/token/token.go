// Package token defines the lexical tokens of the LogQL language.
package token

// Type identifies the kind of a token.
type Type int

const (
	ILLEGAL Type = iota
	EOF

	// Identifiers and literals.
	IDENT  // field names, stage names, function names
	STRING // "quoted"
	NUMBER // 123, 12.5

	// Operators.
	PIPE   // |
	EQ     // ==
	NEQ    // !=
	LT     // <
	LTE    // <=
	GT     // >
	GTE    // >=
	MATCH  // =~  (regex match)
	NMATCH // !~  (regex not match)
	ASSIGN // =   (used in extract: field = pattern)
	COMMA  // ,
	LPAREN // (
	RPAREN // )

	// Keywords.
	AND
	OR
	NOT
	BY
	ASC
	DESC
	AS
)

// Token is a lexical token with its literal text and source position.
type Token struct {
	Type    Type
	Literal string
	Pos     int // byte offset in the source, for error messages
}

var keywords = map[string]Type{
	"and":  AND,
	"or":   OR,
	"not":  NOT,
	"by":   BY,
	"asc":  ASC,
	"desc": DESC,
	"as":   AS,
}

// LookupIdent returns the keyword type for an identifier, or IDENT if it is
// not a reserved word.
func LookupIdent(ident string) Type {
	if t, ok := keywords[ident]; ok {
		return t
	}
	return IDENT
}

// String renders a token type for debugging and error messages.
func (t Type) String() string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case IDENT:
		return "IDENT"
	case STRING:
		return "STRING"
	case NUMBER:
		return "NUMBER"
	case PIPE:
		return "|"
	case EQ:
		return "=="
	case NEQ:
		return "!="
	case LT:
		return "<"
	case LTE:
		return "<="
	case GT:
		return ">"
	case GTE:
		return ">="
	case MATCH:
		return "=~"
	case NMATCH:
		return "!~"
	case ASSIGN:
		return "="
	case COMMA:
		return ","
	case LPAREN:
		return "("
	case RPAREN:
		return ")"
	case AND:
		return "and"
	case OR:
		return "or"
	case NOT:
		return "not"
	case BY:
		return "by"
	case ASC:
		return "asc"
	case DESC:
		return "desc"
	case AS:
		return "as"
	default:
		return "UNKNOWN"
	}
}
