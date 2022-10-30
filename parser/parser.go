package parser

import (
	"fmt"
	"os"
	"quoi/ast"
	"quoi/lexer"
	"quoi/token"
)

// TODO Don't give redundant error messages.

// TODO listofs in subsequent variable declarations

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

func (p *Parser) unmove() {
	if p.ptr < 1 {
		return
	}
	p.ptr--
	p.tok = p.tokens[p.ptr]
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

func (p *Parser) peekN(ahead uint) token.Token {
	tokensLen := uint(len(p.tokens))
	ok := p.ptr+ahead < tokensLen
	if ok {
		return p.tokens[p.ptr+ahead]
	}
	return p.tokens[tokensLen-1]
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

// typ is acceptable for parse expr
func isExpr(typ token.Type) bool {
	// these are EXPECTED token types for parseExpr.
	// these can be expressions.
	//
	// other tokens cannot be exprs.
	acceptableTokens := map[token.Type]bool{
		token.STRING: true, token.INT: true, token.BOOL: true, token.IDENT: true, token.OPENING_PAREN: true, token.OPENING_SQUARE_BRACKET: true,
	}
	_, ok := acceptableTokens[typ]
	return ok
}

func isOperator(tok token.Token) bool {
	lit := tok.Literal
	opm := map[string]bool{
		"+": true, "-": true, "*": true, "/": true, "'": true, "=": true, "lt": true, "lte": true,
		"gt": true, "gte": true, "and": true, "or": true, "not": true, "set": true, "get": true,
	}
	ok := opm[lit]
	return ok
}

func isReturnOrFunctionParamType(tok token.Type) bool {
	rtm := map[token.Type]bool{
		token.INTKW: true, token.STRINGKW: true, token.BOOLKW: true, token.IDENT: true, token.LISTOF: true,
	}
	return rtm[tok]
}

// decide whether there is a comma after <<type> <name>> pair, starting with 'type'.
// there could be many newlines after 'type'.
//
// int
//
//		x, string y = 5, "Hello". 				is valid.
//
// starting from 'int', decide if we should parse it as a *ast.SubsequentVariableDeclarationStatement, or not.
func isASubseqVariableDecl(p *Parser) bool {
	// save ptr here to revert back to the old position of the parser.
	ptr := p.ptr // current token is a type, or a token.LISTOF.
	p.moveif(p.curis(token.LISTOF))
	p.move()
	p.eat(token.NEWLINE)
	if p.curnot(token.IDENT) {
		// in this case, parseStatement will call parseVariableDeclarationStatement, and it will give an error.
		// we don't care about this here actually.
		return false
	}
	p.move()
	ok := p.curis(token.COMMA)
	p.ptr = ptr
	p.tok = p.tokens[p.ptr]
	return ok
}

// dangerously similar to isASubseqVariableDecl.
func isDatatypeInitialization(p *Parser) bool {
	// current: token.IDENT
	ptr := p.ptr
	p.move()
	p.eat(token.NEWLINE)
	if p.curnot(token.OPENING_CURLY) {
		return false
	}
	p.ptr = ptr
	p.tok = p.tokens[p.ptr]
	return true
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

	thisIsAStmt := true
	switch p.tok.Type {
	case token.ILLEGAL:
		p.errorf(p.tok.Line, p.tok.Col, "illegal token '%s'", p.tok.Literal)
		p.skip()
	case token.NEWLINE:
		p.move()
	case token.STRING:
		if stmt := p.parseStringLiteral(thisIsAStmt); stmt != nil {
			return stmt
		}
	case token.INT:
		if stmt := p.parseIntLiteral(thisIsAStmt); stmt != nil {
			return stmt
		}
	case token.BOOL:
		if stmt := p.parseBoolLiteral(thisIsAStmt); stmt != nil {
			return stmt
		}
	case token.OPENING_SQUARE_BRACKET:
		if stmt := p.parseListLiteral(thisIsAStmt); stmt != nil {
			return stmt
		}
	case token.OPENING_PAREN:
		if stmt := p.parseOperator(thisIsAStmt); stmt != nil {
			return stmt
		}
	// parse variable declarations with primitive types
	case token.STRINGKW, token.INTKW, token.BOOLKW:
		isSubseq := isASubseqVariableDecl(p)
		if isSubseq {
			if stmt := p.parseSubsequentVariableDeclarationStatement(); stmt != nil {
				return stmt
			}
		}
		if stmt := p.parseVariableDeclarationStatement(); stmt != nil {
			return stmt
		}
	case token.LISTOF:
		isSubseq := isASubseqVariableDecl(p)
		if isSubseq {
			if stmt := p.parseSubsequentVariableDeclarationStatement(); stmt != nil {
				return stmt
			}
		}
		if stmt := p.parseListVariableDeclarationStatement(); stmt != nil {
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
			if stmt := p.parseFunctionCall(identTok, thisIsAStmt, ""); stmt != nil {
				return stmt
			}
		case token.DOUBLE_COLON:
			if stmt := p.parseFunctionCallFromNamespace(identTok, true); stmt != nil {
				return stmt
			}
		}
		if isDatatypeInitialization(p) {
			if stmt := p.parseDatatypeLiteral(thisIsAStmt); stmt != nil {
				return stmt
			}
		}
		// identifier
		if stmt := p.parseIdentifier(thisIsAStmt); stmt != nil {
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
	case token.IF:
		if stmt := p.parseIfStatement(false); stmt != nil {
			return stmt
		}
	case token.ELSEIF, token.ELSE:
		p.errorf(p.tok.Line, p.tok.Col, "elseif/else statement without a preceding if statement")
		p.skip()
		return nil
	case token.FUN:
		if stmt := p.parseFunctionDeclarationStatement(); stmt != nil {
			return stmt
		}
	case token.EOF:
		break
	default:
		p.errorf(p.tok.Line, p.tok.Col, "unexpected token '%s'", p.tok.Literal)
		tokTyp := p.tok.Type
		// skip token of same type to avoid giving repetitive error messages
		p.eat(tokTyp)
	}
	return nil
}

func (p *Parser) parseStringLiteral(isStmt bool) *ast.StringLiteral {
	if p.errif(p.curnot(token.STRING), newErr(p.tok.Line, p.tok.Col, "illegal string '%s'", p.tok.Literal)) {
		return nil
	}
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
	// this can help me in debugging.
	if p.errif(p.curnot(token.INT), newErr(p.tok.Line, p.tok.Col, "illegal integer '%s'", p.tok.Literal)) {
		return nil
	}
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
	// this can help me in debugging.
	if p.errif(p.curnot(token.BOOL), newErr(p.tok.Line, p.tok.Col, "illegal boolean '%s'", p.tok.Literal)) {
		return nil
	}
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
	// this can help me in debugging.
	if p.errif(p.curnot(token.IDENT), newErr(p.tok.Line, p.tok.Col, "illegal identifier '%s'", p.tok.Literal)) {
		return nil
	}
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
		identTok := p.tok
		switch p.peek().Type {
		case token.OPENING_PAREN:
			return p.parseFunctionCall(identTok, false, "")
		case token.DOUBLE_COLON:
			return p.parseFunctionCallFromNamespace(identTok, false)
		}
		if isDatatypeInitialization(p) {
			return p.parseDatatypeLiteral(false)
		}
		return p.parseIdentifier(false)
	case token.OPENING_PAREN:
		return p.parseOperator(false)
	case token.OPENING_SQUARE_BRACKET:
		return p.parseListLiteral(false)
	}

	return nil
}

// tok, isList, listType, identifier
func (p *Parser) parseVariableTypeAndName() (token.Token, bool, token.Token, *ast.Identifier) {
	// current token is a type must be a type
	var (
		tok     token.Token
		listTyp token.Token
		id      *ast.Identifier
		isList  bool
	)
	if p.errif(!(isReturnOrFunctionParamType(p.tok.Type)), newErr(p.tok.Line, p.tok.Col,
		"illegal type '%s' in variable declaration statement", p.tok.Literal)) {
		return tok, isList, listTyp, nil
	}
	tok = p.tok
	if p.curis(token.LISTOF) {
		isList = true
		p.move()
		if p.errif(!(isReturnOrFunctionParamType(p.tok.Type)), newErr(p.tok.Line, p.tok.Col,
			"illegal type '%s' in list variable declaration statement", p.tok.Literal)) {
			return tok, isList, listTyp, nil
		}
		listTyp = p.tok
	}
	p.move()
	p.eat(token.NEWLINE)
	// allow newline after type.
	// int
	//     x = 5.
	// is legal.
	if p.errif(p.curnot(token.IDENT), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' in variable declaration statement, where an identifier were expected after type '%s'",
		p.tok.Literal, tok.Literal)) {
		return tok, isList, listTyp, nil
	}
	isStmt := false
	id = p.parseIdentifier(isStmt)
	return tok, isList, listTyp, id
}

func (p *Parser) parseVariableDeclarationStatement() *ast.VariableDeclarationStatement {
	var v = &ast.VariableDeclarationStatement{}
	tok, _, _, id := p.parseVariableTypeAndName()
	if id == nil { // parseVariableTypeAndName's second return value is nil, only when there was an error
		// we don't report any errors here; because, parseVariableTypeAndName already did that for us.
		return nil
	}
	v.Tok = tok
	v.Ident = id
	if p.errif(p.curnot(token.EQUAL), newErr(p.tok.Line, p.tok.Col,
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
		newErr(line, col, "no value set to variable '%s'", v.Ident.String())) {
		return nil
	}
	if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
		"unexpected token: need a dot at the end of a statement")) {
		return nil
	}
	p.move()
	return v
}

