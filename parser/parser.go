package parser

import (
	"quoi/lexer"
	"quoi/token"
)

type Parser struct {
	tokens      []token.Token
	ptr         uint
	tok         token.Token // current token pointed to, by ptr
	lexerErrors []lexer.Err
}

func New(l *lexer.Lexer) *Parser {
	toks := []token.Token{}
	for {
		t := l.Next()
		toks = append(toks, t)
		if t.Type == token.EOF {
			break
		}
	}
	if len(toks) == 0 {
		panic("lexer.New: error: len(tokens) is zero (0)")
	}
	p := &Parser{}
	p.tokens = toks
	p.lexerErrors = l.Errs
	return p
}

// increment pointer
func (p *Parser) move() {
	if p.ptr == uint(len(p.tokens)) {
		return
	}
	p.ptr++
	p.tok = p.tokens[p.ptr]
}

func (p *Parser) peek() token.Token {
	if p.tok.Type == token.EOF {
		return p.tok
	}
	return p.tokens[p.ptr+1]
}
