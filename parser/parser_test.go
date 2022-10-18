package parser

import (
	"fmt"
	"quoi/ast"
	"quoi/lexer"
	"quoi/token"
	"reflect"
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
	input := "hey "
	l := lexer.New(input)
	p := New(l)
	printErrs(t, p.lexerErrors)
	//fmt.Printf("%+v\n", p)
	fmt.Println("===========")
	printTok(t, p.tok)
	printTok(t, p.peek())
	fmt.Println("===========")
	p.move()
	fmt.Println("===========")
	printTok(t, p.tok)
	printTok(t, p.peek())
	fmt.Println("===========")
	p.move()
	fmt.Println("===========")
	printTok(t, p.tok)
	printTok(t, p.peek())
	fmt.Println("===========")
}

func _parse(input string) (*ast.Program, []Err, []lexer.Err) {
	l := lexer.New(input)
	p := New(l)
	return p.Parse(), p.Errs, p.lexerErrors
}

func check_stmt_count(t *testing.T, program *ast.Program, expectedNum int) {
	if lps := len(program.Stmts); lps != expectedNum {
		t.Errorf("ERROR <!!!> len(*ast.Program.Stmts) != %d, but '%d'\n\n", expectedNum, lps)
	}
}

/*
func check_lit(t *testing.T, node ast.Node, expectedLit string) {
	if ns := node.String(); ns != expectedLit {
		t.Errorf("node.String() != '%s', but '%s'\n", expectedLit, ns)
	}
}*/

func check_error_count(t *testing.T, errs []Err, expectedNum int) {
	if lpe := len(errs); lpe != expectedNum {
		t.Errorf("ERROR <!!!> len(*Parser.Errs) != %d, but '%d'\n\n", expectedNum, lpe)
	}
}

func print_errs(t *testing.T, errs []Err) {
	for _, v := range errs {
		t.Logf("___________________________________________\n")
		t.Logf("line:col %d:%d :: %s\n", v.Line, v.Column, v.Msg)
		t.Logf("___________________________________________\n")
	}
}

func print_stmts(t *testing.T, program *ast.Program) {
	for _, v := range program.Stmts {
		t.Logf("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
		t.Logf("%s :: %s\n", reflect.TypeOf(v), v)
		t.Logf("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
	}
}

/*
literals,
	string, identifier, int, bool, list
datatype,
block,
prefix expr
variable declaration,
	int, string, bool, list
reassignment
function call
function call from namespace
return
loop
*/

func TestLit1(t *testing.T) {
	input := `
			"Hello".
			1316.
			-5471.
			true.
			false.
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	check_stmt_count(t, program, 5)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestVarDecl1(t *testing.T) {
	input := `
		int n = 4.
		string x = "Hello".
		bool y = true.
		User u = "User#1".
		User u = "hey".
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	check_stmt_count(t, program, 5)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestVarDecl2(t *testing.T) {
	input := `
		int a = 1
		User u = "hey".
		bool x true.
		listof } a = [].
		string
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 4)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestVarDecl3(t *testing.T) {
	input := `
		; listof string names = ["Jennifer", "Hasan", "Ali", "Ayşe",].
		; listof string a = ["He",]
		; listof string a = ["He",
		listof string names = ["Jennifer", "Hasan", "Ali", "Ayşe"].
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestDT1(t *testing.T) {
	input := `
		datatype City {
			string name
			int x
			int y 
			bool z
			User u

			; hello
		}
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestDT2(t *testing.T) {
	input := `
		; datatype City {}
		; datatype {}
		; datatype X {
		; datatype X }
		; datatype X { int x }
		; datatype X { 
		;	int x string name
		;}
		datatype X { int x
		}
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestBlock1(t *testing.T) {
	input := `
		block 
			Stdout::println("Hey").
			print_it(1416).
		end
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestBlock2(t *testing.T) {
	input := `
;		block 
;			Stdout::println("Hey").
;			print_it(1416).
;	block
;	end
	block "Hey". end
`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestPrefExpr1(t *testing.T) {
	input := `
		(+ 1 4).								; 5
		(+ (* Int::from_string("5") 5) 2). 		; 27
		(' ["Hey", "Hello"] 0).					; "Hey"									


	(* 2 Int::from_string(String::from_int(
		(+ 3 5 18925
			Int::from_string("-1516")
		),
	))). 										; 34834

	Stdout::println((* 4 Math::pow(2, 2))).		; 16
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestPrefExpr2(t *testing.T) {
	input := `
		;(m 4 5 67).
		;().
		;(+ 5 6 7 8 9
		;(' [0, 1, 2, 3, 4] 2)
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 1)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestRA1(t *testing.T) {
	input := `
		name = "Hey".
		age = 51.
		u = "User".
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestRA2(t *testing.T) {
	input := `
		;age =.
		;name = "Hey"
		;u = 
		a=    "716".
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestBreakAndReturn(t *testing.T) {
	input := `
		;break
		;continue
		block
			continue.
			break.
		end
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestFC1(t *testing.T) {
	input := `
		string_concat("Hello", "World").
		Os::read_file("hello.txt").
		Math::pow(
			2, 2,
		).
		Stdout::println(
			1, 2,  3,
			5, "Hello", "Yay",
		).
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}

func TestFC2(t *testing.T) {
	input := `
		;string_concat("Hello" "World")
		;Os::read_file ("hello.txt" ).
		;Math::pow(2, 2,).
		;Math::pow(2, 2)
	`
	program, errs, _ := _parse(input)
	check_error_count(t, errs, 0)
	print_stmts(t, program)
	print_errs(t, errs)
}
