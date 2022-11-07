package sema

import (
	"fmt"
	"log"
	"quoi/ast"
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
			// TODO type-checking here ?
			vd := &varDecl{VariableDeclarationStatement: n}
			r.stackOfScopes.addSymbol(vd)
		case *ast.ListVariableDeclarationStatement:
			lvd := &listDecl{ListVariableDeclarationStatement: n}
			r.stackOfScopes.addSymbol(lvd)
		}
	}
}
