package parser

import (
	"fmt"
	"os"
	"quoi/ast"
	"quoi/lexer"
	"quoi/token"
	"strings"
)

type Err struct {
	Msg          string
	Column, Line uint
}

// fmt.Sprintf
var spf = fmt.Sprintf

func newErr(line, col uint, formatMsg string, elems ...interface{}) Err {
	return Err{Msg: spf(formatMsg, elems...), Column: col, Line: line}
}

type Parser struct {
	tokens      []token.Token
	ptr         uint
	tok         token.Token // current token pointed to, by ptr
	lexerErrors []lexer.Err
	Errs        []Err
}

func New(l *lexer.Lexer) *Parser {
	toks := []token.Token{}
	// TODO make this memory efficient
	for {
		t := l.Next()
		toks = append(toks, t)
		if t.Type == token.EOF {
			break
		}
	}
	if len(toks) == 0 {
		panic("lexer.New: error: len(tokens) is zero (0)")
	}
	p := &Parser{}
	p.tokens = toks
	p.lexerErrors = l.Errs
	p.ptr = 0
	p.tok = p.tokens[p.ptr]
	return p
}

func (p *Parser) errorf(line, col uint, formatMsg string, elems ...interface{}) {
	p.Errs = append(p.Errs, Err{
		Msg:    fmt.Sprintf(formatMsg, elems...),
		Column: col,
		Line:   line,
	})
}

// set p.tok to the next token
func (p *Parser) move() {
	outOfBounds := len(p.tokens)-1 == int(p.ptr)
	if outOfBounds {
		return
	}
	p.ptr++
	p.tok = p.tokens[p.ptr]
}

// double move
func (p *Parser) dmove() {
	p.move()
	p.move()
}

// if peek token is WHITESPACE, then move; else, don't do anything
func (p *Parser) movews() {
	if p.peek().Type == token.WHITESPACE {
		p.move()
	}
}

// if current token is WHITESPACE, then move; else, don't do anything
func (p *Parser) movecws() {
	if p.tok.Type == token.WHITESPACE {
		p.move()
	}
}

func (p *Parser) peek() token.Token {
	hasNextToken := len(p.tokens)-2 >= int(p.ptr)
	if hasNextToken {
		return p.tokens[p.ptr+1]
	}
	return p.tok
}

// check if peek token's type is t.
// if it is, then move.
func (p *Parser) expect(t token.Type) bool {
	if peek := p.peek(); peek.Type != t {
		return false
	}
	p.move()
	return true
}

// go until 'toTheEndOf', and move once more; and return.
//
// useful in situations like coming across an erroneous expression, or statement;
// and wanting to ignore the whole statement to prevent giving redundant error messages.
func (p *Parser) skip(toTheEndOf token.Type) {
	for {
		if p.tok.Type == token.EOF || p.tok.Type == toTheEndOf {
			p.move()
			break
		}
		p.move()
	}
}

// skip to the next statement (not expr statement)
func (p *Parser) skip2() {
	kwx := []token.Type{
		token.INTKW, token.STRINGKW, token.BOOLKW, token.DATATYPE, token.FUN, token.BLOCK, token.END, token.IF,
		token.ELSEIF, token.ELSE, token.LOOP, token.RETURN, token.LISTOF,
	}
	in := func(typ token.Type, l []token.Type) bool {
		for _, v := range l {
			if v == typ {
				return true
			}
		}
		return false
	}
	for {
		if in(p.tok.Type, kwx) || p.tok.Type == token.EOF {
			break
		}
		p.move()
	}
}

func isnil(v interface{}) bool {
	return v == nil
}

// append error if cond.
// skip to the next statement if cond.
// return true if cond.
func (p *Parser) errif(cond bool, err Err) bool {
	if cond {
		p.Errs = append(p.Errs, err)
		p.skip2()
	}
	return cond
}

// current token's type is not typ.
func (p *Parser) curnot(typ token.Type) bool {
	return !(p.curis(typ))
}

// current token's type is typ
func (p *Parser) curis(typ token.Type) bool {
	return p.tok.Type == typ
}