/* parse comma separated variables */
/* like: */
/* int n, string y, bool z = <expr>, ... . */
/*  */
/* this is called whenever we see a comma after an identifier in parseVariableDecl */
func (p *Parser) parseSubsequentVariableDeclarationStatement() *ast.SubsequentVariableDeclarationStatement {
	// current token is a type
	var res = &ast.SubsequentVariableDeclarationStatement{}
	// parse types, and names
	for p.curnot(token.EQUAL) {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: unfinished subsequent variable declaration statement")) {
			return nil
		}
		/*
			NOTE: very interesting bug here
			if we return listTyp as a *token.Token here, the returned tokens are ',', and '=', respectively; when the following
			input is given:
				``` listof int x, listof string y = [], []. ```
			The interesting thing here is that, in parseVariableTypeAndName, the types of lists are actually correctly set.
			(int, and string; respectively.)
			I couldn't figure out the reason why, so I switched to returning non-pointer token.Token type.
		*/
		tok, isList, listTyp, id := p.parseVariableTypeAndName()
		if id == nil {
			return nil
		}
		typ := ast.VarType{Tok: tok, IsList: isList, TypeOfList: listTyp}
		res.Types = append(res.Types, typ)
		res.Names = append(res.Names, id)
		if p.curis(token.EQUAL) {
			break
		}
		if p.errif(p.curnot(token.COMMA), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s' where a comma was expected in subsequent variable declaration '%s %s'",
			p.tok.Literal, tok.Literal, id.String())) {
			return nil
		}
		p.move() // skip ,
	}
	p.move() // skip =
	for p.curnot(token.DOT) {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: unfinished subsequent variable declaration statement")) {
			return nil
		}
		if p.errif(!(isExpr(p.tok.Type)), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s' as value in subsequent variable declaration statement", p.tok.Literal)) {
			return nil
		}
		p.unmove()
		if xpr := p.parseExpr(); xpr != nil {
			res.Values = append(res.Values, xpr)
		}
		if p.curis(token.DOT) {
			break
		}
		if p.errif(p.curnot(token.COMMA), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s' in subsequent variable declaration statement, where a comma was expected", p.tok.Literal)) {
			return nil
		}
		p.move() // skip ,
		p.eat(token.NEWLINE)
	}
	p.move() // skip .
	return res
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
		// I could've just swapped this if statement, and the one below; I know.
		goto noval
	}
	if p.errif(!(isExpr(p.peek().Type)), newErr(p.peek().Line, p.peek().Col,
		"unexpected token '%s' as return value in return statement", p.peek().Literal)) {
		return nil
	}
