// Package lexer turns LogQL source text into a stream of tokens.
package lexer

import (
	"github.com/kumar-ayush101/new--experiment/internal/token"
)

// Lexer scans source text one token at a time.
type Lexer struct {
	src  string
	pos  int // current byte offset
	read int // next byte offset
	ch   byte
}

// New creates a Lexer over src.
func New(src string) *Lexer {
	l := &Lexer{src: src}
	l.advance()
	return l
}

func (l *Lexer) advance() {
	if l.read >= len(l.src) {
		l.ch = 0
	} else {
		l.ch = l.src[l.read]
	}
	l.pos = l.read
	l.read++
}

func (l *Lexer) peek() byte {
	if l.read >= len(l.src) {
		return 0
	}
	return l.src[l.read]
}

// Next returns the next token in the source.
func (l *Lexer) Next() token.Token {
	l.skipSpaceAndComments()
	start := l.pos

	switch l.ch {
	case 0:
		return token.Token{Type: token.EOF, Pos: start}
	case '|':
		l.advance()
		return tok(token.PIPE, "|", start)
	case ',':
		l.advance()
		return tok(token.COMMA, ",", start)
	case '(':
		l.advance()
		return tok(token.LPAREN, "(", start)
	case ')':
		l.advance()
		return tok(token.RPAREN, ")", start)
	case '=':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return tok(token.EQ, "==", start)
		}
		if l.peek() == '~' {
			l.advance()
			l.advance()
			return tok(token.MATCH, "=~", start)
		}
		l.advance()
		return tok(token.ASSIGN, "=", start)
	case '!':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return tok(token.NEQ, "!=", start)
		}
		if l.peek() == '~' {
			l.advance()
			l.advance()
			return tok(token.NMATCH, "!~", start)
		}
		l.advance()
		return tok(token.ILLEGAL, "!", start)
	case '<':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return tok(token.LTE, "<=", start)
		}
		l.advance()
		return tok(token.LT, "<", start)
	case '>':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return tok(token.GTE, ">=", start)
		}
		l.advance()
		return tok(token.GT, ">", start)
	case '"', '\'':
		return l.lexString(l.ch, start)
	}

	if isLetter(l.ch) {
		return l.lexIdent(start)
	}
	if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peek())) {
		return l.lexNumber(start)
	}

	ch := l.ch
	l.advance()
	return tok(token.ILLEGAL, string(ch), start)
}

func (l *Lexer) lexString(quote byte, start int) token.Token {
	l.advance() // consume opening quote
	var b []byte
	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' && (l.peek() == quote || l.peek() == '\\') {
			l.advance()
		}
		b = append(b, l.ch)
		l.advance()
	}
	if l.ch == quote {
		l.advance() // consume closing quote
	}
	return token.Token{Type: token.STRING, Literal: string(b), Pos: start}
}

func (l *Lexer) lexIdent(start int) token.Token {
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == '.' {
		l.advance()
	}
	lit := l.src[start:l.pos]
	return token.Token{Type: token.LookupIdent(lit), Literal: lit, Pos: start}
}

func (l *Lexer) lexNumber(start int) token.Token {
	if l.ch == '-' {
		l.advance()
	}
	for isDigit(l.ch) {
		l.advance()
	}
	if l.ch == '.' && isDigit(l.peek()) {
		l.advance()
		for isDigit(l.ch) {
			l.advance()
		}
	}
	return token.Token{Type: token.NUMBER, Literal: l.src[start:l.pos], Pos: start}
}

func (l *Lexer) skipSpaceAndComments() {
	for {
		switch {
		case l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r':
			l.advance()
		case l.ch == '#': // line comment
			for l.ch != '\n' && l.ch != 0 {
				l.advance()
			}
		default:
			return
		}
	}
}

func tok(t token.Type, lit string, pos int) token.Token {
	return token.Token{Type: t, Literal: lit, Pos: pos}
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// Tokenize returns all tokens up to and including EOF. Convenience for tests.
func Tokenize(src string) []token.Token {
	l := New(src)
	var out []token.Token
	for {
		t := l.Next()
		out = append(out, t)
		if t.Type == token.EOF {
			return out
		}
	}
}
