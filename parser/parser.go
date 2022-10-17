package parser

import (
	"fmt"
	"os"
	"quoi/ast"
	"quoi/lexer"
	"quoi/token"
)

type Err struct {
	Msg          string
	Column, Line uint
}

func newErr(line, col uint, formatMsg string, elems ...interface{}) Err {
	return Err{Msg: fmt.Sprintf(formatMsg, elems...), Column: col, Line: line}
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

func (p *Parser) moveif(cond bool) {
	if cond {
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

// skip to the next statement (not expr statement)
//
// useful in situations like coming across an erroneous expression, or statement;
// and wanting to ignore the whole statement to prevent giving redundant error messages.
func (p *Parser) skip() {
	kwm := map[token.Type]bool{
		token.INTKW: true, token.STRINGKW: true, token.BOOLKW: true, token.DATATYPE: true,
		token.FUN: true, token.BLOCK: true, token.END: true, token.IF: true, token.ELSEIF: true,
		token.ELSE: true, token.LOOP: true, token.RETURN: true, token.LISTOF: true, token.CONTINUE: true,
		token.BREAK: true,
	}
	/*
		if we are already on a token that is in kwm, that means we wanted to check the peek token.
		we should move here.

		example case :
			listof { x = [].

			in list decl. parse fn :

			if peek := p.peek(); p.errif(!(canBeATypeForList(peek.Type)), newErr(peek.Line, peek.Col,
				"unexpected token '%s' as type for list", peek.Literal)) {
				return nil
			}
	*/
	_, iskw := kwm[p.tok.Type]
	if iskw {
		p.move()
	}
	for {
		if _, iskw := kwm[p.tok.Type]; iskw || p.tok.Type == token.EOF {
			break
		}
		p.move()
	}
}

func (p *Parser) eat(typ token.Type) {
	for p.curis(typ) {
		p.move()
	}
}

// append error if cond.
// skip to the next statement if cond.
// return true if cond.
func (p *Parser) errif(cond bool, err Err) bool {
	if cond {
		p.Errs = append(p.Errs, err)
		p.skip()
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

func (p *Parser) Parse() *ast.Program {
	if len(p.lexerErrors) > 0 {
		for _, e := range p.lexerErrors {
			fmt.Printf("[!] line:col(%d:%d) %s\n", e.Line, e.Column, e.Msg)
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
	case token.ILLEGAL:
		p.errorf(p.tok.Line, p.tok.Col, "illegal token '%s'", p.tok.Literal)
		p.skip()
	case token.NEWLINE:
		p.move()
	case token.STRING:
		if stmt := p.parseStringLiteral(true); stmt != nil {
			return stmt
		}
	case token.INT:
		if stmt := p.parseIntLiteral(true); stmt != nil {
			return stmt
		}
	case token.BOOL:
		if stmt := p.parseBoolLiteral(true); stmt != nil {
			return stmt
		}
	case token.OPENING_SQUARE_BRACKET:
		if stmt := p.parseListLiteral(true); stmt != nil {
			return stmt
		}
	case token.OPENING_PAREN:
		if stmt := p.parseOperator(true); stmt != nil {
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
		switch p.peek().Type {
		case token.EQUAL:
			if stmt := p.parseReassignmentStatement(identTok); stmt != nil {
				return stmt
			}
		case token.OPENING_PAREN:
			if stmt := p.parseFunctionCall(identTok, true); stmt != nil {
				return stmt
			}
		case token.DOUBLE_COLON:
			if stmt := p.parseFunctionCallFromNamespace(identTok, true); stmt != nil {
				return stmt
			}
		case token.IDENT:
			if stmt := p.parseVariableDecl(); stmt != nil {
				return stmt
			}
		}
		// identifier
		if stmt := p.parseIdentifier(true); stmt != nil {
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
	case token.BREAK:
		if stmt := p.parseBreakStatement(); stmt != nil {
			return stmt
		}
	case token.CONTINUE:
		if stmt := p.parseContinueStatement(); stmt != nil {
			return stmt
		}
	case token.LOOP:
		if stmt := p.parseLoopStatement(); stmt != nil {
			return stmt
		}
	case token.DATATYPE:
		if stmt := p.parseDatatypeDeclarationStatement(); stmt != nil {
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

/*
func (p *Parser) parseExprStmt() ast.ExprStmt {
	// expression statements need a dot at the end.
	//
	// for example,
	// 		Stdout::println("Hello world").
	//		Stdout::println(String::from_int(6)).
	// println calls are expr statements, whereas String::from_int call is an expr.

}
*/
// typ is acceptable for parse expr
func isExpr(typ token.Type) bool {
	// these are EXPECTED token types for parseExpr.
	// these can be expressions.
	//
	// other tokens cannot be exprs.
	acceptableTokens := map[token.Type]bool{
		token.STRING: true, token.INT: true, token.BOOL: true, token.IDENT: true, token.OPENING_PAREN: true,
	}
	_, ok := acceptableTokens[typ]
	return ok
}

func (p *Parser) parseStringLiteral(isStmt bool) *ast.StringLiteral {
	// if isStmt, then this is an expression statement.
	// expression statements need a dot at the end, because they are statements.
	s := &ast.StringLiteral{Typ: p.tok.Type, Val: p.tok.Literal}
	p.move()
	if isStmt {
		if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s'. need a dot at the end of a statement", p.tok.Literal)) {
			return nil
		}
		p.move()
	}
	return s
}

func (p *Parser) parseIntLiteral(isStmt bool) *ast.IntLiteral {
	n := atoi(p)
	i := &ast.IntLiteral{Typ: p.tok.Type, Val: n}
	p.move()
	if isStmt {
		if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s'. need a dot at the end of a statement", p.tok.Literal)) {
			return nil
		}
		p.move()
	}
	return i
}

func (p *Parser) parseBoolLiteral(isStmt bool) *ast.BoolLiteral {
	b := atob(p)
	boo := &ast.BoolLiteral{Typ: p.tok.Type, Val: b}
	p.move()
	if isStmt {
		if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s'. need a dot at the end of a statement", p.tok.Literal)) {
			return nil
		}
		p.move()
	}
	return boo
}

func (p *Parser) parseIdentifier(isStmt bool) *ast.Identifier {
	i := &ast.Identifier{Tok: p.tok}
	p.move()
	if isStmt {
		if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s'. need a dot at the end of a statement", p.tok.Literal)) {
			return nil
		}
		p.move()
	}
	return i
}

// decide depending on peek token
func (p *Parser) parseExpr() ast.Expr {
	peek := p.peek()
	p.move()
	switch peek.Type {
	case token.STRING:
		return p.parseStringLiteral(false)
	case token.INT:
		return p.parseIntLiteral(false)
	case token.BOOL:
		return p.parseBoolLiteral(false)
	case token.IDENT:
		return p.parseIdentifier(false)
	case token.OPENING_PAREN:
		return p.parseOperator(false)
	case token.OPENING_SQUARE_BRACKET:
		return p.parseListLiteral(false)
	}
	return nil
}

func (p *Parser) parseVariableDecl() *ast.VariableDeclaration {
	v := &ast.VariableDeclaration{Tok: p.tok}
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token: expected an identifier, but got '%s'", peek.Type)
		p.move()
		return nil
	}
	v.Ident = p.parseIdentifier(false)
	if p.errif(p.curnot(token.EQUAL),
		newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s', expected an equal sign", p.tok.Literal)) {
		return nil
	}
	line, col := p.peek().Line, p.peek().Col
	if p.peekis(token.DOT) {
		goto noval
	}
	if p.errif(!(isExpr(p.peek().Type)),
		newErr(line, col,
			"unexpected token '%s' as value in variable declaration", p.peek().Literal)) {
		return nil
	}
noval:
	v.Value = p.parseExpr()
	if p.errif(v.Value == nil,
		newErr(line, col, "no value set to variable")) {
		return nil
	}
	if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
		"unexpected token: need a dot at the end of a statement")) {
		return nil
	}
	p.move() // skip dot
	return v
}

func (p *Parser) parseReassignmentStatement(identTok token.Token) *ast.ReassignmentStatement {
	r := &ast.ReassignmentStatement{Tok: identTok, Ident: &ast.Identifier{Tok: identTok}}
	if eqOk, peek := p.expect(token.EQUAL), p.peek(); !(eqOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token: expected an equal sign, got '%s'", peek.Type)
		p.move()
		return nil
	}
	line, col := p.peek().Line, p.peek().Col
	if p.errif(!(isExpr(p.peek().Type)), newErr(line, col,
		"unexpected token '%s' as new value in reassignment statement", p.peek().Literal)) {
		fmt.Println("reas: ", p.tok)
		return nil
	}
	r.NewValue = p.parseExpr()
	if p.errif(r.NewValue == nil, newErr(line, col, "no value in reassignment")) {
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
	p.move()
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
	if p.peekis(token.DOT) {
		// first time using goto :D
		goto noval
	}
	if p.errif(!(isExpr(p.peek().Type)), newErr(p.peek().Line, p.peek().Col,
		"unexpected token '%s' as return value in return statement", p.peek().Literal)) {
		return nil
	}
noval:
	r.Expr = p.parseExpr()
	if p.errif(r.Expr == nil, newErr(line, col, "return statement with no value")) {
		return nil
	}
	if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
		"unexpected token: need a dot at the end of a return statement")) {
		return nil
	}
	p.move()
	return r
}

func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	// current token is token.BREAK
	b := &ast.BreakStatement{Tok: p.tok}
	if p.errif(!(p.peekis(token.DOT)), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' at the end of break statement where a dot was expected", p.peek().Literal)) {
		return nil
	}
	p.dmove()
	return b
}

func (p *Parser) parseContinueStatement() *ast.ContinueStatement {
	// current token is token.CONTINUE
	c := &ast.ContinueStatement{Tok: p.tok}
	if p.errif(!(p.peekis(token.DOT)), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' at the end of continue statement where a dot was expected", p.peek().Literal)) {
		return nil
	}
	p.dmove()
	return c
}

func (p *Parser) parseLoopStatement() *ast.LoopStatement {
	l := &ast.LoopStatement{Tok: p.tok}
	line, col := p.peek().Line, p.peek().Col
	if p.peekis(token.OPENING_CURLY) {
		goto nocond
	}
	if p.errif(!(isExpr(p.peek().Type)), newErr(line, col,
		"unexpected token '%s' in loop statement. loop statement condition must be an expression", p.peek().Literal)) {
		return nil
	}
nocond:
	l.Cond = p.parseExpr()
	if p.errif(l.Cond == nil, newErr(line, col, "missing condition in loop statement")) {
		return nil
	}
	if p.errif(p.curnot(token.OPENING_CURLY), newErr(p.tok.Line, p.tok.Col,
		"unexpected token: expected an opening curly brace, got '%s'", p.tok.Type)) {
		return nil
	}
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
		if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
			p.errorf(peek.Line, peek.Col, "missing identifier: expected an identifier in datatype field")
			p.skip()
			return nil
		}
		f.Ident = p.parseIdentifier(false)
		if p.errif(p.curnot(token.NEWLINE), newErr(p.tok.Line, p.tok.Col,
			"missing newline after datatype field")) {
			return nil
		}
	default:
		p.errorf(p.tok.Line, p.tok.Col, "invalid token '%s' for datatype field", p.tok.Literal)
		return nil
	}
	p.move()
	return f
}