func (p *Parser) peekis(typ token.Type) bool {
	return p.peek().Type == typ
}

func (p *Parser) moveif(cond bool) {
	if cond {
		p.move()
	}
}

func (p *Parser) Parse() *ast.Program {
	if len(p.lexerErrors) > 0 {
		for _, e := range p.lexerErrors {
			fmt.Printf("[!] error code:%d: line:col(%d:%d) %s\n", e.ErrCode, e.Line, e.Column, e.Msg)
		}
		os.Exit(1)
	}
	program := &ast.Program{}
loop:
	for {
		if p.tok.Type == token.EOF {
			break loop
		}
		if stmt := p.parseStatement(); stmt != nil {
			program.PushStmt(stmt)
		}
	}
	return program
}

// > advance parser at the end

func (p *Parser) parseStatement() ast.Statement {
	// we are doing "if-stmt-is-not-nil" checks here because when calling this function in *Parser.Parse,
	// "if p.parseStatement() != nil" checks do not work. This may be because Statement is an interface,
	// and even if a pointer to a struct that implements ast.Statement is nil, *Parser.Parse thinks
	// it is not nil, because the actual pointer-to-struct type is "hidden" behind an interface
	// (ast.Statement).
	//
	// Am I guessing correct?

	switch p.tok.Type {
	case token.WHITESPACE:
		p.move()

	// string literal
	case token.STRING:
		// no need to check nil for this method, but there's no harm in doing just that
		if stmt := p.parseStringLiteral(); stmt != nil {
			return stmt
		}
	case token.INT:
		if stmt := p.parseIntLiteral(); stmt != nil {
			return stmt
		}
	case token.BOOL:
		if stmt := p.parseBoolLiteral(); stmt != nil {
			return stmt
		}
	case token.OPENING_SQUARE_BRACKET:
		if stmt := p.parseListLiteral(); stmt != nil {
			return stmt
		}
	case token.OPENING_PAREN:
		if stmt := p.parseOperator(); stmt != nil {
			return stmt
		}
	// parse variable declarations with primitive types
	case token.STRINGKW, token.INTKW, token.BOOLKW:
		if stmt := p.parseVariableDecl(); stmt != nil {
			return stmt
		}
	case token.LISTOF:
		if stmt := p.parseListVariableDecl(); stmt != nil {
			return stmt
		}
	case token.IDENT:
		identTok := p.tok
		p.movews() // move to peek if it is ws
		thisIsAReassignment := p.peek().Type == token.EQUAL
		if thisIsAReassignment {
			if stmt := p.parseReassignmentStatement(identTok); stmt != nil {
				return stmt
			}
			break
		}
		thisIsAFunctionCall := p.peek().Type == token.OPENING_PAREN
		if thisIsAFunctionCall {
			if stmt := p.parseFunctionCall(identTok); stmt != nil {
				return stmt
			}
			break
		}
		thisIsAFunctionFromANamespace := p.peek().Type == token.DOUBLE_COLON
		if thisIsAFunctionFromANamespace {
			if stmt := p.parseFunctionCallFromNamespace(identTok); stmt != nil {
				return stmt
			}
			break
		}
		// identifier
		if stmt := p.parseIdentifier(); stmt != nil {
			return stmt
		}
	case token.BLOCK:
		if stmt := p.parseBlockStatement(); stmt != nil {
			return stmt
		}
	case token.RETURN:
		if stmt := p.parseReturnStatement(); stmt != nil {
			return stmt
		}
	case token.LOOP:
		if stmt := p.parseLoopStatement(); stmt != nil {
			return stmt
		}
	case token.DATATYPE:
		if stmt := p.parseDatatypeDeclaration(); stmt != nil {
			return stmt
		}
	case token.EOF:
		break
	default:
		p.errorf(p.tok.Line, p.tok.Col, "unexpected token '%s'", p.tok.Literal)
		tokTyp := p.tok.Type
		// skip token of same type to avoid giving repetitive error messages
		for {
			if p.tok.Type != tokTyp {
				break
			}
			p.move()
		}
	}
	return nil
}