noval:
	expr := p.parseExpr()
	if p.errif(expr == nil, newErr(line, col, "return statement with no value")) {
		return nil
	}
	r.ReturnValues = append(r.ReturnValues, expr)
	// multiple returns
	if p.curis(token.COMMA) {
		for p.curnot(token.DOT) {
			if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col, "unexpected end-of-file: unfinished return statement")) {
				return nil
			}
			if p.errif(!(isExpr(p.peek().Type)), newErr(p.peek().Line, p.peek().Col,
				"unexpected token '%s' as return value in return statement", p.peek().Literal)) {
				return nil
			}
			if expr := p.parseExpr(); expr != nil {
				r.ReturnValues = append(r.ReturnValues, expr)
			}
		}
	}
	if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' where a dot was expected at the end of a return statement", p.tok.Literal)) {
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
	if p.errif(p.peekis(token.OPENING_CURLY), newErr(line, col, "missing condition in loop statement")) {
		return nil
	}
	if p.errif(!(isExpr(p.peek().Type)), newErr(line, col,
		"unexpected token '%s' in loop statement. loop statement condition must be an expression", p.peek().Literal)) {
		return nil
	}
	l.Cond = p.parseExpr()
	if p.errif(l.Cond == nil, newErr(line, col, "missing condition in loop statement")) {
		return nil
	}
	if p.errif(p.curnot(token.OPENING_CURLY), newErr(p.tok.Line, p.tok.Col,
		"unexpected token: expected an opening curly brace, got '%s'", p.tok.Type)) {
		return nil
	}
	p.move() // skip '{'
	for p.curnot(token.CLOSING_CURLY) {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: unclosed loop statement")) {
			return nil
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

func (p *Parser) parseOperator(isStmt bool) *ast.PrefixExpr {
	// current token is token.OPENING_PAREN
	pe := &ast.PrefixExpr{}
	if p.errif(p.peekis(token.EOF), newErr(p.tok.Line, p.tok.Col,
		"unexpected end-of-file: expected an operator after '(', in prefix expression")) {
		return nil
	}
	if p.errif(p.peekis(token.CLOSING_PAREN), newErr(p.tok.Line, p.tok.Col,
		"missing operator in prefix expression")) {
		return nil
	}
	if peek := p.peek(); p.errif(!(isOperator(peek)), newErr(peek.Line, peek.Col,
		"unknown operator '%s' in prefix expression", peek.Literal)) {
		return nil
	}
	p.move()
	pe.Tok = p.tok // set operator
	p.move()       // skip operator
	for p.curnot(token.CLOSING_PAREN) {
		p.moveif(p.curis(token.NEWLINE))
		if p.curis(token.CLOSING_PAREN) {
			break
		}
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: expected a closing ')'")) {
			return nil
		}
		if p.errif(!(isExpr(p.tok.Type)), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s' as argument to operator '%s'. it expects expressions",
			p.tok.Literal, token.PrefixExprName(pe.Tok.Type))) {
			return nil
		}
		p.unmove()
		if el := p.parseExpr(); el != nil {
			pe.Args = append(pe.Args, el)
		}
	}
	// current token is token.CLOSING_PAREN
	p.move()
	if isStmt {
		if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s' at the end of prefix expression, where a dot was expected", p.tok.Literal)) {
			return nil
		}
		p.move()
	}
	return pe
}

