package analyzer

import (
	"fmt"
	"quoi/lexer"
	"quoi/parser"
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

func TestPrefExpr1(t *testing.T) {
	input := `
		;int a = (+ 1 2).
		;int a = "3".
		;string a = (+ 1 2).
		;int a = (+ "hello " "world").
		;int a = (+ 1 "hello").
		int b = 5.
		int a = (+ 1 b).
		`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	for _, v := range program.IRStatements {
		fmt.Println(v)
	}
}

func TestList1(t *testing.T) {
	input := `
			;listof int nx = ["hey", 2, 3].
			;listof string names = ["jennifer"].
			;listof int numbers = [40, 50, 7, 567, 517].
			;listof int numbers2 = numbers.
			;listof int nx1 = 1.
			;listof int nx2 = [1, 2].
			;listof string strings = "hey".
			;listof string strings2 = [].
			;listof int a = [1].
			int a = 5.
			int b = a.
			int c = (+ "hey" b).
			listof string strings3 = c.
			`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	for _, v := range program.IRStatements {
		fmt.Println(v)
	}
}

func TestOps1(t *testing.T) {
	input := `
		;int a = (+ 1).
		;int b = (+ "hey" " world").
		;int c = (/ 2).
		;int z = (lt 5 4).
		;bool x = (not (lt 5 6)).
		`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	for _, v := range program.IRStatements {
		fmt.Println(v)
	}
}

func TestTopLevel1(t *testing.T) {
	input := `
		break.
		continue.
		(+ 1 2).
		`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	for _, v := range program.IRStatements {
		fmt.Println(v)
	}
}

func TestIf1(t *testing.T) {
	input := `
		string y = "Hello".
		if true {
			int x = 1.
		} elseif false {
			int y = 6.
		} else {
			; this 'y' should refer to "int y = 6" above.
			int x = y.
		}
		;if "hey" {}
		if (lt 5 6) {
			string q = "hey".
		}
		;string qq = q.
	`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	_ = program
}

func TestIf2(t *testing.T) {
	input := `
		if true {
			datatype X {}
		} elseif false {
			fun w() -> {}
		} else {
		}
	`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	_ = program
}

func TestDatatype1(t *testing.T) {
	input := `
	datatype X {}
	;datatype X {}
	datatype Y {
		string x
		;int x
		int y
		User user
	}
`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	_ = program
}