func (p *Parser) parseDatatypeDeclarationStatement() *ast.DatatypeDeclaration {
	d := &ast.DatatypeDeclaration{Tok: p.tok}
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(peek.Line, peek.Col, "datatype without a name")
		p.skip()
		return nil
	}
	line, col := p.tok.Line, p.tok.Col
	name := p.parseIdentifier(false)
	if p.errif(name == nil, newErr(line, col, "expected a name for the datatype declaration")) {
		return nil
	}
	d.Name = name
	if p.errif(p.curnot(token.OPENING_CURLY), newErr(p.tok.Line, p.tok.Col,
		"missing opening curly brace in datatype declaration")) {
		return nil
	}
	p.move() // skip {
	p.moveif(p.curis(token.NEWLINE))
	for {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: expected a closing curly brace at the end of datatype declaration")) {
			return nil
		}
		if p.curis(token.CLOSING_CURLY) {
			break
		}
		field := p.parseDatatypeField()
		if field == nil {
			p.skip()
			return nil
		}
		/*
			after parsing a field, there may be extra newlines.
			e.g.
			datatype User {
				int x


				;; heey
			}
			consume those newlines, and, hopefully, stop at }
		*/
		p.eat(token.NEWLINE)
		d.Fields = append(d.Fields, field)
	}
	if p.errif(p.curnot(token.CLOSING_CURLY), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s'. expected a closing curly brace at the end of datatype declaration",
		p.tok.Literal)) {
		return nil
	}
	p.move()
	return d
}