// namespace argument is used in errors.
// e.g.
// report "... in function call 'Math::pow', instead of 'pow'"
func (p *Parser) parseFunctionCall(ident token.Token, isStmt bool, namespace string) *ast.FunctionCall {
	// current token is token.IDENT
	fc := &ast.FunctionCall{Tok: p.tok, Ident: &ast.Identifier{Tok: p.tok}}
	var fnName = fc.Ident.String()
	if len(namespace) > 0 {
		fnName = namespace + "::" + fnName
	}
	p.move()
	if p.errif(p.curnot(token.OPENING_PAREN), newErr(p.tok.Line, p.tok.Col,
		"missing '(' in function call '%s'", fnName)) {
		return nil
	}
	if p.peekis(token.CLOSING_PAREN) {
		p.dmove()
		goto end
	}
	p.moveif(p.peekis(token.NEWLINE))
	for p.curnot(token.CLOSING_PAREN) {
		p.moveif(p.peekis(token.NEWLINE))
		if p.peekis(token.CLOSING_PAREN) {
			p.dmove()
			goto end
		}
		if peek := p.peek(); p.errif(!(isExpr(peek.Type)), newErr(peek.Line, peek.Col,
			"unexpected token '%s' as argument to function call '%s'. it expects expressions", peek.Literal, fnName)) {
			return nil
		}
		if arg := p.parseExpr(); arg != nil {
			fc.Args = append(fc.Args, arg)
		}
		if p.errif(p.curis(token.COMMA) && p.peekis(token.CLOSING_PAREN), newErr(p.tok.Line, p.tok.Col,
			"redundant comma in function call '%s'", fnName)) {
			return nil
		}
		if p.curis(token.CLOSING_PAREN) {
			p.move()
			goto end
		}
		if p.errif(p.curnot(token.COMMA), newErr(p.tok.Line, p.tok.Col, "missing comma in function call '%s'", fnName)) {
			return nil
		}
	}
end:
	if isStmt {
		if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s' at the end of function call expression statement '%s', where a dot was expected", p.tok.Literal, fnName)) {
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
	if fn := p.parseFunctionCall(p.tok, isStmt, fcfn.Namespace.Tok.Literal); fn != nil {
		fcfn.Function = fn
	}
	return fcfn
}

func (p *Parser) parseListVariableDeclarationStatement() *ast.ListVariableDeclarationStatement {
	// current token is 'listof'
	l := &ast.ListVariableDeclarationStatement{Tok: p.tok}
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

// isAlternative:
// 	true => elseif
//	false => if
func (p *Parser) parseIfStatement(isAlternative bool) *ast.IfStatement {
	// current token is token.IF
	i := &ast.IfStatement{Tok: p.tok}
	stmtType := "if"
	if isAlternative {
		stmtType = "elseif"
	}
	if peek := p.peek(); p.errif(!(isExpr(p.peek().Type)), newErr(peek.Line, peek.Col,
		"unexpected token '%s' as condition to %s statement", peek.Literal, stmtType)) {
		return nil
	}
	line, col := p.peek().Line, p.peek().Col
	if cond := p.parseExpr(); cond != nil {
		i.Cond = cond
	}
	if p.errif(i.Cond == nil, newErr(line, col, "no condition in %s statement body", stmtType)) {
		return nil
	}
	if p.errif(p.curnot(token.OPENING_CURLY), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' in if statement, where a '{' was expected", p.tok.Literal)) {
		return nil
	}
	p.move() // skip {
	for p.curnot(token.CLOSING_CURLY) {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col, "unexpected end-of-file: unclosed %s statement", stmtType)) {
			return nil
		}
		if stmt := p.parseStatement(); stmt != nil {
			i.Stmts = append(i.Stmts, stmt)
		}
	}
	// current token is token.CLOSING_CURLY
	p.move()
	switch p.tok.Type {
	case token.ELSEIF:
		i.Alternative = p.parseIfStatement(true)
	case token.ELSE:
		i.Default = p.parseElseStatement()
	}
	return i
}

func (p *Parser) parseElseStatement() *ast.ElseStatement {
	// current token is token.ELSE
	e := &ast.ElseStatement{Tok: p.tok}
	if openingCurlyOk := p.expect(token.OPENING_CURLY); !(openingCurlyOk) {
		p.errorf(p.tok.Line, p.tok.Col, "unexpected token '%s', where a '{' was expected in else statement", p.tok.Literal)
		p.skip()
		return nil
	}
	p.move() // skip {
	for p.curnot(token.CLOSING_CURLY) {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col, "unexpected end-of-file: unclosed else statement")) {
			return nil
		}
		if stmt := p.parseStatement(); stmt != nil {
			e.Stmts = append(e.Stmts, stmt)
		}
	}
	p.move()
	return e
}