// typ is acceptable for parse expr
func isAccExpr(typ token.Type) bool {
	// these are EXPECTED token types for parseExpr.
	// these can be expressions.
	//
	// other tokens cannot be exprs.
	acceptableTokens := []token.Type{
		token.STRING, token.INT, token.BOOL, token.IDENT, token.OPENING_PAREN,
	}
	for _, v := range acceptableTokens {
		if v == typ {
			return true
		}
	}
	return false
}

func (p *Parser) parseStringLiteral() *ast.StringLiteral {
	s := &ast.StringLiteral{Typ: p.tok.Type, Val: p.tok.Literal}
	p.move()
	return s
}

func (p *Parser) parseIntLiteral() *ast.IntLiteral {
	n := atoi(p)
	i := &ast.IntLiteral{Typ: p.tok.Type, Val: n}
	p.move()
	return i
}

func (p *Parser) parseBoolLiteral() *ast.BoolLiteral {
	b := atob(p)
	boo := &ast.BoolLiteral{Typ: p.tok.Type, Val: b}
	p.move()
	return boo
}

func (p *Parser) parseIdentifier() *ast.Identifier {
	i := &ast.Identifier{Tok: p.tok}
	p.move()
	return i
}

// decide depending on peek token
func (p *Parser) parseExpr() ast.Expr {
	peek := p.peek()
	p.move()
	switch peek.Type {
	case token.STRING:
		return p.parseStringLiteral()
	case token.INT:
		return p.parseIntLiteral()
	case token.BOOL:
		return p.parseBoolLiteral()
	case token.IDENT:
		return p.parseIdentifier()
	case token.OPENING_PAREN:
		return p.parseOperator()
	case token.OPENING_SQUARE_BRACKET:
		return p.parseListLiteral()
	}
	return nil
}

func (p *Parser) parseVariableDecl() *ast.VariableDeclaration {
	v := &ast.VariableDeclaration{Tok: p.tok}
	p.movews() // move to ws
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token: expected an identifier, but got '%s'", peek.Type)
		p.move()
		return nil
	}
	v.Ident = p.parseIdentifier()
	p.moveif(p.curis(token.WHITESPACE))
	if p.errif(p.curnot(token.EQUAL),
		newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s', expected an equal sign", p.tok.Literal)) {
		return nil
	}
	p.movews()
	line, col := p.peek().Line, p.peek().Col
	if p.peekis(token.DOT) {
		goto noval
	}
	if p.errif(!(isAccExpr(p.peek().Type)),
		newErr(line, col,
			"unexpected token '%s' as value in variable declaration", p.peek().Literal)) {
		return nil
	}
noval:
	v.Value = p.parseExpr()
	if p.errif(isnil(v.Value),
		newErr(line, col, "no value set to variable")) {
		return nil
	}
	p.movecws()
	if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
		"unexpected token: need a dot at the end of a statement")) {
		return nil
	}
	p.move() // skip dot
	return v
}

func (p *Parser) parseReassignmentStatement(identTok token.Token) *ast.ReassignmentStatement {
	r := &ast.ReassignmentStatement{Tok: identTok, Ident: &ast.Identifier{Tok: identTok}}
	p.movews()
	if eqOk, peek := p.expect(token.EQUAL), p.peek(); !(eqOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token: expected an equal sign, got '%s'", peek.Type)
		p.move()
		return nil
	}
	p.movews()
	line, col := p.peek().Line, p.peek().Col
	if p.errif(!(isAccExpr(p.peek().Type)), newErr(line, col,
		"unexpected token '%s' as new value in reassignment statement", p.peek().Literal)) {
		return nil
	}
	r.NewValue = p.parseExpr()
	if p.errif(isnil(r.NewValue), newErr(line, col, "no value in reassignment")) {
		return nil
	}
	if p.errif(p.curnot(token.DOT),
		newErr(p.tok.Line, p.tok.Col,
			fmt.Sprintf("unexpected token '%s' need a dot at the end of a statement", p.tok.Literal))) {
		return nil
	}
	p.move()
	return r
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	b := &ast.BlockStatement{Tok: p.tok}
	p.movews()
	for p.tok.Type != token.END {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: unclosed block statement")) {
			return nil
		}
		if stmt := p.parseStatement(); stmt != nil {
			b.Stmts = append(b.Stmts, stmt)
		}
	}
	p.move()
	return b
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	r := &ast.ReturnStatement{Tok: p.tok}
	line, col := p.tok.Line, p.tok.Col
	p.movews()
	if p.peekis(token.DOT) {
		// first time using goto :D
		goto noval
	}
	if p.errif(!(isAccExpr(p.peek().Type)), newErr(p.peek().Line, p.peek().Col,
		"unexpected token '%s' as return value in return statement", p.peek().Literal)) {
		return nil
	}
