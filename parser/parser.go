package parser

import (
	"fmt"
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
		return nil
	}
	program := &ast.Program{}
loop:
	for {
		switch p.tok.Type {
		case token.EOF:
			break loop
		case token.WHITESPACE:
			p.move()
			continue
		// string literal
		case token.STRING:
			// no need to check nil for this method, but there's no harm in doing just that
			if stmt := p.parseStringLiteral(); stmt != nil {
				program.PushStmt(stmt)
			}
		case token.INT:
			if stmt := p.parseIntLiteral(); stmt != nil {
				program.PushStmt(stmt)
			}
		case token.BOOL:
			if stmt := p.parseBoolLiteral(); stmt != nil {
				program.PushStmt(stmt)
			}
		// parse variable declarations with primitive types
		case token.STRINGKW, token.INTKW, token.BOOLKW:
			if stmt := p.parseVariableDecl(); stmt != nil {
				program.PushStmt(stmt)
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
		default:
			panic("Parse: error: NOT IMPLEMENTED: " + p.tok.Type.String())
		}
	}
	return program
}

// > advance parser at the end

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

func (p *Parser) parseVariableDecl() *ast.VariableDeclaration {
	v := &ast.VariableDeclaration{Tok: p.tok}
	p.move() // move to ws
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(ErrUnexpectedToken, peek.Line, peek.Col, "unexpected token: expected an identifier, but got '%s'", peek.Type)
		p.move()
		return nil
	}
	v.Name = p.tok.Literal
	p.move() // move to ws
	if eqOk, peek := p.expect(token.EQUAL), p.peek(); !(eqOk) {
		p.errorf(ErrUnexpectedToken, peek.Line, peek.Col, "unexpected token: expected an equal sign, but got '%s'", peek.Type)
		p.move()
		return nil
	}
	p.move() // move to ws
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