func (p *Parser) parseFunctionParam(fnName string) *ast.FunctionParameter {
	// current token is a type, or 'listof'
	param := &ast.FunctionParameter{Tok: p.tok}
	param.IsList = p.curis(token.LISTOF)
	validType := isReturnOrFunctionParamType(param.Tok.Type)
	if param.IsList {
		// expect the type of list
		p.move()
		// no multi-dimensional list
		if p.errif(p.curis(token.LISTOF), newErr(p.tok.Line, p.tok.Col,
			"illegal multi-dimensional list as parameter type in function declaration '%s'", fnName)) {
			return nil
		}
		if p.errif(!(isReturnOrFunctionParamType(p.tok.Type)), newErr(p.tok.Line, p.tok.Col,
			"invalid parameter type '%s' in function declaration '%s'", p.tok.Literal, fnName)) {
			return nil
		}
		param.TypeOfList = p.tok
		validType = isReturnOrFunctionParamType(param.TypeOfList.Type)
	}
	type_ := param.Tok.Literal
	if param.IsList {
		type_ = "listof " + param.TypeOfList.Literal
	}
	if p.errif(!(validType), newErr(p.tok.Line, p.tok.Col,
		"invalid type '%s' for parameter in function declaration '%s'", type_, fnName)) {
		return nil
	}
	p.move()
	type_ = param.Tok.Literal
	if param.IsList {
		type_ += " " + param.TypeOfList.Literal
	}
	if p.errif(p.curis(token.NEWLINE), newErr(p.tok.Line, p.tok.Col,
		"illegal newline after type '%s' in parameter list, in function declaration '%s'", type_, fnName)) {
		return nil
	}
	if p.errif(p.curnot(token.IDENT), newErr(p.tok.Line, p.tok.Col,
		"missing parameter name in function declaration '%s'", fnName)) {
		return nil
	}
	// I want named arguments in Go.
	// like: `p.parseIdentifier(isStmt=false)`, or `p.parseIdentifier(isStmt: false)`
	// it'd be so good; especially when there are booleans to pass.
	isStmt := false
	param.Name = p.parseIdentifier(isStmt)
	// expect comma
	if p.curis(token.CLOSING_PAREN) {
		return param
	}
	if p.errif(p.curnot(token.COMMA) && !(p.peekis(token.CLOSING_PAREN)), newErr(p.tok.Line, p.tok.Col,
		"missing comma between parameters in function declaration '%s'", fnName)) {
		return nil
	}
	line, col := p.tok.Line, p.tok.Col
	stoppedAtComma := p.curis(token.COMMA)
	redundantComma := stoppedAtComma && (p.peekis(token.CLOSING_PAREN) || p.peekis(token.NEWLINE) && p.peekN(2).Type == token.CLOSING_PAREN)
	if p.errif(redundantComma, newErr(line, col,
		"redundant comma in parameter list of function declaration '%s'", fnName)) {
		return nil
	}
	p.move() // skip ,
	p.moveif(p.curis(token.NEWLINE))
	return param
}

