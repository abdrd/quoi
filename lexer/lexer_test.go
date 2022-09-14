package lexer

import (
	"fmt"
	"quoi/token"
	"testing"
)

func TestLexWs(t *testing.T) {
	input := " "
	input += "  \n   	"
	l := New(input)
	for {
		tok := l.NextToken()
		fmt.Printf("ws: %+v\n", tok)
		if tok.Type == token.EOF {
			break
		}
	}
	fmt.Println("l.pos: ", l.pos)
}

/*
func TestLexInt(t *testing.T) {
	input :=
}*/
