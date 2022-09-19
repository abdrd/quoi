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