func (p *Parser) parseFunctionParams(fnName string) []ast.FunctionParameter {
	// current token is on a type
	var res = []ast.FunctionParameter{}
	for p.curnot(token.CLOSING_PAREN) {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: unclosed parameter list in function '%s'", fnName)) {
			return nil
		}
		param := p.parseFunctionParam(fnName)
		if param == nil {
			return nil
		}
		res = append(res, *param)
	}
	p.move() // skip )
	return res
}

func (p *Parser) parseFunctionReturnType(fnName string) *ast.FunctionReturnType {
	// current token is a type
	frt := &ast.FunctionReturnType{Tok: p.tok}
	frt.IsList = frt.Tok.Type == token.LISTOF
	if frt.IsList {
		p.move()
		if p.errif(!(isReturnOrFunctionParamType(p.tok.Type)), newErr(p.tok.Line, p.tok.Col,
			"invalid type 'listof %s' as return type in function declaration '%s'", p.tok.Literal, fnName)) {
			return nil
		}
		if p.errif(p.curis(token.LISTOF), newErr(p.tok.Line, p.tok.Col,
			"illegal multi-dimensional list as return type in function declaration '%s'", fnName)) {
			return nil
		}
		frt.TypeOfList = p.tok
	}
	if p.errif(!(isReturnOrFunctionParamType(frt.Tok.Type)), newErr(p.tok.Line, p.tok.Col,
		"invalid type '%s' as return type in function declaration '%s'", p.tok.Literal, fnName)) {
		return nil
	}
	p.move()
	if p.curis(token.OPENING_CURLY) {
		return frt
	}
	if p.errif(p.curnot(token.COMMA) && !(p.peekis(token.OPENING_CURLY)), newErr(p.tok.Line, p.tok.Col,
		"missing comma between return types in function declaration '%s'", fnName)) {
		return nil
	}
	type_ := frt.Tok.Literal
	if frt.IsList {
		type_ = "listof " + frt.TypeOfList.Literal
	}
	if p.errif(p.curis(token.NEWLINE) && p.peekis(token.COMMA), newErr(p.tok.Line, p.tok.Col,
		"illegal newline after return type '%s' in function declaration '%s'", type_, fnName)) {
		return nil
	}
	stoppedAtComma := p.curis(token.COMMA)
	redundantComma := stoppedAtComma && (p.peekis(token.OPENING_CURLY) || p.peekis(token.NEWLINE) && p.peekN(2).Type == token.OPENING_CURLY)
	if p.errif(redundantComma, newErr(p.tok.Line,
		p.tok.Col, "redundant comma after return type '%s' in function declaration '%s'", type_, fnName)) {
		return nil
	}
	p.moveif(p.curis(token.COMMA))
	p.moveif(p.curis(token.NEWLINE))
	return frt
}

