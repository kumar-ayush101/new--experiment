package lexer

import (
	"testing"

	"github.com/kumar-ayush101/new--experiment/internal/token"
)

func TestLexPipeline(t *testing.T) {
	src := `parse logfmt | where status >= 500 and path =~ "/api" | stats count by path | sort count desc | limit 10`
	want := []token.Type{
		token.IDENT, token.IDENT, // parse logfmt
		token.PIPE, token.IDENT, token.IDENT, token.GTE, token.NUMBER, token.AND, token.IDENT, token.MATCH, token.STRING,
		token.PIPE, token.IDENT, token.IDENT, token.BY, token.IDENT,
		token.PIPE, token.IDENT, token.IDENT, token.DESC,
		token.PIPE, token.IDENT, token.NUMBER,
		token.EOF,
	}
	got := Tokenize(src)
	if len(got) != len(want) {
		t.Fatalf("token count = %d, want %d\ngot: %v", len(got), len(want), types(got))
	}
	for i, w := range want {
		if got[i].Type != w {
			t.Fatalf("token %d = %v (%q), want %v", i, got[i].Type, got[i].Literal, w)
		}
	}
}

func TestLexOperators(t *testing.T) {
	src := `== != < <= > >= =~ !~ = | , ( )`
	want := []token.Type{
		token.EQ, token.NEQ, token.LT, token.LTE, token.GT, token.GTE,
		token.MATCH, token.NMATCH, token.ASSIGN, token.PIPE, token.COMMA,
		token.LPAREN, token.RPAREN, token.EOF,
	}
	got := Tokenize(src)
	for i, w := range want {
		if got[i].Type != w {
			t.Fatalf("op %d = %v, want %v", i, got[i].Type, w)
		}
	}
}

func TestLexStringsAndNumbers(t *testing.T) {
	got := Tokenize(`"hello world" 'single' 42 3.14 -7`)
	if got[0].Type != token.STRING || got[0].Literal != "hello world" {
		t.Fatalf("string 0 = %+v", got[0])
	}
	if got[1].Type != token.STRING || got[1].Literal != "single" {
		t.Fatalf("string 1 = %+v", got[1])
	}
	if got[2].Type != token.NUMBER || got[2].Literal != "42" {
		t.Fatalf("number 42 = %+v", got[2])
	}
	if got[3].Type != token.NUMBER || got[3].Literal != "3.14" {
		t.Fatalf("number 3.14 = %+v", got[3])
	}
	if got[4].Type != token.NUMBER || got[4].Literal != "-7" {
		t.Fatalf("number -7 = %+v", got[4])
	}
}

func TestLexEscapesAndComments(t *testing.T) {
	got := Tokenize(`"a\"b" # trailing comment
msg`)
	if got[0].Type != token.STRING || got[0].Literal != `a"b` {
		t.Fatalf("escaped string = %+v", got[0])
	}
	if got[1].Type != token.IDENT || got[1].Literal != "msg" {
		t.Fatalf("ident after comment = %+v", got[1])
	}
}

func types(ts []token.Token) []token.Type {
	out := make([]token.Type, len(ts))
	for i, t := range ts {
		out[i] = t.Type
	}
	return out
}
