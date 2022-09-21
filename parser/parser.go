package parser

import (
	"fmt"
	"os"
	"quoi/ast"
	"quoi/lexer"
	"quoi/token"
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
	ErrWrongNumberOfArgs
	ErrUnknownOperator
	ErrLonelyExpr // expressions that are not tied to any variable
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
			fmt.Printf("[!] error %d: col:line(%d:%d) %s\n", e.ErrCode, e.Line, e.Column, e.Msg)
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
	case token.PRINT:
		if stmt := p.parsePrintStatement(); stmt != nil {
			return stmt
		}
	// parse infix expression
	case token.OPERATOR:
		if stmt := p.parseOperator(); stmt != nil {
			return stmt
		}
	case token.BLOCK:
		if stmt := p.parseBlockStatement(); stmt != nil {
			return stmt
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
	case token.OPERATOR:
		return p.parseOperator()
	case token.DOT, token.EOF:
		p.move()
		return nil
	default:
		panic("parseExpr: error: NOT IMPLEMENTED: " + peek.Type.String())
	}
}

func (p *Parser) parseVariableDecl() *ast.VariableDeclaration {
	v := &ast.VariableDeclaration{Tok: p.tok}
	p.move() // move to ws
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(ErrUnexpectedToken, peek.Line, peek.Col, "unexpected token: expected an identifier, but got '%s'", peek.Type)
		p.move()
		return nil
	}
	v.Ident = &ast.Identifier{Tok: p.tok}
	p.movews()
	if eqOk, peek := p.expect(token.EQUAL), p.peek(); !(eqOk) {
		p.errorf(ErrUnexpectedToken, peek.Line, peek.Col, "unexpected token: expected an equal sign, but got '%s'", peek.Type)
		p.move()
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

func (p *Parser) parsePrintStatement() *ast.PrintStatement {
	s := &ast.PrintStatement{Tok: p.tok}
	p.move() // move to ws
	if peek := p.peek(); peek.Type == token.EOF {
		p.errorf(ErrUnexpectedEOF, peek.Line, peek.Col, "unexpected end-of-file: expected an argument to 'print' statement")
		p.move()
		return nil
	}
	// parseExpr depends on peek, so, don't move here.
	line, col := p.tok.Line, p.tok.Col
	s.Arg = p.parseExpr()
	if s.Arg == nil {
		p.errorf(ErrNoValue, line, col, "print statement needs one argument, but none was given.")
		return nil
	}
	if p.tok.Type != token.DOT {
		p.errorf(ErrUnexpectedToken, p.tok.Line, p.tok.Col, "unexpected token: need a dot at the end of a print statement")
		return nil
	}
	p.move() // skip dot
	return s
}

func parseOperatorWith(p *Parser, nArgs int) *ast.PrefixExpr {
	operatorLit := p.tok.Literal
	pe := &ast.PrefixExpr{}
	pe.Operator = p.tok
	p.movews()
	if nArgs == 0 || nArgs > 3 {
		panic(fmt.Sprintf("parseOperatorWith: invalid nArgs: %d", nArgs))
	}
	line, col := p.tok.Line, p.tok.Col
	for i := 0; i < nArgs; i++ {
		expr := p.parseExpr()
		if expr != nil {
			pe.Args = append(pe.Args, expr)
		}
	}
	switch nArgs {
	case 1:
		if len(pe.Args) != 1 {
			p.errorf(ErrNoValue, line, col, "no value when calling an operator with one required parameter")
			return nil
		}
	case 2:
		if len(pe.Args) != 2 {
			p.errorf(ErrWrongNumberOfArgs, line, col, "wrong number of arguments (%d) to call '%s'. it needs two arguments", len(pe.Args), operatorLit)
			return nil
		}
	case 3:
		if len(pe.Args) != 3 {
			p.errorf(ErrWrongNumberOfArgs, line, col, "wrong number of arguments (%d) to call '%s'. it needs three arguments", len(pe.Args), operatorLit)
			return nil
		}
	}
	return pe
}

func (p *Parser) parseOperator() *ast.PrefixExpr {
	switch p.tok.Literal {
	case "@new", "@listnew":
		panic("implement these specials @new, @listnew, etc.")
	case "@inc", "@dec", "@str", "@not":
		return parseOperatorWith(p, 1)
	case "@strreplace", "@listreplace":
		return parseOperatorWith(p, 3)
	default:
		return parseOperatorWith(p, 2)
	}
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