func (p *Parser) parseFunctionReturnTypes(fnName string) (int, []ast.FunctionReturnType) {
	// current token is '->'
	rtx := []ast.FunctionReturnType{}
	p.move()
	for p.curnot(token.OPENING_CURLY) {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: missing function body in function declaration '%s'", fnName)) {
			return -1, nil
		}
		t := p.parseFunctionReturnType(fnName)
		if t == nil {
			return -1, nil
		}
		rtx = append(rtx, *t)
	}
	return len(rtx), rtx
}

func (p *Parser) parseFunctionDeclarationStatement() *ast.FunctionDeclarationStatement {
	// current token is token.FUN
	fds := &ast.FunctionDeclarationStatement{Tok: p.tok}
	p.move()
	line, col := p.tok.Line, p.tok.Col
	isStmt := false
	if p.errif(p.curnot(token.IDENT), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' as function name where an identifier was expected", p.tok.Literal)) {
		return nil
	}
	if name := p.parseIdentifier(isStmt); name != nil {
		fds.Name = name
	}
	if p.errif(fds.Name == nil, newErr(line, col, "missing function name")) {
		return nil
	}
	if p.errif(p.curnot(token.OPENING_PAREN), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s', where a '(' was expected in function declaration '%s'",
		p.tok.Literal, fds.Name)) {
		return nil
	}
	p.move() // either a type, or a )
	if p.curis(token.CLOSING_PAREN) {
		p.move()
		goto noparam
	}
	p.moveif(p.curis(token.NEWLINE)) // allow newline after '('
	fds.Params = p.parseFunctionParams(fds.Name.String())
	// we could've returned an error in parseFunctionParams, but I decided to just return nil in case of an error.
	// this may cause bugs when we define `res` as `var res []ast.FunctionParameter`, in which case, res is nil.
	// there could be no parameters in declaration, and we just return res as is, then that causes this function
	// to return nil. Currently, parseFunctionParams returns nil, only when there is an error.
	if fds.Params == nil {
		return nil
	}
	// current token is either ->, or {
	// if it is '->', then that means, there is at least one return type.
noparam:
	if p.curis(token.ARROW) {
		count, types := p.parseFunctionReturnTypes(fds.Name.String())
		fds.ReturnCount = count
		fds.ReturnTypes = types
	}
	if fds.ReturnCount < 0 {
		return nil
	}
	// current token is '{'
	if p.errif(p.curnot(token.OPENING_CURLY), newErr(p.tok.Line, p.tok.Col,
		"unexpected token '%s' in function '%s', where a '{' was expected as the beginning of body block",
		p.tok.Literal, fds.Name)) {
		return nil
	}
	p.move()
	for p.curnot(token.CLOSING_CURLY) {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: unclosed body of function '%s'", fds.Name)) {
			return nil
		}
		if stmt := p.parseStatement(); stmt != nil {
			fds.Stmts = append(fds.Stmts, stmt)
		}
	}
	p.move() // skip '}'
	return fds
}

