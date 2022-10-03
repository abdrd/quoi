package parser

import (
	"fmt"
	"os"
	"quoi/ast"
	"quoi/lexer"
	"quoi/token"
	"strings"
)

type ErrCode int

type Err struct {
	ErrCode      ErrCode
	Msg          string
	Column, Line uint
}

const (
	ErrInvalidInteger ErrCode = iota // unable to atoi
	ErrInvalidBoolean
	ErrUnexpectedToken
	ErrNoValue // when we parse a variable's value but it turns out to be nil (no value)
	ErrMissingOperator
	ErrUnfinishedStatement // forgot dot
	ErrUnexpectedEOF
	ErrMissingNewline
	ErrInvalidTokenForDatatypeField
)

type Parser struct {
	tokens      []token.Token
	ptr         uint
	tok         token.Token // current token pointed to, by ptr
	lexerErrors []lexer.Err
	Errs        []Err
}

func New(l *lexer.Lexer) *Parser {
	toks := []token.Token{}
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

func (p *Parser) errorf(errCode ErrCode, line, col uint, formatMsg string, elems ...interface{}) {
	p.Errs = append(p.Errs, Err{
		ErrCode: errCode,
		Msg:     fmt.Sprintf(formatMsg, elems...),
		Column:  col,
		Line:    line,
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

/*
func (p *Parser) peekN(ahead uint) token.Token {
	tokensLen := uint(len(p.tokens))
	ok := p.ptr+ahead < tokensLen
	if ok {
		return p.tokens[p.ptr+ahead]
	}
	return p.tokens[tokensLen-1]
}
*/

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
	// parse variable declarations with primitive types
	case token.STRINGKW, token.INTKW, token.BOOLKW:
		if stmt := p.parseVariableDecl(); stmt != nil {
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
	case token.OPENING_PAREN:
		if stmt := p.parseOperator(); stmt != nil {
			return stmt
		}
	case token.EOF:
		break
	default:
		p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token '%s'", p.tok.Literal)
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

func isAcceptableTokenForParseExpr(typ token.Type) bool {
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
	}
	return nil
}

func (p *Parser) parseVariableDecl() *ast.VariableDeclaration {
	v := &ast.VariableDeclaration{Tok: p.tok}
	p.movews() // move to ws
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(ErrUnexpectedToken, peek.Line, peek.Col, "unexpected token: expected an identifier, but got '%s'", peek.Type)
		p.move()
		return nil
	}
	v.Ident = p.parseIdentifier()
	if p.tok.Type == token.WHITESPACE {
		p.move()
	}
	if p.tok.Type != token.EQUAL {
		p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token: expected an equal sign, but got '%s'", p.tok.Type)
		return nil
	}
	p.movews()
	line, col := p.peek().Line, p.peek().Col
	if !(isAcceptableTokenForParseExpr(p.peek().Type)) {
		if p.peek().Type == token.DOT {
			goto noval
		}
		p.errorf(ErrUnexpectedToken, line, col, "unexpected token '%s' as value in variable declaration", p.peek().Literal)
		p.skip(token.DOT)
		return nil
	}
noval:
	v.Value = p.parseExpr()
	if v.Value == nil {
		p.errorf(ErrNoValue, line, col, "no value set to variable")
		p.skip(token.DOT)
		return nil
	}
	p.movecws()
	if p.tok.Type != token.DOT {
		p.errorf(ErrUnfinishedStatement, p.tok.Line, p.tok.Col, "unexpected token: need a dot at the end of a statement")
		return nil
	}
	p.move() // skip dot
	return v
}

func (p *Parser) parseReassignmentStatement(identTok token.Token) *ast.ReassignmentStatement {
	r := &ast.ReassignmentStatement{Tok: identTok, Ident: &ast.Identifier{Tok: identTok}}
	p.movews()
	if eqOk, peek := p.expect(token.EQUAL), p.peek(); !(eqOk) {
		p.errorf(ErrUnexpectedToken, peek.Line, peek.Col, "unexpected token: expected an equal sign, got '%s'", peek.Type)
		p.move()
		return nil
	}
	p.movews()
	line, col := p.peek().Line, p.peek().Col
	if !(isAcceptableTokenForParseExpr(p.peek().Type)) {
		p.errorf(ErrUnexpectedToken, line, col, "unexpected token '%s' as new value in reassignment statement", p.peek().Literal)
		p.skip(token.DOT)
		return nil
	}
	r.NewValue = p.parseExpr()
	if r.NewValue == nil {
		p.errorf(ErrNoValue, line, col, "no value in reassignment")
		p.skip(token.DOT)
		return nil
	}
	if p.tok.Type != token.DOT {
		p.errorf(ErrUnfinishedStatement, p.tok.Line, p.tok.Col, "unexpected token: need a dot at the end of a statement")
		return nil
	}
	p.move()
	return r
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	b := &ast.BlockStatement{Tok: p.tok}
	p.movews()
	for p.tok.Type != token.END {
		if p.tok.Type == token.EOF {
			p.errorf(ErrUnexpectedEOF, p.tok.Line, p.tok.Col, "unexpected end-of-file: unclosed block statement")
			p.move()
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
	if !(isAcceptableTokenForParseExpr(p.peek().Type)) {
		if p.peek().Type == token.DOT {
			// first time using goto :D
			goto noval
		}
		p.errorf(ErrUnexpectedToken, p.peek().Line, p.peek().Col, "unexpected token '%s' as return value in return statement", p.peek().Literal)
		p.skip(token.DOT)
		return nil
	}
noval:
	r.Expr = p.parseExpr()
	if r.Expr == nil {
		p.errorf(ErrNoValue, line, col, "return statement with no value")
		p.move()
		return nil
	}
	if p.tok.Type != token.DOT {
		p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token: need a dot at the end of a return statement")
		return nil
	}
	p.move()
	return r
}

func (p *Parser) parseLoopStatement() *ast.LoopStatement {
	l := &ast.LoopStatement{Tok: p.tok}
	p.movews()
	line, col := p.peek().Line, p.peek().Col
	if !(isAcceptableTokenForParseExpr(p.peek().Type)) {
		if p.peek().Type == token.OPENING_CURLY {
			goto nocond
		}
		p.errorf(ErrUnexpectedToken, line, col, "unexpected token '%s' in loop statement. loop statement condition must be an expression", p.peek().Literal)
		p.skip(token.CLOSING_CURLY)
		return nil
	}
nocond:
	l.Cond = p.parseExpr()
	if l.Cond == nil {
		p.errorf(ErrNoValue, line, col, "missing condition in loop statement")
		p.skip(token.CLOSING_CURLY)
		return nil
	}
	p.movecws()
	if p.tok.Type != token.OPENING_CURLY {
		p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token: expected an opening curly brace, got '%s'", p.tok.Type)
		p.move()
		return nil
	}
	p.movews()
	for {
		if p.tok.Type == token.EOF {
			p.errorf(ErrUnexpectedEOF, p.tok.Line, p.tok.Col, "unexpected end-of-file: unclosed loop statement")
			return nil
		}
		if p.tok.Type == token.CLOSING_CURLY {
			break
		}
		if stmt := p.parseStatement(); stmt != nil {
			l.Stmts = append(l.Stmts, stmt)
		}
	}
	if p.tok.Type != token.CLOSING_CURLY {
		p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token: expected a closing curly brace, got '%s'", p.tok.Type)
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
			p.errorf(ErrUnexpectedToken, peek.Line, peek.Col, "missing identifier: expected an identifier in datatype field")
			p.move()
			return nil
		}
		f.Ident = p.parseIdentifier()
		if p.tok.Type != token.WHITESPACE {
			p.errorf(ErrMissingNewline, p.tok.Line, p.tok.Col, "missing newline after datatype field")
			return nil
		}
		// no newline in whitespace
		if !(strings.Contains(p.tok.Literal, "\n")) {
			p.errorf(ErrMissingNewline, p.tok.Line, p.tok.Col, "missing newline at the end of datatype field")
			p.move()
			return nil
		}
	default:
		p.errorf(ErrInvalidTokenForDatatypeField, p.tok.Line, p.tok.Col, "invalid token '%s' for datatype field", p.tok.Literal)
		return nil
	}
	p.move()
	return f
}

func (p *Parser) parseDatatypeDeclaration() *ast.DatatypeDeclaration {
	d := &ast.DatatypeDeclaration{Tok: p.tok}
	p.movews()
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(ErrNoValue, peek.Line, peek.Col, "datatype without a name")
		p.movews()
		if p.peek().Type == token.OPENING_CURLY {
			p.move()
		}
		if p.peek().Type == token.CLOSING_CURLY {
			p.move()
		}
		p.move()
		return nil
	}
	line, col := p.tok.Line, p.tok.Col
	name := p.parseIdentifier()
	if name == nil {
		p.errorf(ErrNoValue, line, col, "expected a name for the datatype declaration")
		return nil
	}
	d.Name = name
	if p.tok.Type == token.WHITESPACE {
		p.move()
	}
	if p.tok.Type != token.OPENING_CURLY {
		p.errorf(ErrNoValue, p.tok.Line, p.tok.Col, "missing opening curly brace in datatype declaration")
		p.move()
		return nil
	}
	p.move() // skip {
	for {
		if p.tok.Type == token.EOF {
			p.errorf(ErrUnexpectedEOF, p.tok.Line, p.tok.Col, "unexpected end-of-file: expected a closing curly brace at the end of datatype declaration")
			p.move()
			return nil
		}
		if p.tok.Type == token.CLOSING_CURLY {
			break
		}
		if p.tok.Type == token.WHITESPACE {
			p.move()
		}
		field := p.parseDatatypeField()
		if field == nil {
			// get out of the block
			for {
				if p.tok.Type == token.EOF || p.tok.Type == token.CLOSING_CURLY {
					break
				}
				p.move()
			}
			if p.tok.Type == token.CLOSING_CURLY {
				p.move()
			}
			return nil
		}
		d.Fields = append(d.Fields, field)
	}
	if p.tok.Type == token.WHITESPACE {
		p.move() // skip the last whitespace (containing \n)
	}
	if p.tok.Type != token.CLOSING_CURLY {
		p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token: expected a closing curly brace at the end of datatype declaration")
		return nil
	}
	p.move()
	return d
}

func (p *Parser) parseOperator() *ast.PrefixExpr {
	// current token is token.OPENING_PAREN
	pe := &ast.PrefixExpr{}
	p.movews()
	p.move()
	if p.tok.Type == token.EOF {
		p.errorf(ErrUnexpectedEOF, p.tok.Line, p.tok.Col, "unexpected end-of-file: missing ')' in prefix expression")
		return nil
	}
	if p.tok.Type == token.CLOSING_PAREN {
		p.errorf(ErrMissingOperator, p.tok.Line, p.tok.Col, "missing operator in prefix expression")
		p.move()
		return nil
	}
	pe.Tok = p.tok
	p.move()
	for p.tok.Type != token.CLOSING_PAREN && p.tok.Type != token.EOF {
		p.movecws() // if current token is whitespace: move.
		if p.tok.Type == token.CLOSING_PAREN {
			break
		}
		if !(isAcceptableTokenForParseExpr(p.tok.Type)) {
			p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token '%s' in '%s' expression", p.tok.Literal, pe.Tok.Literal)
			p.skip(token.CLOSING_PAREN)
			return nil
		}
		var arg ast.Expr
		switch p.tok.Type {
		case token.STRING:
			arg = p.parseStringLiteral()
		case token.INT:
			arg = p.parseIntLiteral()
		case token.BOOL:
			arg = p.parseBoolLiteral()
		case token.IDENT:
			arg = p.parseIdentifier()
		case token.OPENING_PAREN:
			arg = p.parseOperator()
		}
		if arg != nil {
			pe.Args = append(pe.Args, arg)
		}
	}
	if p.tok.Type == token.EOF {
		p.errorf(ErrUnexpectedEOF, p.tok.Line, p.tok.Col, "unexpected end-of-file: missing ')' at the end of prefix expression")
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
	if p.tok.Type != token.OPENING_PAREN {
		p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token '%s' in function call expression. expected a '('", p.tok.Literal)
		p.skip(token.CLOSING_PAREN)
		return nil
	}
	p.movews()
	if p.peek().Type == token.CLOSING_PAREN {
		p.dmove()
		return fc
	}
	if !(isAcceptableTokenForParseExpr(p.peek().Type)) {
		p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token '%s' as argument value to function call", p.peek().Literal)
		p.skip(token.CLOSING_PAREN)
		return nil
	}
	arg := p.parseExpr()
	if arg != nil {
		fc.Args = append(fc.Args, arg)
	}
	p.movecws()
	// one argument
	if p.tok.Type == token.CLOSING_PAREN {
		goto end
	}
	for {
		p.movecws()
		if p.tok.Type == token.CLOSING_PAREN {
			goto end
		}
		if p.tok.Type == token.EOF {
			p.errorf(ErrUnexpectedEOF, p.tok.Line, p.tok.Col, "unexpected end-of-file: unclosed function call expression")
			return nil
		}
		if p.tok.Type != token.COMMA {
			p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token: expected comma between arguments")
			p.skip(token.CLOSING_PAREN)
			return nil
		}
		p.move() // skip comma
		if !(isAcceptableTokenForParseExpr(p.peek().Type)) {
			p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token '%s' as argument value to function call", p.peek().Literal)
			p.skip(token.CLOSING_PAREN)
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
		p.errorf(ErrUnexpectedToken, peek.Line, peek.Col, "unexpected token '%s' when calling a function from namespace '%s'. expected `::`", peek.Literal, fcfn.Namespace.Identifier.String())
		p.move()
		return nil
	}
	p.movews()
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(ErrUnexpectedToken, peek.Line, peek.Col, "unexpected token '%s'. expected a function name", peek.Literal)
		p.move()
		return nil
	}
	if fn := p.parseFunctionCall(p.tok); fn != nil {
		fcfn.Function = fn
	}
	return fcfn
}
