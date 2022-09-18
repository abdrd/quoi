package parser

import (
	"quoi/lexer"
	"quoi/token"
)

type Parser struct {
	tokens         []token.Token
	pos, lenTokens uint
	hasReachedEOF  bool // whether the next token's type is token.EOF
	lexerErrors    []lexer.Err
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
	p.pos = 0
	p.tokens = toks
	p.lenTokens = uint(len(p.tokens))
	p.hasReachedEOF = p.tokens[p.pos].Type == token.EOF
	return p
}

func (p *Parser) next() token.Token {
	if p.pos == p.lenTokens-1 {
		return p.tokens[p.lenTokens-1]
	}
	if p.pos == p.lenTokens-2 {
		p.hasReachedEOF = true
	}
	return p.tokens[p.pos+1]
}
