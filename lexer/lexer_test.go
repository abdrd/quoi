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
	fmt.Printf("len(tok.Literal)=%d  len(input)=%d", len(tok.Literal), len(input))
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

func check1(t *testing.T, val token.Token, expectedLit string, expectedType token.Type) {
	if val.Literal != expectedLit {
		t.Errorf("expected literal '%s', but got '%s'\n", expectedLit, val.Literal)
	}
	if val.Type != expectedType {
		t.Errorf("expected type '%s', but got '%s'\n", expectedType, val.Type)
	}
}

func TestLexIdent(t *testing.T) {
	input := "hello first_name _hey person15"
	l := New(input)
	hello := l.Next()
	_ = l.Next()
	first_name := l.Next()
	_ = l.Next()
	_hey := l.Next()
	_ = l.Next()
	person15 := l.Next()
	check1(t, hello, "hello", token.IDENT)
	check1(t, first_name, "first_name", token.IDENT)
	check1(t, _hey, "_hey", token.IDENT)
	check1(t, person15, "person15", token.IDENT)
}

func printErrs(t *testing.T, errs []Err) {
	if len(errs) > 0 {
		for i, e := range errs {
			t.Logf("err#%d: %+v\n", i, e)
		}
	}
}

func printTok(t *testing.T, tok token.Token) {
	t.Logf("Token_%s(Lit: %s, Line:Col(%d:%d)\n", tok.Type.String(), tok.Literal, tok.Line, tok.Col)
}

func TestLexIdentErrors(t *testing.T) {
	input := "@hey"
	l := New(input)
	ill := l.Next()
	hey := l.Next()
	check1(t, ill, "@", token.ILLEGAL)
	check1(t, hey, "hey", token.IDENT)
	printTok(t, ill)
	printTok(t, hey)
}

func TestLexKw(t *testing.T) {
	input := "fun datatype if\nblock"
	l := New(input)
	fun := l.Next()
	_ = l.Next()
	datatype := l.Next()
	_ = l.Next()
	if_ := l.Next()
	_ = l.Next()
	block := l.Next()
	check1(t, fun, "fun", token.FUN)
	check1(t, datatype, "datatype", token.DATATYPE)
	check1(t, if_, "if", token.IF)
	check1(t, block, "block", token.BLOCK)
}

func TestLexSymbol(t *testing.T) {
	input := "{.)->"
	l := New(input)
	lcurly := l.Next()
	dot := l.Next()
	closingParen := l.Next()
	arrow := l.Next()
	check1(t, lcurly, "{", token.OPENING_CURLY)
	check1(t, dot, ".", token.DOT)
	check1(t, closingParen, ")", token.CLOSING_PAREN)
	check1(t, arrow, "->", token.ARROW)
	printTok(t, lcurly)
	printTok(t, dot)
	printTok(t, closingParen)
	printTok(t, arrow)
	printErrs(t, l.Errs)
}

func TestLexNewlineInString(t *testing.T) {
	input := `"Hello
guys"`
	l := New(input)
	str := l.Next()
	check1(t, str, "Hello\nguys", token.STRING)
	printTok(t, str)
	printErrs(t, l.Errs)
}

func TestLexBoolLiteral(t *testing.T) {
	input := "true false"
	l := New(input)
	tr := l.Next()
	_ = l.Next()
	f := l.Next()
	check1(t, tr, "true", token.BOOL)
	check1(t, f, "false", token.BOOL)
	printTok(t, tr)
	printTok(t, f)
	printErrs(t, l.Errs)
}

func TestStringPrecedingWhitespace(t *testing.T) {
	input := `"Hello"  `
	l := New(input)
	str := l.Next()
	ws := l.Next()
	check1(t, str, "Hello", token.STRING)
	//check1(t, ws, "  ", token.WHITESPACE)
	printTok(t, str)
	printTok(t, ws)
	printErrs(t, l.Errs)
}

func TestStartingWithString(t *testing.T) {
	input := "\"hey "
	l := New(input)
	str := l.Next()
	printErrs(t, l.Errs)
	printTok(t, str)
}

func TestJustString(t *testing.T) {
	input := `"Hey"`
	l := New(input)
	str := l.Next()
	str2 := l.Next()
	printTok(t, str)
	printTok(t, str2)
	printErrs(t, l.Errs)
}

func TestJustInt(t *testing.T) {
	input := "1246-1516"
	l := New(input)
	_1246 := l.Next()
	_m1516 := l.Next()
	printTok(t, _1246)
	printTok(t, _m1516)
	printErrs(t, l.Errs)
	printTok(t, l.Next())
	printTok(t, l.Next())
}