// TODO BUGGY vvv

// > rewrite this

func (p *Parser) parseOperator(isStmt bool) *ast.PrefixExpr {
	// current token is token.OPENING_PAREN
	pe := &ast.PrefixExpr{}
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
	fmt.Println(p.tok)
	for p.curnot(token.CLOSING_PAREN) && p.curnot(token.EOF) {
		// call parseStatement, instead of parseExpr.
		// because we are dealing with p.tok. (parseExpr looks to p.peek())
		// I am lazy to restructure the whole method.
		if arg := p.parseExpr(); arg != nil {
			pe.Args = append(pe.Args, arg)
		}
		fmt.Println("p.tt: ", p.tok)
		if p.errif(!(isExpr(p.tok.Type)), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s' in '%s' expression",
			p.tok.Literal, pe.Tok.Literal)) {
			return nil
		}
	}
	if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
		"unexpected end-of-file: missing ')' at the end of prefix expression")) {
		return nil
	}
	// no need for 'p.tok.Type == token.CLOSING_PAREN'
	p.move() // skip )
	if isStmt {
		if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s'. need a dot at the end of a statement", p.tok.Literal)) {
			return nil
		}
		p.move()
	}
	return pe
}

func (p *Parser) parseFunctionCall(ident token.Token, isStmt bool) *ast.FunctionCall {
	fc := &ast.FunctionCall{Tok: ident, Ident: &ast.Identifier{Tok: ident}}
	// peek token is (
	p.move()
	if p.errif(p.curnot(token.OPENING_PAREN), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' in function call expression. expected a '('",
		p.tok.Literal)) {
		return nil
	}
	if p.peekis(token.CLOSING_PAREN) {
		p.dmove()
		return fc
	}
	if p.errif(!(isExpr(p.peek().Type)), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' as argument value to function call",
		p.peek().Literal)) {
		return nil
	}
	arg := p.parseExpr()
	if arg != nil {
		fc.Args = append(fc.Args, arg)
	}
	// one argument
	if p.curis(token.CLOSING_PAREN) {
		goto end
	}
	for {
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
		if p.errif(!(isExpr(p.peek().Type)), newErr(p.tok.Line, p.tok.Col,
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
	if isStmt {
		if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s'. need a dot at the end of a statement", p.tok.Literal)) {
			return nil
		}
		p.move()
	}
	return fc
}

func (p *Parser) parseFunctionCallFromNamespace(namespaceTok token.Token, isStmt bool) *ast.FunctionCallFromNamespace {
	fcfn := &ast.FunctionCallFromNamespace{Namespace: &ast.Namespace{Tok: namespaceTok, Identifier: &ast.Identifier{Tok: namespaceTok}}}
	if dColonOk, peek := p.expect(token.DOUBLE_COLON), p.peek(); !(dColonOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token '%s' when calling a function from namespace '%s'. expected `::`", peek.Literal, fcfn.Namespace.Identifier.String())
		p.move()
		return nil
	}
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token '%s'. expected a function name", peek.Literal)
		p.move()
		return nil
	}
	if fn := p.parseFunctionCall(p.tok, isStmt); fn != nil {
		fcfn.Function = fn
	}
	return fcfn
}
func (p *Parser) parseListVariableDecl() *ast.ListVariableDecl {
	// current token is 'listof'
	l := &ast.ListVariableDecl{Tok: p.tok}
	var canBeATypeForList = func(tok token.Type) bool {
		tflm := map[token.Type]bool{
			token.INTKW: true, token.STRINGKW: true, token.BOOLKW: true, token.IDENT: true,
			token.LISTOF: true,
		}
		_, ok := tflm[tok]
		return ok
	}
	if peek := p.peek(); p.errif(!(canBeATypeForList(peek.Type)), newErr(peek.Line, peek.Col,
		"unexpected token '%s' as type for list", peek.Literal)) {
		return nil
	}
	p.move()
	l.Typ = p.tok
	if identOk, peek := p.expect(token.IDENT), p.peek(); !(identOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token '%s'. expected an identifier in list declaration", peek.Literal)
		p.skip()
		return nil
	}
	l.Name = p.parseIdentifier(false)
	if p.errif(p.curnot(token.EQUAL), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s'. expected an equal sign",
		p.tok.Literal)) {
		return nil
	}
	if openSqBrOk, peek := p.expect(token.OPENING_SQUARE_BRACKET), p.peek(); !(openSqBrOk) {
		p.errorf(peek.Line, peek.Col, "unexpected token '%s', where a '[' was expected", peek.Literal)
		p.skip()
		return nil
	}
	list := p.parseListLiteral(false)
	if list == nil {
		p.skip()
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

func (p *Parser) parseListLiteral(isStmt bool) *ast.ListLiteral {
	// current token is '['
	l := &ast.ListLiteral{Tok: p.tok}
	// no elems
	if p.peekis(token.CLOSING_SQUARE_BRACKET) {
		p.dmove()
		return l
	}
	if peek := p.peek(); p.errif(!(isExpr(peek.Type)), newErr(peek.Line, peek.Col,
		"unexpected token '%s' as list element",
		peek.Literal)) {
		return nil
	}
	if firstEl := p.parseExpr(); firstEl != nil {
		l.Elems = append(l.Elems, firstEl)
	}
	for {
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
		if peek := p.peek(); p.errif(!(isExpr(peek.Type)), newErr(peek.Line, peek.Col,
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
	if isStmt {
		if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s'. need a dot at the end of a statement", p.tok.Literal)) {
			return nil
		}
		p.move() // skip .
	}
	return l
}
