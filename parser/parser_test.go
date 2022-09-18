package parser

import (
	"quoi/lexer"
	"quoi/token"
	"testing"
)

func printErrs(t *testing.T, errs []lexer.Err) {
	if len(errs) > 0 {
		for i, e := range errs {
			t.Logf("err#%d: %+v\n", i, e)
		}
	}
}

func printTok(t *testing.T, tok token.Token) {
	t.Logf("Token_%s(Lit: %s, Line:Col(%d:%d)\n", tok.Type.String(), tok.Literal, tok.Line, tok.Col)
}

func TestParserAdvance(t *testing.T) {
	input := "\"hey "
	l := lexer.New(input)
	p := New(l)
	printErrs(t, p.lexerErrors)
	//fmt.Printf("%+v\n", p)
	printTok(t, p.tok)
	printTok(t, p.peek())
	p.move()
	printTok(t, p.tok)
	printTok(t, p.peek())
	p.move()
	printTok(t, p.tok)
	printTok(t, p.peek())
}