noval:
	r.Expr = p.parseExpr()
	if p.errif(isnil(r.Expr), newErr(line, col, "return statement with no value")) {
		return nil
	}
	if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
		"unexpected token: need a dot at the end of a return statement")) {
		return nil
	}
	p.move()
	return r
}

func (p *Parser) parseLoopStatement() *ast.LoopStatement {
	l := &ast.LoopStatement{Tok: p.tok}
	p.movews()
	line, col := p.peek().Line, p.peek().Col
	if p.peekis(token.OPENING_CURLY) {
		goto nocond
	}
	if p.errif(!(isAccExpr(p.peek().Type)), newErr(line, col,
		"unexpected token '%s' in loop statement. loop statement condition must be an expression", p.peek().Literal)) {
		return nil
	}
nocond:
	l.Cond = p.parseExpr()
	if p.errif(isnil(l.Cond), newErr(line, col, "missing condition in loop statement")) {
		return nil
	}
	p.movecws()
	if p.errif(p.curnot(token.OPENING_CURLY), newErr(p.tok.Line, p.tok.Col,
		"unexpected token: expected an opening curly brace, got '%s'", p.tok.Type)) {
		return nil
	}
	p.movews()
	for {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: unclosed loop statement")) {
			return nil
		}
		if p.curis(token.CLOSING_CURLY) {
			break
		}
		if stmt := p.parseStatement(); stmt != nil {
			l.Stmts = append(l.Stmts, stmt)
		}
	}
	if p.errif(p.curnot(token.CLOSING_CURLY), newErr(p.tok.Line, p.tok.Col,
		"unexpected token: expected a closing curly brace, got '%s'", p.tok.Type)) {
		return nil
	}
	p.move()
	return l
}

func (p *Parser) parseDatatypeField() *ast.DatatypeField {
	// require newline at the end of every field
	f := &ast.DatatypeField{Tok: p.tok}
	switch p.tok.Type {
	case token.INTKW, token.STRINGKW, token.BOOLKW, token.IDENT:
		p.movews()
		if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
			p.errorf(peek.Line, peek.Col, "missing identifier: expected an identifier in datatype field")
			p.move()
			return nil
		}
		f.Ident = p.parseIdentifier()
		if p.errif(p.curnot(token.WHITESPACE), newErr(p.tok.Line, p.tok.Col,
			"missing newline after datatype field")) {
			return nil
		}
		// no newline in whitespace
		if p.errif(!(strings.Contains(p.tok.Literal, "\n")), newErr(p.tok.Line, p.tok.Col,
			"missing newline at the end of datatype field")) {
			return nil
		}
	default:
		p.errorf(p.tok.Line, p.tok.Col, "invalid token '%s' for datatype field", p.tok.Literal)
		return nil
	}
	p.move()
	return f
}

