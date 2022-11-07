package sema

import (
	"fmt"
	"quoi/lexer"
	"quoi/parser"
	"testing"
)

func TestResolve(t *testing.T) {
	input := `
		;int x = 0.
		;string y = "hey".
		;listof int xx = [1, 2, 3, 4].
		;User { name="Jennifer" }.
		;continue.
		;break.
		;5.
		;true.
		;(+ 5 5 5 5 5   5).
		`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.Parse()
	r := NewResolver(program)
	r.Resolve()
	fmt.Println(r.stackOfScopes.findSymbol("x"))
	fmt.Println(r.stackOfScopes.findSymbol("y"))
	fmt.Println(r.stackOfScopes.findSymbol("xx"))
}
