package sema

import (
	"fmt"
	"quoi/lexer"
	"quoi/parser"
	"testing"
)

func _nr(input string) *Resolver {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.Parse()
	return NewResolver(program)
}

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
	r := _nr(input)
	r.Resolve()
	fmt.Println(r.stackOfScopes.findSymbol("x"))
	fmt.Println(r.stackOfScopes.findSymbol("y"))
	fmt.Println(r.stackOfScopes.findSymbol("xx"))
}

func TestCheckVarDecl1(t *testing.T) {
	input := `
		;int a = "hello".
		int a = 5.
		bool y = 156.
	`
	r := _nr(input)
	r.Resolve()
}

func TestCheckPrefExpr1(t *testing.T) {
	input := `
		;int a = (+ 1 true).
		;bool b = (+ 1 1).
		;string c = (* 3 5).
		;int d = (* 6 7 8 "hey").
		int e = (* 2 3 (+ 1 2 3)).
		`
	r := _nr(input)
	r.Resolve()
}

func TestCheckPrefExpr2(t *testing.T) {
	input := `
			;bool a = (and true true).
			;bool b = (not (and true true))
			bool c = (not true true).
			`
	r := _nr(input)
	r.Resolve()
}