func (p *Parser) parseDatatypeDeclaration() *ast.DatatypeDeclaration {
	d := &ast.DatatypeDeclaration{Tok: p.tok}
	p.movews()
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(peek.Line, peek.Col, "datatype without a name")
		p.movews()
		p.moveif(p.peekis(token.OPENING_CURLY))
		p.moveif(p.peekis(token.CLOSING_CURLY))
		p.move()
		return nil
	}
	line, col := p.tok.Line, p.tok.Col
	name := p.parseIdentifier()
	if p.errif(isnil(name), newErr(line, col, "expected a name for the datatype declaration")) {
		return nil
	}
	d.Name = name
	p.movecws()
	if p.errif(p.curnot(token.OPENING_CURLY), newErr(p.tok.Line, p.tok.Col,
		"missing opening curly brace in datatype declaration")) {
		return nil
	}
	p.move() // skip {
	for {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: expected a closing curly brace at the end of datatype declaration")) {
			return nil
		}
		if p.curis(token.CLOSING_CURLY) {
			break
		}
		p.movecws()
		field := p.parseDatatypeField()
		if isnil(field) {
			p.skip2()
			return nil
		}
		d.Fields = append(d.Fields, field)
	}
	p.movecws() // skip the last whitespace (containing \n)
	if p.errif(p.curnot(token.CLOSING_CURLY), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s'. expected a closing curly brace at the end of datatype declaration",
		p.tok.Literal)) {
		return nil
	}
	p.move()
	return d
}

// TODO BUGGY vvv

func (p *Parser) parseOperator() *ast.PrefixExpr {
	// current token is token.OPENING_PAREN
	pe := &ast.PrefixExpr{}
	p.movews()
	p.move()
	if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
		"unexpected end-of-file: missing ')' in prefix expression")) {
		return nil
	}
	if p.errif(p.curis(token.CLOSING_PAREN), newErr(p.tok.Line, p.tok.Col,
		"missing operator in prefix expression")) {
		return nil
	}
	pe.Tok = p.tok
	p.move()
	for p.tok.Type != token.CLOSING_PAREN && p.tok.Type != token.EOF {
		p.movecws() // if current token is whitespace: move.
		if p.curis(token.CLOSING_PAREN) {
			break
		}
		if p.errif(!(isAccExpr(p.tok.Type)), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s' in '%s' expression",
			p.tok.Literal, pe.Tok.Literal)) {
			return nil
		}
		// call parseStatement, instead of parseExpr.
		// because we are dealing with p.tok. (parseExpr looks to p.peek())
		// I am lazy to restructure the whole method.
		if arg := p.parseStatement(); arg != nil {
			pe.Args = append(pe.Args, arg)
		}
	}
	if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
		"unexpected end-of-file: missing ')' at the end of prefix expression")) {
		return nil
	}
	// no need for 'p.tok.Type == token.CLOSING_PAREN'
	p.move() // skip )
	return pe
}

func (p *Parser) parseFunctionCall(ident token.Token) *ast.FunctionCall {
	fc := &ast.FunctionCall{Tok: ident, Ident: &ast.Identifier{Tok: ident}}
	// peek token is (
	p.move()
	if p.errif(p.curnot(token.OPENING_PAREN), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' in function call expression. expected a '('",
		p.tok.Literal)) {
		return nil
	}
	p.movews()
	if p.peekis(token.CLOSING_PAREN) {
		p.dmove()
		return fc
	}
	if p.errif(!(isAccExpr(p.peek().Type)), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' as argument value to function call",
		p.peek().Literal)) {
		return nil
	}
	arg := p.parseExpr()
	if arg != nil {
		fc.Args = append(fc.Args, arg)
	}
	p.movecws()
	// one argument
	if p.curis(token.CLOSING_PAREN) {
		goto end
	}
	for {
		p.movecws()
		if p.curis(token.CLOSING_PAREN) {
			goto end
		}
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: unclosed function call expression")) {
			return nil
		}
		if p.errif(p.curnot(token.COMMA), newErr(p.tok.Line, p.tok.Col,
			"unexpected token: expected comma between arguments")) {
			return nil
		}
		p.move() // skip comma
		if p.errif(!(isAccExpr(p.peek().Type)), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s' as argument value to function call",
			p.peek().Literal)) {
			return nil
		}
		if arg := p.parseExpr(); arg != nil {
			fc.Args = append(fc.Args, arg)
		}
	}