func (p *Parser) parseDatatypeLiteralField(literal string) *ast.DataypeLiteralField {
	// current token is token.IDENT.
	dlf := &ast.DataypeLiteralField{}
	isStmt := false
	line, col := p.tok.Line, p.tok.Col
	if p.errif(p.curnot(token.IDENT), newErr(line, col,
		"unexpected token '%s' in datatype literal '%s', where an identifier was expected", p.tok.Literal, literal)) {
		return nil
	}
	dlf.Name = p.parseIdentifier(isStmt)
	if p.errif(dlf.Name == nil, newErr(line, col, "missing field name in datatype literal '%s'", literal)) {
		return nil
	}
	line, col = p.tok.Line, p.tok.Col
	if p.errif(p.curnot(token.EQUAL), newErr(line, col,
		"unexpected token '%s' after identifier in datatype literal '%s', where an '=' was expected", p.tok.Literal, literal)) {
		return nil
	}
	// don't skip '=', because parseExpr checks p.peek().Type
	if peek := p.peek(); p.errif(!(isExpr(peek.Type)), newErr(line, col,
		"unexpected token '%s' as value to '%s' in datatype literal '%s'", peek.Literal, dlf.Name, literal)) {
		return nil
	}
	dlf.Value = p.parseExpr()
	// is it possible ? I think not...
	if p.errif(dlf.Value == nil, newErr(line, col, "missing value to '%s' field in datatype literal '%s'", dlf.Name, literal)) {
		return nil
	}
	p.eat(token.NEWLINE)
	return dlf
}

func (p *Parser) parseDatatypeLiteral(isStmt bool) *ast.DatatypeLiteral {
	dl := &ast.DatatypeLiteral{Tok: p.tok}
	p.eat(token.NEWLINE)
	p.move()
	p.moveif(p.curis(token.OPENING_CURLY))
	p.eat(token.NEWLINE)
	p.moveif(p.curis(token.OPENING_CURLY))
	p.eat(token.NEWLINE)
	literal := dl.Tok.Literal
	for p.curnot(token.CLOSING_CURLY) {
		if p.errif(p.curis(token.EOF), newErr(p.tok.Line, p.tok.Col,
			"unexpected end-of-file: unclosed datatype literal '%s'", literal)) {
			return nil
		}
		f := p.parseDatatypeLiteralField(literal)
		if f == nil {
			return nil
		}
		dl.Fields = append(dl.Fields, f)
		p.eat(token.NEWLINE)
	}
	p.move()
	if isStmt {
		if p.errif(p.curnot(token.DOT), newErr(p.tok.Line, p.tok.Col,
			"unexpected token '%s' at the end of datatype literal '%s' where a dot was expected", p.tok.Literal, literal)) {
			return nil
		}
		p.move()
	}
	return dl
}
