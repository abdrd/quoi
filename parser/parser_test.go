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
		int i = 0.
		int age= 30.
		string name="Jennifer".
		bool is_raining=       true.`
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	if len(parsed.Stmts) != 4 {
		t.Errorf("1: %d\n", len(parsed.Stmts))
	}
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func commonThing(t *testing.T, input string) {
	l := lexer.New(input)
	p := New(l)
	parsed := p.Parse()
	printErrs(t, p.lexerErrors)
	printErrs1(t, p.Errs)
	printStmts(t, parsed.Stmts)
}

func TestParseVarDecl2(t *testing.T) {
	input := `
		int age = "Hey".
		int city = true.
		string name = 67.
		bool is_raining = true.
	`
	commonThing(t, input)
}

func TestParseReassignment(t *testing.T) {
	input := `
		name = "Abidin".
		age=35.
		age =65.
		weather =  "Sunny".
	`
	commonThing(t, input)
}

func TestBlock1(t *testing.T) {
	input := `
		block
		end
	`
	commonThing(t, input)
}

func TestBlock2(t *testing.T) {
	input := `
		block
	`
	commonThing(t, input)
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
	commonThing(t, input)
}

func TestReturn1(t *testing.T) {
	input := "return."
	commonThing(t, input)
}

func TestReturn2(t *testing.T) {
	input := "return \"hello guys\""
	commonThing(t, input)
}

func TestReturn3(t *testing.T) {
	input := `return @strconcat "Hello " "world".`
	commonThing(t, input)
}

func TestLoop1(t *testing.T) {
	input := `
		loop  {}
	`
	commonThing(t, input)
}

func TestLoop2(t *testing.T) {
	input := `
		loop (lte 5 5) {
	`
	commonThing(t, input)
}

func TestLoop3(t *testing.T) {
	input := `
		loop (lte 5 5) {
			print "Heeey".
		}
	`
	commonThing(t, input)
}

func TestDatatype1(t *testing.T) {
	input := "datatype{}"
	commonThing(t, input)
}

func TestDatatype2(t *testing.T) {
	input := "datatype {}"
	commonThing(t, input)
}

func TestDatatype3(t *testing.T) {
	input := "datatype"
	commonThing(t, input)
}

func TestDatatype4(t *testing.T) {
	input := "datatype{"
	commonThing(t, input)
}

func TestDatatype5(t *testing.T) {
	input := "datatype {"
	commonThing(t, input)
}

func TestDatatype6(t *testing.T) {
	input := "datatype }"
	commonThing(t, input)
}

func TestDatatype7(t *testing.T) {
	input := "datatype "
	commonThing(t, input)
}

func TestDatatype8(t *testing.T) {
	input := `datatype City {`
	commonThing(t, input)
}

func TestDatatype9(t *testing.T) {
	commonThing(t, `datatype City{`)
}

func TestDatatype10(t *testing.T) {
	commonThing(t, `datatype City}`)
}

func TestDatatype11(t *testing.T) {
	commonThing(t, `datatype City {}`)
}

func TestDatatype12(t *testing.T) {
	commonThing(t, `datatype City {`)
}

func TestDatatype13(t *testing.T) {
	commonThing(t, `datatype City`)
}

func TestDatatype14(t *testing.T) {
	input := `
		datatype City {
			string name }
		print a.
	`
	commonThing(t, input)
}

func TestDatatype15(t *testing.T) {
	input := `
		datatype City {
			string name
}
	`
	commonThing(t, input)
}

func TestDatatype16(t *testing.T) {
	input := `
		datatype City { 
			string name
			int founded_in
			bool is_beautiful
}
	print "something".
	`
	commonThing(t, input)
}

func TestGeneral1(t *testing.T) {
	input := `
		datatype X {
			int y
			string z
			bool x
		}
		return true.
		int i = 0.
		loop true {
			string y = "y".
		}
		block
			int n = 0.
		end
`
	commonThing(t, input)
}

func TestPrefixExpr1(t *testing.T) {
	input := `(+ 2 2)`
	commonThing(t, input)
}

func TestPrefixExpr2(t *testing.T) {
	input := `(+)`
	commonThing(t, input)
}

func TestPrefixExpr3(t *testing.T) {
	input := `(+ 2 datatype)`
	commonThing(t, input)
}

func TestPrefixExpr4(t *testing.T) {
	input := `(+ 2 `
	commonThing(t, input)
}

func TestPrefixExpr5(t *testing.T) {
	input := `(+ 2 2`
	commonThing(t, input)
}

func TestPrefixExpr6(t *testing.T) {
	input := `
		(* 3 (/ 6 2))
	`
	commonThing(t, input)
}

func TestPrefixExpr7(t *testing.T) {
	input := `
		(not
			(and true true)
			   2
		)
	`
	commonThing(t, input)
}

func TestPrefixExpr8(t *testing.T) {
	input := `(not 2 2)`
	commonThing(t, input)
}

func TestParseNot1(t *testing.T) {
	input := `(not a)`
	commonThing(t, input)
}

func TestParseNot2(t *testing.T) {
	input := `(not a`
	commonThing(t, input)
}

func TestParseNot3(t *testing.T) {
	input := `(not a b c)`
	commonThing(t, input)
}

func TestParseNot4(t *testing.T) {
	input := `(not fun)`
	commonThing(t, input)
}

func TestParseNot5(t *testing.T) {
	input := `(
		not true
		)`
	commonThing(t, input)
}

func TestParsePref1(t *testing.T) {
	input := `
		(+ 2 2)
	`
	commonThing(t, input)
}

func TestParsePref2(t *testing.T) {
	input := `
		(/ 2 2
	`
	commonThing(t, input)
}

func TestParsePref3(t *testing.T) {
	input := `
		(/) 6
	`
	commonThing(t, input)
}

func TestParsePref4(t *testing.T) {
	input := `
		(^ f b) 5
	`
	commonThing(t, input)
}
func TestParsePref5(t *testing.T) {
	input := `
		(lte 5 5)
	`
	commonThing(t, input)
}

func TestParsePref6(t *testing.T) {
	input := `
		(gte 5 5 6)
	`
	commonThing(t, input)
}

func TestParserDotDot(t *testing.T) {
	input := `... block end`
	commonThing(t, input)
}

func TestParserErroneous1(t *testing.T) {
	input := `
		garbage and here it is
	`
	commonThing(t, input)
}
