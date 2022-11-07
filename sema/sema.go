package sema

import (
	"fmt"
	"log"
	"quoi/ast"
	"quoi/token"
)

type Resolver struct {
	program       *ast.Program
	stackOfScopes *scopeStack
}

func NewResolver(program *ast.Program) *Resolver {
	r := &Resolver{program: program}
	r.stackOfScopes = newScopeStack()
	return r
}

func (r *Resolver) errorf(line, col uint, msgf string, args ...interface{}) {
	log.Fatalf("Analysis: line:col=%d:%d  %s\n", line, col, fmt.Sprintf(msgf, args...))
}

func (r *Resolver) errorf2(msgf string, args ...interface{}) {
	log.Fatalf("Analysis: %s\n", fmt.Sprintf(msgf, args...))
}

func (r *Resolver) Resolve() {
	for _, n := range r.program.Stmts {
		switch n := n.(type) {
		case *ast.IntLiteral:
			r.errorf2("unused int literal '%s'", n.String())
		case *ast.StringLiteral:
			r.errorf2("unused string literal '%s'", n.String())
		case *ast.BoolLiteral:
			r.errorf2("unused boolean literal '%s'", n.String())
		case *ast.DatatypeLiteral:
			r.errorf(n.Tok.Line, n.Tok.Col, "unused datatype literal '%s'", n.String())
		case *ast.PrefixExpr:
			r.errorf(n.Tok.Line, n.Tok.Col, "unused prefix expression literal '%s'", n.String())
		case *ast.BreakStatement:
			r.errorf(n.Tok.Line, n.Tok.Col, "top-level break statement")
		case *ast.ContinueStatement:
			r.errorf(n.Tok.Line, n.Tok.Col, "top-level continue statement")
		case *ast.VariableDeclarationStatement:
			// ensure that variable type, and variable's value's type are the same
			r.check_VarDecl(n)
			vd := &varDecl{VariableDeclarationStatement: n}
			r.stackOfScopes.addSymbol(vd)
		case *ast.ListVariableDeclarationStatement:
			lvd := &listDecl{ListVariableDeclarationStatement: n}
			r.stackOfScopes.addSymbol(lvd)
		}
	}
}

func (r *Resolver) check_VarDecl(decl *ast.VariableDeclarationStatement) {
	typ := decl.Tok.Literal
	xprTyp := ""
	switch xpr := decl.Value.(type) {
	case *ast.StringLiteral:
		xprTyp = "string"
	case *ast.IntLiteral:
		xprTyp = "int"
	case *ast.BoolLiteral:
		xprTyp = "bool"
	case *ast.PrefixExpr:
		r.check_PrefixExpr(xpr, typ)
	default:
		if typ != xprTyp {
			r.errorf(decl.Tok.Line, decl.Tok.Col, "mismatched types '%s', and '%s'", typ, xprTyp)
		}
	}
}

func (r *Resolver) check_PrefixExpr(xpr *ast.PrefixExpr, wantTyp string) {
	op := xpr.Tok.Type
	returnTypeOfOp := ""
	switch op {
	// TODO argument counts for operators
	// TODO these cases below are very similar
	case token.ADD, token.MINUS, token.MUL, token.DIV, token.LT, token.LTE, token.GT, token.GTE:
		returnTypeOfOp = "int"
		for _, v := range xpr.Args {
			switch vTyp := v.(type) {
			case *ast.IntLiteral:
				continue
			case *ast.PrefixExpr:
				r.check_PrefixExpr(vTyp, "int")
			default:
				// TODO use errorf
				r.errorf2("invalid '%s' operand for '%s' operator", vTyp.String(), token.PrefixExprName(op))
			}
		}
	case token.AND, token.OR, token.NOT:
		returnTypeOfOp = "bool"
		for _, v := range xpr.Args {
			switch vTyp := v.(type) {
			case *ast.BoolLiteral:
				continue
			case *ast.PrefixExpr:
				r.check_PrefixExpr(vTyp, "bool")
			default:
				// TODO use errorf
				r.errorf2("invalid '%s' operand for '%s' operator", vTyp.String(), token.PrefixExprName(op))
			}
		}
	}
	if returnTypeOfOp != wantTyp {
		r.errorf(xpr.Tok.Line, xpr.Tok.Col, "mismatched types '%s' and '%s'", wantTyp, returnTypeOfOp)
	}
}
