package analyzer

import (
	"fmt"
	"quoi/ast"
	"quoi/lexer"
	"quoi/parser"
	"quoi/token"
	"testing"
)

func _new(input string) *Analyzer {
	l := lexer.New(input)
	for _, v := range l.Errs {
		fmt.Printf("lexer err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
	}
	p := parser.New(l)
	program := p.Parse()
	for _, v := range p.Errs {
		fmt.Printf("parser err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
	}
	a := New(program)
	return a
}

func TestFirstPass1(t *testing.T) {
	input := `
		fun hello(int a) -> int, string, bool, User {
			return 5.
		}
	`
	a := _new(input)
	a.Analyze()
	x := a.env.GetFunc("hello")
	fmt.Println(x)
	for _, v := range a.Errs {
		t.Logf("analyzer err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
	}
}

func TestFirstPass2(t *testing.T) {
	input := `
		datatype User { 
			string name
			int age
			City city
		}
	`
	a := _new(input)
	a.Analyze()
	x := a.env.GetDatatype("User")
	fmt.Println(x)
	for _, v := range a.Errs {
		t.Logf("analyzer err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
	}
}

func TestTC1(t *testing.T) {
	input := &ast.Program{Stmts: []ast.Statement{
		&ast.PrefixExpr{Tok: token.New(token.ADD, "+", 1, 2), Args: []ast.Expr{
			&ast.IntLiteral{Typ: token.New(token.INT, "5", 1, 4), Val: 5},
			&ast.IntLiteral{Typ: token.New(token.INT, "6", 1, 6), Val: 6},
		}},
	}}
	a := New(input)
	res := a.typecheckExpr(a.program.Stmts[0], "int")
	fmt.Println(res)
}

func TestTC2(t *testing.T) {
	input := &ast.Program{Stmts: []ast.Statement{
		&ast.PrefixExpr{Tok: token.New(token.ADD, "+", 1, 2), Args: []ast.Expr{
			&ast.StringLiteral{Typ: token.New(token.STRING, "Hello", 1, 4), Val: "Hello"},
			//&ast.StringLiteral{Typ: token.New(token.INT, "World", 1, 6), Val: "World"},
			&ast.IntLiteral{Typ: token.New(token.INT, "6", 1, 6), Val: 6},
		}},
	}}
	a := New(input)
	res := a.typecheckExpr(a.program.Stmts[0], "")
	fmt.Println(res)
}

func TestTC3(t *testing.T) {
	input := `
		;int a = 5.
		;int b = "string".
		;int c = (+ 1 2).
		;int d = (+ 1 2 3 (* 2 3)).
		;int e = (* "hey" (+ 1 2)).
		;bool aa = (= "hello" "hello" "Hi").
		;bool ab = (= 4 "hey").
		;bool ac = (= "hello" "hi" (+ 4 5)).
		;bool ad = (not (not (not (+ 2 2)))).
		;bool ae = (and true (+ "hello" " world")).
		;string aaa = "hello".
		;string aab = (= "hey" "Hey").
		int a = (not false).
		`
	// TODO string aab = (= "hey" "Hey").
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("analyzer err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	t.Logf("program: \n")
	t.Logf("%s\n", program)
}
