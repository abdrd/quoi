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

func printStmts(t *testing.T, stmts []ast.Statement) {
	if len(stmts) > 0 {
		for i, v := range stmts {
			t.Logf("%s Statement#%d: %s\n", reflect.TypeOf(v), i, v.String())
		}
	}
}

func TestParseStringLit(t *testing.T) {
	input := `"Hey""very good"`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	if parsed == nil {
		t.Errorf("parsed is nil")
		t.FailNow()
	}
	if len(parsed.Stmts) < 1 {
		t.Errorf("1: %d\n", len(parsed.Stmts))
	}
	if parsed.Stmts[0].String() != "\"Hey\"" {
		t.Errorf("2: %s\n", parsed.Stmts[0].String())
	}
	printStmts(t, parsed.Stmts)
}

func TestParseIntLit(t *testing.T) {
	input := `1245-1516` // 1245, and -1516
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	if parsed == nil {
		t.Errorf("parsed is nil")
		t.FailNow()
	}
	if len(parsed.Stmts) < 1 {
		t.Errorf("1: %d\n", len(parsed.Stmts))
	}
	printStmts(t, parsed.Stmts)
}

func TestParseBoolLit(t *testing.T) {
	input := `true`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	if parsed == nil {
		t.Errorf("parsed is nil")
		t.FailNow()
	}
	if len(parsed.Stmts) < 1 {
		t.Errorf("1: %d\n", len(parsed.Stmts))
	}
	printStmts(t, parsed.Stmts)
}

func printErrs1(t *testing.T, errs []Err) {
	if len(errs) > 0 {
		for i, e := range errs {
			t.Logf("error#%d: %+v\n", i, e)
		}
	}
}

func TestParseVarDecl1(t *testing.T) {
	input := `
		int age= 30.
		string name="Jennifer".
		bool is_raining=       true.`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	if len(parsed.Stmts) != 3 {
		t.Errorf("1: %d\n", len(parsed.Stmts))
	}
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestParseVarDeclErr(t *testing.T) {
	input := `
		int age = "Hey".
		int city = true.
		string name = 67.
		bool is_raining = true.
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	fmt.Println("err: ", len(parsed.Stmts))
	printStmts(t, parsed.Stmts)
}

func TestParseReassignment(t *testing.T) {
	input := `
		name = "Abidin".
		age=35.
		age =65.
		weather =  "Sunny".
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestParsePrintStmt(t *testing.T) {
	input := `
		print a.
		print        16.
		print "Hey, how are you?".
		print true.
		print false. print "hey this was false".
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestOperatorWithOneArg(t *testing.T) {
	input := `
		@inc a  .
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestOperatorWithTwoArgs(t *testing.T) {
	input := `
		@gte a b.
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestOperatorWithThreeArgs(t *testing.T) {
	input := `
		@strreplace s 1 "h" a Heh
		block 
			print "hey".
		end
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestOperatorUnknown(t *testing.T) {
	input := `
		@unknown "he" "he" a
		block 
			print a.
		end
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestBlock1(t *testing.T) {
	input := `
		block
		end
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestBlock2(t *testing.T) {
	input := `
		block
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestBlock3(t *testing.T) {
	input := `
		block
			print a.
			print @lte 5 5.
		end
		block 
			print "Hello world!".
		end.
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestReturn1(t *testing.T) {
	input := "return."
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestReturn2(t *testing.T) {
	input := "return \"hello guys\""
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestReturn3(t *testing.T) {
	input := `return @strconcat "Hello " "world".`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestLoop1(t *testing.T) {
	input := `
		loop  {}
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestLoop2(t *testing.T) {
	input := `
		loop @lte 5 5 {
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestLoop3(t *testing.T) {
	input := `
		loop @lte 5 5 {
			print "Heeey".
		}
	`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}
