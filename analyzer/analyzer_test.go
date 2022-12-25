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

func TestOps2(t *testing.T) {
	input := `
		;bool x = (= 5 5).
		;bool x = (= 5 "hey").
		;bool x = (= "hey" 5).
		;int x = (= "hey" "hey").
		;int x = (+ true true).

		int x = 555.
		int y, string q = x, "Hello".
		int total = (+ 1 2 3 q).
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

func TestSubseq1(t *testing.T) {
	input := `
		listof int nx, listof string strx = [1, 2, 3], ["h", "e", "y"].
		listof int nx2, listof string strx2 = nx, strx.
		;listof int nxq = strx.
		;int x, listof string strx, bool y = 1, [], true.
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

func TestReas1(t *testing.T) {
	input := `
		int x = 1.
		int y = x.
		x = 2.
		int q = x.
	`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	/*
		fmt.Println(program.IRStatements[0].(*IRVariable).Value.(*IRInt).Value)
		fmt.Println(program.IRStatements[1].(*IRReassigment).NewValue.(*IRInt).Value)
	*/
	fmt.Println(program.IRStatements[0].(*IRVariable).Value)
	fmt.Println(program.IRStatements[1].(*IRVariable).Value.(*IRVariableReference).Value)
	fmt.Println(program.IRStatements[2].(*IRReassigment).NewValue)
	fmt.Println(program.IRStatements[3].(*IRVariable).Value.(*IRVariableReference).Value)
}

func TestBlock1(t *testing.T) {
	input := `
		int x = 10.
		;int xx = 100.
		block 
			int x = 1.
			int y = (+ x 1).
			;string s = xx.
		end
		;int q = x.
		block 
;			break.
;			continue.
		end
	`
	a := _new(input)
	program := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
	fmt.Println(program.IRStatements[1].(*IRBlock).Stmts[1].(*IRVariable).Value.(*IRPrefExpr).Operands[0].(*IRVariableReference).Value)
}

func TestLoop1(t *testing.T) {
	input := `
		;loop (+ 1 2) {
		;	int x = 1.
		;}

		int i = 0.
		loop (lt i 10) {
			i = (+ i 1).
			;fun a() -> string {}
			;datatype Song {
			;	string name
			;	int year
			;}
			break.
			continue.
		}
	`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestTopLevel2(t *testing.T) {
	input := `
		"Hello".
		1.
		true.
		User{}.
		(+ 1 2).
	`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestFun1(t *testing.T) {
	input := `
		fun a() -> {}
		;fun b() -> string {}
		;fun c() -> { return 1. }
		;fun d() -> string { return 1. }
		;fun e() -> string { return "Hello". }
		;fun f(string b) -> string { return b. }
		;int b = 6.
		;fun g(string z) -> int { return b. }
		;fun h(string b) -> string { return b. }
		fun j(listof string names) -> listof string, int { return names, 5. }
		`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}

func TestFun2(t *testing.T) {
	/* 		fun a() -> int {
		if true {
			return 5.
		}
	}

			fun a() -> int {
			block end
			if true {}
			loop true {}
		}

				fun a() -> int {
			if true {
				if true {

				} elseif false {
					return "5".
				}
			}
		}
	*/
	input := `
		fun a() -> int, listof bool {
			loop true {
				if true {

				} elseif false {

				} else {
					return 5, [true, true].
					if true {
						if true {

						} else {
							block
								if true {
									return 1, [false, "hey"].
								} 
							end
						}
					}
				}
			}
		}
	`
	a := _new(input)
	_ = a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			t.Logf("Analyzer err : %d:%d -- %s\n", v.Line, v.Column, v.Msg)
		}
		return
	}
}
