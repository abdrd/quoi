package lexer

import (
	"fmt"
	"quoi/token"
	"testing"
)

func TestNewLexer(t *testing.T) {
	input := "Some text"
	got := New(input)

	if got.ch != 'S' {
		t.Errorf("1: %s\n", string(got.ch))
	}
	if got.hasReachedEOF != false {
		t.Errorf("2: %v\n", got.hasReachedEOF)
	}
	if got.lenSrc != uint(len(input)) {
		t.Errorf("3: %d\n", got.lenSrc)
	}
	if got.pointer != 0 {
		t.Errorf("4: %d\n", got.pointer)
	}
	if got.state != stateStart {
		t.Errorf("5: %d\n", got.state)
	}
}

func TestCanBeAnIdentifierName(t *testing.T) {
	_1 := canBeAnIdentifierName('@')
	_2 := canBeAnIdentifierName('_')
	_3 := canBeAnIdentifierName('3')
	_4 := canBeAnIdentifierName('A')
	_5 := canBeAnIdentifierName('b')
	_6 := canBeAnIdentifierName('\'')
	if !(!_1 && _2 && !_3 && _4 && _5 && !_6) {
		t.Error(_1, _2, _3, _4, _5, _6)
	}
}

func TestLexerAdvance(t *testing.T) {
	input := "Some text"
	l := New(input)
	if l.ch != 'S' {
		t.Errorf("1: %s\n", string(l.ch))
	}
	l.advance()
	if l.ch != 'o' {
		t.Errorf("2: %s\n", string(l.ch))
	}
	for i := 0; i < 7; i++ {
		l.advance()
	}
	if l.hasReachedEOF != true {
		t.Errorf("3: %v\n", l.hasReachedEOF)
	}
	if l.ch != 't' {
		t.Errorf("4: %s\n", string(l.ch))
	}
}

func TestLexerPeek(t *testing.T) {
	input := "Some text"
	l := New(input)
	if p := l.peek(); p != 'o' {
		t.Errorf("1: %s\n", string(p))
	}
	l.advance()
	l.advance()
	if p := l.peek(); p != 'e' {
		t.Errorf("2: %s\n", string(p))
	}
	for i := 0; i < 6; i++ {
		l.advance()
	}
	if p := l.peek(); p != eof {
		t.Errorf("3: %s\n", string(p))
	}
}

func TestPos(t *testing.T) {
	input := "Some test\nHey"
	l := New(input)
	if l.col != 0 {
		t.Errorf("1: %d\n", l.col)
	}
	if l.line != 1 {
		t.Errorf("2: %d\n", l.line)
	}
	l.advance()
	l.advance()
	if l.col != 2 {
		t.Errorf("3: %d\n", l.col)
	}
	if l.line != 1 {
		t.Errorf("4: %d\n", l.line)
	}
	for i := 0; i < 8; i++ {
		l.advance()
	}
	if l.col != 1 {
		t.Errorf("5: %d\n", l.col)
	}
	if l.line != 2 {
		t.Errorf("6: %d\n", l.line)
	}
}

func TestLexWs(t *testing.T) {
	input := "\n   "
	l := New(input)
	tok := l.Next()
	if tok.Type != token.WHITESPACE {
		t.Errorf("1: %s\n", tok.Type)
	}
	if tok.Literal != input {
		t.Errorf("2: %s\n", tok.Literal)
		t.Errorf("2.2: %d\n", len(tok.Literal))
		t.Errorf("2.3: %d\n", l.ch)
	}
}

func TestLexInt(t *testing.T) {
	input := "123\n-1415"
	l := New(input)
	_123 := l.Next()
	_nl := l.Next()
	_m1415 := l.Next()
	if _123.Type != token.INT {
		t.Errorf("1: %s\n", _123.Type)
	}
	if _123.Literal != "123" {
		t.Errorf("2: %s\n", _123.Literal)
	}
	if _m1415.Type != token.INT {
		t.Errorf("3: %s\n", _m1415.Type)
	}
	if _m1415.Literal != "-1415" {
		t.Errorf("4: %s\n", _m1415.Literal)
	}
	t.Logf("%+v\n", _123)
	t.Logf("%+v\n", _nl)
	t.Logf("%+v\n", _m1415)
}

func TestLexIntError(t *testing.T) {
	input := "-\n"
	l := New(input)
	a := l.Next()
	_ = l.Next()

	if len(l.Errs) == 0 {
		t.Errorf("1")
	}
	t.Logf("a: %+v\n", a)
	t.Logf("lexer errors: %+v\n", l.Errs)
}

func TestLexString(t *testing.T) {
	input := `"hello"` + "\n" + `"this is a string"`
	l := New(input)
	first := l.Next()
	_ = l.Next()
	second := l.Next()
	fmt.Printf("first: %+v\n", first)
	fmt.Printf("second: %+v\n", second)
	if first.Type != token.STRING {
		t.Errorf("1: %s\n", first.Type)
	}
	if first.Literal != "hello" {
		t.Errorf("2: %s\n", first.Literal)
	}
	if second.Type != token.STRING {
		t.Errorf("3: %s\n", second.Type)
	}
	if second.Literal != "this is a string" {
		t.Errorf("4: %s\n", second.Literal)
	}
	t.Logf("g")
}

func TestLexString2(t *testing.T) {
	input := `"hello"`
	l := New(input)
	first := l.Next()
	fmt.Printf("first: %+v\n", first)
	if first.Type != token.STRING {
		t.Errorf("1: %s\n", first.Type)
	}
	if first.Literal != "hello" {
		t.Errorf("2: %s\n", first.Literal)
	}
	t.Logf("g")
}

/*
func TestLexStringWithEscape(t *testing.T) {
	input := `"\n"`
	l := New(input)
	tok := l.Next()
	if tok.Type != token.STRING {
		t.Errorf("1: %s\n", tok.Type)
	}
	if tok.Literal != "\n" {
		t.Errorf("2: %s\n", tok.Literal)
	}
	if len(l.Errs) > 0 {
		for _, v := range l.Errs {
			t.Logf("lexer err: %+v\n", v)
		}
	}
}
*/

func TestLexStringError(t *testing.T) {
	input := `"hey`
	l := New(input)
	fmt.Println(l.state)
	tok := l.Next()
	if tok.Type != token.STRING {
		t.Errorf("1: %s\n", tok.Type)
	}
	if tok.Literal != "hey" {
		t.Errorf("2: %s\n", tok.Literal)
	}
	if len(l.Errs) == 0 {
		t.Errorf("3")
	}
	t.Logf("lexer errors: %+v", l.Errs)
}
