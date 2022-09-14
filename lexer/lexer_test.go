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
		fmt.Printf("tok: %+v\n", tok)
		if tok.Type == token.EOF {
			break
		}
	}
	fmt.Println("l.pos: ", l.pos)
}

func TestLexInt(t *testing.T) {
	input := "212\n"
	input += "1716"
	input += "-315"
	l := New(input)
	for {
		tok := l.NextToken()
		fmt.Printf("tok: %+v\n", tok)
		if tok.Type == token.EOF {
			break
		}
	}
	fmt.Println("l.pos: ", l.pos)
}
