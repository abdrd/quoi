package sema

import (
	"fmt"
	"quoi/ast"
	"quoi/token"
)

type Err struct {
	Line, Col uint
	Msg       string
}

type Typechecker struct {
	program *ast.Program
	stack   *scopeStack
	Errs    []Err
}

func NewTypechecker(program *ast.Program) *Typechecker {
	return &Typechecker{program: program, stack: newScopeStack()}
}

func (t *Typechecker) errorf(line, col uint, msgf string, args ...interface{}) {
	t.Errs = append(t.Errs, Err{
		Line: line,
		Col:  col,
		Msg:  fmt.Sprintf(msgf, args...),
	})
}

func (t *Typechecker) Typecheck() *CheckedProgram {
	for _, n := range t.program.Stmts {
		switch n := n.(type) {
		case *ast.BreakStatement, *ast.ContinueStatement, *ast.ElseStatement:
			t.warnTopLevel(n)
		case *ast.IntLiteral, *ast.StringLiteral, *ast.BoolLiteral, *ast.DatatypeLiteral, *ast.PrefixExpr:
			t.warnUnusedExpr(n)
		case *ast.VariableDeclarationStatement:
			t.checkVarDecl(n)
		}
	}
}

func (t *Typechecker) warnTopLevel(n ast.Node) {
	brk, isbrk := n.(*ast.BreakStatement)
	cont, iscont := n.(*ast.ContinueStatement)
	else_, iselse := n.(*ast.ElseStatement)
	var (
		token          = "UNHANDLED (TOP LEVEL)"
		col, line uint = 0, 1
	)
	// TODO refactor
	if isbrk {
		token = "break"
		col, line = brk.Tok.Col, brk.Tok.Line
	} else if iscont {
		token = "continue"
		col, line = cont.Tok.Col, cont.Tok.Line
	} else if iselse {
		token = "else"
		col, line = else_.Tok.Col, else_.Tok.Line
	}
	t.errorf(line, col, "forbidden top-level '%s' statement", token)
}

func (t *Typechecker) warnUnusedExpr(n ast.Node) {
	// TODO refactor
	// reflection ?
	int_, isint := n.(*ast.IntLiteral)
	string_, isstr := n.(*ast.StringLiteral)
	bool_, isbool := n.(*ast.BoolLiteral)
	datat, isdatat := n.(*ast.DatatypeLiteral)
	prefexpr, isprefexpr := n.(*ast.PrefixExpr)
	var (
		token          = "UNHANDLED (UNUSED EXPR)"
		line, col uint = 0, 1
	)
	if isint {
		token = "integer"
		line, col = int_.Typ.Line, int_.Typ.Col
	} else if isstr {
		token = "string"
		line, col = string_.Typ.Line, string_.Typ.Col
	} else if isbool {
		token = "boolean"
		line, col = bool_.Typ.Line, bool_.Typ.Col
	} else if isdatat {
		token = "data type"
		line, col = datat.Tok.Line, datat.Tok.Col
	} else if isprefexpr {
		token = "prefix expression"
		line, col = prefexpr.Tok.Line, prefexpr.Tok.Col
	}
	t.errorf(line, col, "unused %s literal", token)
}

func (t *Typechecker) typeOf(tok token.Token) VarType {
	lc := lineColStruct{Line: tok.Line, Col: tok.Col}
	switch tok.Type {
	case token.INTKW:
		return TypeInt{lc}
	case token.STRINGKW:
		return TypeString{lc}
	case token.BOOLKW:
		return TypeBool{lc}
	case token.DATATYPE:
		return TypeDatatype{lineColStruct: lc, Datatype: tok.Literal}
	default:
		panic("typeOf: UNHANDLED : " + tok.Literal)
	}
}

func (t *Typechecker) exprOf(expr ast.Expr) CheckedExpr {
	switch expr := expr.(type) {
	case *ast.IntLiteral:
		return &IntExpr{lineColStruct: lineColStruct{Line: expr.Typ.Line, Col: expr.Typ.Col}}
	case *ast.StringLiteral:
		return &StringExpr{lineColStruct: lineColStruct{Line: expr.Typ.Line, Col: expr.Typ.Col}}
	case *ast.BoolLiteral:
		return &BoolExpr{lineColStruct: lineColStruct{Line: expr.Typ.Line, Col: expr.Typ.Col}}
	default:
		panic("exprOf: UNHANDLED : " + expr.String())
	}
}

func (t *Typechecker) checkVarDecl(decl *ast.VariableDeclarationStatement) /*no return for now */ {
	cv := CheckedVarDecl{Name: decl.Ident}
	cv.Type = t.typeOf(decl.Tok)
	cv.Value = t.exprOf(decl.Value)
	switch t := cv.Type.(type) {
	case TypeInt:
		if _, ok := cv.Value.(*IntExpr); !(ok) {
			//
		}
	}
}