end:
	p.move() // skip )
	return fc
}

func (p *Parser) parseFunctionCallFromNamespace(namespaceTok token.Token) *ast.FunctionCallFromNamespace {
	fcfn := &ast.FunctionCallFromNamespace{Namespace: &ast.Namespace{Tok: namespaceTok, Identifier: &ast.Identifier{Tok: namespaceTok}}}
	p.movews()
	if dColonOk, peek := p.expect(token.DOUBLE_COLON), p.peek(); !(dColonOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token '%s' when calling a function from namespace '%s'. expected `::`", peek.Literal, fcfn.Namespace.Identifier.String())
		p.move()
		return nil
	}
	p.movews()
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token '%s'. expected a function name", peek.Literal)
		p.move()
		return nil
	}
	if fn := p.parseFunctionCall(p.tok); fn != nil {
		fcfn.Function = fn
	}
	return fcfn
}
func (p *Parser) parseListVariableDecl() *ast.ListVariableDecl {
	// current token is 'listof'
	l := &ast.ListVariableDecl{Tok: p.tok}
	var canBeATypeForList = func(tok token.Type) bool {
		for _, v := range []token.Type{
			token.INTKW, token.STRINGKW, token.BOOLKW, token.IDENT,
			token.LISTOF, // multidimensional lists
		} {
			if tok == v {
				return true
			}
		}
		return false
	}
	p.movews()
	if peek := p.peek(); !(canBeATypeForList(peek.Type)) {
		p.errorf(peek.Line, peek.Col, "unexpected token '%s' as type for list", peek.Literal)
		p.skip(token.CLOSING_SQUARE_BRACKET)
		return nil
	}
	p.move()
	l.Typ = p.tok
	p.movews()
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token '%s'. expected an identifier in list declaration", peek.Literal)
		p.skip(token.CLOSING_SQUARE_BRACKET)
		return nil
	}
	l.Name = p.parseIdentifier()
	p.movecws()
	if p.errif(p.curnot(token.EQUAL), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s'. expected an equal sign",
		p.tok.Literal)) {
		return nil
	}
	p.movews()
	if openSqBrOk, peek := p.expect(token.OPENING_SQUARE_BRACKET), p.peek(); !(openSqBrOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token '%s', where a '[' was expected", peek.Literal)
		p.skip(token.CLOSING_SQUARE_BRACKET)
		return nil
	}
	list := p.parseListLiteral()
	if isnil(list) {
		p.skip2()
		return nil
	}
	l.List = list
	if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s', expected a dot. unfinished list declaration statement",
		p.tok.Literal)) {
		return nil
	}
	p.move() // skip .
	return l
}

func (p *Parser) parseListLiteral() *ast.ListLiteral {
	// current token is '['
	l := &ast.ListLiteral{Tok: p.tok}
	p.movews()
	// no elems
	if p.peekis(token.CLOSING_SQUARE_BRACKET) {
		p.dmove()
		return l
	}
	if peek := p.peek(); p.errif(!(isAccExpr(peek.Type)), newErr(peek.Line, peek.Col,
		"unexpected token '%s' as list element",
		peek.Literal)) {
		return nil
	}
	if firstEl := p.parseExpr(); firstEl != nil {
		l.Elems = append(l.Elems, firstEl)
	}
	p.movecws()
	for {
		p.movecws()
		if p.curis(token.CLOSING_SQUARE_BRACKET) {
			goto end
		}
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: unclosed list literal")) {
			return nil
		}
		if p.errif(p.curnot(token.COMMA), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s'. missing comma in list literal",
			p.tok.Literal)) {
			return nil
		}
		p.move() // skip comma
		if peek := p.peek(); p.errif(!(isAccExpr(peek.Type)), newErr(peek.Line, peek.Col,
			"unexpected token '%s' as list element",
			peek.Literal)) {
			return nil
		}
		if el := p.parseExpr(); el != nil {
			l.Elems = append(l.Elems, el)
		}
	}
end:
	p.move() // skip ]
	return l
}
