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
	ErrWrongType           // for example, when you try to assign a string literal to a variable declared as of type int.
	ErrNoValue             // when we parse a variable's value but it turns out to be nil (no value)
	ErrUnfinishedStatement // forgot dot
	ErrUnexpectedEOF
	ErrUnknownOperator
	ErrLonelyExpr // expressions that are not tied to any variable
	ErrMissingNewline
	ErrInvalidTokenForDatatypeField
	ErrUnclosedPrefixExpr // forgot )
	ErrInvalidExpr        // when passing a non-expression to a prefix expr as an argument (e.g. (+ 2 fun), or (* 5 ]))
	ErrUnwantedArgument
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
	// the dot may be here because ErrWrongType was appended to parser in parseVariableDecl.
	// if we double move in parseVariableDecl, assuming there is a dot at the end, we are doing a wrong thing.
	// there may not be a dot at the end when there is ErrWrongType in parseVariableDecl, and we can
	// skip over an important keyword like int, string, etc.
	// so when we come across a dot here, we know that we must skip over this dot HERE.
	//
	// if you don't understand what I am saying above, don't worry. I think I forgot English.
	// I should probably go outside, and walk for a bit.
	case token.DOT:
		p.move()
	case token.IDENT:
		identTok := p.tok
		p.movews()
		if p.peek().Type == token.EQUAL {
			// reassignment
			if stmt := p.parseReassignmentStatement(identTok); stmt != nil {
				return stmt
			}
		}
		p.errorf(ErrLonelyExpr, p.tok.Line, p.tok.Col, "not used variable: value of this variable '%s' is not used", identTok.Literal)
		p.move()
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
		p.movews()
		p.move()
		switch p.tok.Type {
		case token.ADD, token.MINUS, token.DIV, token.MUL, token.AND, token.OR,
			token.LT, token.GT, token.LTE, token.GTE:
			if stmt := p.parseTwoArgsPrefixExpr(); stmt != nil {
				return stmt
			}
		case token.NOT:
			if stmt := p.parseNotExpr(); stmt != nil {
				return stmt
			}
		default:
			op := p.tok.Literal
			p.movews()
			p.move()
			if p.tok.Type != token.CLOSING_PAREN {
				p.errorf(ErrUnknownOperator, p.tok.Line, p.tok.Col, "unknown operator '%s'", op)
				for {
					if p.tok.Type == token.EOF {
						break
					}
					if p.tok.Type == token.CLOSING_PAREN {
						p.move()
						return nil
					}
					p.move()
				}
			}
		}
	default:
		panic("parseStatement: error: NOT IMPLEMENTED: " + p.tok.Type.String())
	}
	return nil
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
		return p.parseStatement()
	case token.DOT, token.EOF:
		p.move()
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
	// > right now, we only accept primitive literals (string, int, bool)
	// > in the future we will have other expressions (prefix exprs., function calls, lists, ...).
	switch v.Tok.Type {
	case token.STRINGKW:
		if stringLitOk, peek := p.expect(token.STRING), p.peek(); !(stringLitOk) {
			p.errorf(ErrWrongType, peek.Line, peek.Col, "wrong type for variable: expected a value of type string, but got '%s'", peek.Type)
			p.move()
			return nil
		}
		line, col := p.tok.Line, p.tok.Col
		v.Value = p.parseStringLiteral()
		if v.Value == nil {
			p.errorf(ErrNoValue, line, col, "variable with no value")
			return nil
		}
	case token.INTKW:
		if intLitOk, peek := p.expect(token.INT), p.peek(); !(intLitOk) {
			p.errorf(ErrWrongType, peek.Line, peek.Col, "wrong type for variable: expected a value of type integer, but got '%s'", peek.Type)
			p.move()
			return nil
		}
		line, col := p.tok.Line, p.tok.Col
		v.Value = p.parseIntLiteral()
		if v.Value == nil {
			p.errorf(ErrNoValue, line, col, "variable with no value")
			return nil
		}
	case token.BOOLKW:
		if boolLitOk, peek := p.expect(token.BOOL), p.peek(); !(boolLitOk) {
			p.errorf(ErrWrongType, peek.Line, peek.Col, "wrong type for variable: expected a value of type boolean, but got '%s'", peek.Type)
			p.move()
			return nil
		}
		line, col := p.tok.Line, p.tok.Col
		v.Value = p.parseBoolLiteral()
		if v.Value == nil {
			p.errorf(ErrNoValue, line, col, "variable with no value")
			return nil
		}
	default:
		panic("parseVariableDecl: error: NOT IMPLEMENTED: " + v.Tok.Type.String())
	}
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
	r.NewValue = p.parseExpr()
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
	r.Expr = p.parseExpr()
	if r.Expr == nil {
		p.errorf(ErrNoValue, line, col, "return statement with no value")
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
	line, col := p.tok.Line, p.tok.Col
	cond := p.parseExpr()
	if cond == nil {
		p.errorf(ErrNoValue, line, col, "missing condition in loop statement")
		p.move() // skip {
		if p.tok.Type == token.CLOSING_CURLY {
			p.move() // skip }
		}
		return nil
	}
	l.Cond = cond
	if p.tok.Type == token.WHITESPACE {
		p.move()
	}
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

// go until ), or EOF.
// skip ')' if found.
func getOutOfPrefixExpr(p *Parser) {
	for {
		if p.tok.Type == token.EOF {
			break
		}
		if p.tok.Type == token.CLOSING_PAREN {
			p.move()
			break
		}
		p.move()
	}
}

// TODO better error messages for these two methods

func (p *Parser) parseTwoArgsPrefixExpr() *ast.PrefixExpr {
	pe := &ast.PrefixExpr{Tok: p.tok}
	p.movews()
	for i := 0; i < 2; i++ {
		argLit := p.peek().Literal
		if arg := p.parseExpr(); arg != nil {
			pe.Args = append(pe.Args, arg)
		} else {
			p.errorf(ErrInvalidExpr, p.tok.Line, p.tok.Col, "invalid expression passed to '%s': '%s'", token.PrefixExprName(pe.Tok.Type), argLit)
			getOutOfPrefixExpr(p)
			return nil
		}
	}
	line, col := p.tok.Line, p.tok.Col
	p.movecws()
	if p.tok.Type != token.CLOSING_PAREN {
		p.errorf(ErrUnclosedPrefixExpr, line, col, "unclosed '%s' expression", token.PrefixExprName(pe.Tok.Type))
		getOutOfPrefixExpr(p)
		return nil
	}
	p.move() // skip )
	return pe
}

func (p *Parser) parseNotExpr() *ast.PrefixExpr {
	pe := &ast.PrefixExpr{Tok: p.tok}
	p.movews()
	argLit := p.peek().Literal
	if arg := p.parseExpr(); arg != nil {
		pe.Args = append(pe.Args, arg)
	} else {
		p.errorf(ErrInvalidExpr, p.tok.Line, p.tok.Col, "invalid expression passed to '%s': '%s'", token.PrefixExprName(pe.Tok.Type), argLit)
		getOutOfPrefixExpr(p)
		return nil
	}
	line, col := p.tok.Line, p.tok.Col
	p.movecws()
	if p.tok.Type != token.CLOSING_PAREN {
		p.errorf(ErrUnclosedPrefixExpr, line, col, "unclosed '%s' expression", token.PrefixExprName(pe.Tok.Type))
		getOutOfPrefixExpr(p)
		return nil
	}
	p.move() // skip )
	return pe
}