func TestLexFunctionDef(t *testing.T) {
	input := `fun greet(string name) -> string {
	return "Hello".
}`
	l := New(input)
	want := []struct {
		typ token.Type
		lit string
	}{
		{token.FUN, "fun"},
		{token.IDENT, "greet"},
		{token.OPENING_PAREN, "("},
		{token.STRINGKW, "string"},
		{token.IDENT, "name"},
		{token.CLOSING_PAREN, ")"},
		{token.ARROW, "->"},
		{token.STRINGKW, "string"},
		{token.OPENING_CURLY, "{"},
		{token.NEWLINE, "\\n"},
		{token.RETURN, "return"},
		{token.STRING, "Hello"},
		{token.DOT, "."},
		{token.NEWLINE, "\\n"},
		{token.CLOSING_CURLY, "}"},
		{token.EOF, "<<<EOF>>>"},
	}

	got := []token.Token{}
	for {
		tok := l.Next()
		got = append(got, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	i := 0
	for {
		if i == len(want) || i == len(got) {
			fmt.Println("end: ", len(want), len(got))
			break
		}
		fmt.Printf("#%d: ", i)
		check1(t, got[i], want[i].lit, want[i].typ)
		i++
	}
	/*
		for _, v := range got {
			printTok(t, v)
		}*/
	printErrs(t, l.Errs)
}

func TestLexDatatype(t *testing.T) {
	input := `datatype User {
	string name
	int age
	string city
}
`

	l := New(input)
	want := []struct {
		typ token.Type
		lit string
	}{
		{token.DATATYPE, "datatype"},
		{token.IDENT, "User"},
		{token.OPENING_CURLY, "{"},
		{token.NEWLINE, "\\n"},
		{token.STRINGKW, "string"},
		{token.IDENT, "name"},
		{token.NEWLINE, "\\n"},
		{token.INTKW, "int"},
		{token.IDENT, "age"},
		{token.NEWLINE, "\\n"},
		{token.STRINGKW, "string"},
		{token.IDENT, "city"},
		{token.NEWLINE, "\\n"},
		{token.CLOSING_CURLY, "}"},
		{token.NEWLINE, "\\n"},
	}
	got := []token.Token{}
	for {
		tok := l.Next()
		got = append(got, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	i := 0
	for {
		if i == len(want) || i == len(got) {
			fmt.Println("end: ", len(want), len(got))
			break
		}
		fmt.Printf("#%d: ", i)
		check1(t, got[i], want[i].lit, want[i].typ)
		i++
	}
}

func TestNewTokens1(t *testing.T) {
	input := "Stdout::print().(and)(or)(not)-+*/-,'[]listof"
	l := New(input)
	want := []struct {
		typ token.Type
		lit string
	}{
		{token.IDENT, "Stdout"},
		{token.DOUBLE_COLON, "::"},
		{token.IDENT, "print"},
		{token.OPENING_PAREN, "("},
		{token.CLOSING_PAREN, ")"},
		{token.DOT, "."},
		{token.OPENING_PAREN, "("},
		{token.AND, "and"},
		{token.CLOSING_PAREN, ")"},
		{token.OPENING_PAREN, "("},
		{token.OR, "or"},
		{token.CLOSING_PAREN, ")"},
		{token.OPENING_PAREN, "("},
		{token.NOT, "not"},
		{token.CLOSING_PAREN, ")"},
		{token.MINUS, "-"},
		{token.ADD, "+"},
		{token.MUL, "*"},
		{token.DIV, "/"},
		{token.MINUS, "-"},
		{token.COMMA, ","},
		{token.SINGLE_QUOTE, "'"},
		{token.OPENING_SQUARE_BRACKET, "["},
		{token.CLOSING_SQUARE_BRACKET, "]"},
		{token.LISTOF, "listof"},
	}
	got := []token.Token{}
	for {
		tok := l.Next()
		got = append(got, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	i := 0
	for {
		if i == len(want) || i == len(got) {
			fmt.Println("end: ", len(want), len(got))
			break
		}
		fmt.Printf("#%d: ", i)
		check1(t, got[i], want[i].lit, want[i].typ)
		i++
	}
}

func TestLexSimplePrefExpr(t *testing.T) {
	input := `(+ 2 2)`
	l := New(input)
	for {
		tok := l.Next()
		if tok.Type == token.EOF {
			break
		}
		printTok(t, tok)
	}
	printErrs(t, l.Errs)
}
