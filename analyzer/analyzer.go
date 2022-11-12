package analyzer

import (
	"fmt"
	"quoi/ast"
)

type Err struct {
	Line, Column uint
	Msg          string
}

func newErr(line, col uint, msgf string, args ...interface{}) Err {
	return Err{Line: line, Column: col, Msg: fmt.Sprintf(msgf, args...)}
}

// implement `error` interface to be able to return nil when we need to return Err.
func (e Err) Error() string {
	return e.Msg
}

type Analyzer struct {
	program    *ast.Program
	scopeStack *ScopeStack
	Errs       []Err

	// -------------- state ------------
	inFunctionDeclaration,
	hasSeenReturn bool
}

func NewAnalyzer(program *ast.Program) *Analyzer {
	return &Analyzer{program: program, scopeStack: NewScopeStack()}
}

func (a *Analyzer) errorf(line, col uint, msgf string, args ...interface{}) {
	a.Errs = append(a.Errs, newErr(line, col, msgf, args...))
}

// first pass adds function declarations to the global scope, and checks for type errors in them.
func (a *Analyzer) FirstPass() {
	for _, n := range a.program.Stmts {
		switch n := n.(type) {
		case *ast.FunctionDeclarationStatement:
			// error is returned to notify the analyzer when to halt and stop proceeding to the next typechecking phase.
			if err := a.typecheckFunctionDeclaration(n); err != nil {
				a.errorf(n.Tok.Line, n.Tok.Col, err.Error())
				return
			}
			if err := a.scopeStack.AddFunc(n.Name.String(), n); err != nil {
				a.errorf(n.Tok.Line, n.Tok.Col, err.Error())
			}
		}
	}
}

func typeOfExpr(expr ast.Expr) string {
	if _, ok := expr.(*ast.StringLiteral); ok {
		return "string"
	} else if _, ok = expr.(*ast.IntLiteral); ok {
		return "int"
	} else if _, ok := expr.(*ast.BoolLiteral); ok {
		return "bool"
	} else if v, ok := expr.(*ast.DatatypeLiteral); ok {
		return v.Tok.Literal
	}
	// function calls(from namespaces), lists
	panic("typeOfExpr: NOT IMPLEMENTED: " + expr.String())
}

func (a *Analyzer) typecheckFunctionDeclaration(decl *ast.FunctionDeclarationStatement) error {
	a.inFunctionDeclaration = true
	// check return statements inside the function body.
	if err := a.typecheckFunctionReturnCounts(decl); err != nil {
		//a.inFunctionDeclaration = false
		return err
	}
	err := a.typecheckFunctionReturnTypes(decl)
	a.inFunctionDeclaration = false
	a.hasSeenReturn = false
	return err
}

func (a *Analyzer) assertReturnStmtReturnCount(stmt *ast.ReturnStatement, returnCount int) error {
	a.hasSeenReturn = true
	rtStmtRtCount := len(stmt.ReturnValues)
	if rtStmtRtCount < returnCount {
		return Err{Line: stmt.Tok.Line, Column: stmt.Tok.Col, Msg: "missing return value"}
	} else if rtStmtRtCount > returnCount {
		return Err{Line: stmt.Tok.Line, Column: stmt.Tok.Col, Msg: "excessive return value"}
	}
	return nil
}

func (a *Analyzer) typecheckFunctionReturnCounts(decl *ast.FunctionDeclarationStatement) (err error) {
	fReturnCount := decl.ReturnCount
	if len(decl.Stmts) == 0 && fReturnCount > 0 {
		return Err{Line: decl.Tok.Line, Column: decl.Tok.Col, Msg: "missing return statement"}
	}
	for _, n := range decl.Stmts {
		switch n := n.(type) {
		case *ast.ReturnStatement:
			if err := a.assertReturnStmtReturnCount(n, fReturnCount); err != nil {
				return err
			}
		case *ast.BlockStatement:
			if err := a.assertBlockStmtReturnCount(n, fReturnCount); err != nil {
				return err
			}
		case *ast.IfStatement:
			if err := a.assertIfStmtReturnCount(n, fReturnCount); err != nil {
				return err
			}
		}
	}
	if a.inFunctionDeclaration && !(a.hasSeenReturn) {
		a.errorf(decl.Tok.Line, decl.Tok.Col, "missing return statement")
	}
	return
}

func (a *Analyzer) assertIfStmtReturnCount(stmt *ast.IfStatement, returnCount int) (err error) {
	for _, s := range stmt.Stmts {
		switch s := s.(type) {
		case *ast.ReturnStatement:
			if err := a.assertReturnStmtReturnCount(s, returnCount); err != nil {
				return err
			}
		case *ast.IfStatement:
			if err := a.assertIfStmtReturnCount(s, returnCount); err != nil {
				return err
			}
		case *ast.BlockStatement:
			if err := a.assertBlockStmtReturnCount(s, returnCount); err != nil {
				return err
			}
		}
	}
	if stmt.Alternative != nil {
		if err := a.assertIfStmtReturnCount(stmt.Alternative, returnCount); err != nil {
			return err
		}
	}
	if stmt.Default != nil {
		if err := a.assertElseStmtReturnCount(stmt.Default, returnCount); err != nil {
			return err
		}
	}
	return
}

func (a *Analyzer) assertElseStmtReturnCount(stmt *ast.ElseStatement, returnCount int) (err error) {
	for _, s := range stmt.Stmts {
		switch s := s.(type) {
		case *ast.ReturnStatement:
			err = a.assertReturnStmtReturnCount(s, returnCount)
			if err != nil {
				return err
			}
		case *ast.IfStatement:
			err = a.assertIfStmtReturnCount(s, returnCount)
			if err != nil {
				return err
			}
		}
	}
	return
}

func (a *Analyzer) assertBlockStmtReturnCount(stmt *ast.BlockStatement, returnCount int) (err error) {
	for _, s := range stmt.Stmts {
		switch s := s.(type) {
		case *ast.ReturnStatement:
			if err := a.assertReturnStmtReturnCount(s, returnCount); err != nil {
				return err
			}
		case *ast.BlockStatement:
			if err := a.assertBlockStmtReturnCount(s, returnCount); err != nil {
				return err
			}
		case *ast.IfStatement:
			if err := a.assertIfStmtReturnCount(s, returnCount); err != nil {
				return err
			}
		}
	}
	return
}

func (a *Analyzer) typecheckFunctionReturnTypes(decl *ast.FunctionDeclarationStatement) (err error) {
	retTypes := decl.ReturnTypes
	for _, s := range decl.Stmts {
		switch s := s.(type) {
		case *ast.ReturnStatement:
			if err := a.assertReturnStmtReturnTypes(s, retTypes); err != nil {
				return err
			}
		case *ast.IfStatement:
			if err := a.assertIfStmtReturnTypes(s, retTypes); err != nil {
				return err
			}
		case *ast.BlockStatement:
			if err := a.assertBlockStmtReturnTypes(s, retTypes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Analyzer) assertReturnStmtReturnTypes(s *ast.ReturnStatement, retTypes []ast.FunctionReturnType) error {
	var mismatchErr = func(expected, got string) error {
		if expected != got {
			return fmt.Errorf("expected '%s' as return type, but got '%s'", expected, got)
		}
		return nil
	}
	for i, v := range s.ReturnValues {
		// TODO lists
		if err := mismatchErr(retTypes[i].Tok.Literal, typeOfExpr(v)); err != nil {
			// TODO add a `Pos()` method to `ast.Expr` interface, and return an `Err` here.
			return err
		}
	}
	return nil
}

func (a *Analyzer) assertIfStmtReturnTypes(s *ast.IfStatement, retTypes []ast.FunctionReturnType) error {
	for _, st := range s.Stmts {
		switch st := st.(type) {
		case *ast.ReturnStatement:
			if err := a.assertReturnStmtReturnTypes(st, retTypes); err != nil {
				return err
			}
		case *ast.IfStatement:
			if err := a.assertIfStmtReturnTypes(st, retTypes); err != nil {
				return err
			}
		case *ast.BlockStatement:
			if err := a.assertBlockStmtReturnTypes(st, retTypes); err != nil {
				return err
			}
		}
	}
	if s.Alternative != nil {
		if err := a.assertIfStmtReturnTypes(s.Alternative, retTypes); err != nil {
			return err
		}
	}
	if s.Default != nil {
		if err := a.assertElseStmtReturnTypes(s.Default, retTypes); err != nil {
			return err
		}
	}
	return nil
}

func (a *Analyzer) assertElseStmtReturnTypes(s *ast.ElseStatement, retTypes []ast.FunctionReturnType) error {
	for _, st := range s.Stmts {
		switch st := st.(type) {
		case *ast.ReturnStatement:
			if err := a.assertReturnStmtReturnTypes(st, retTypes); err != nil {
				return err
			}
		case *ast.IfStatement:
			if err := a.assertIfStmtReturnTypes(st, retTypes); err != nil {
				return err
			}
		case *ast.BlockStatement:
			if err := a.assertBlockStmtReturnTypes(st, retTypes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Analyzer) assertBlockStmtReturnTypes(s *ast.BlockStatement, retTypes []ast.FunctionReturnType) error {
	for _, st := range s.Stmts {
		switch st := st.(type) {
		case *ast.ReturnStatement:
			if err := a.assertReturnStmtReturnTypes(st, retTypes); err != nil {
				return err
			}
		case *ast.IfStatement:
			if err := a.assertIfStmtReturnTypes(st, retTypes); err != nil {
				return err
			}
		case *ast.BlockStatement:
			if err := a.assertBlockStmtReturnTypes(st, retTypes); err != nil {
				return err
			}
		}
	}
	return nil
}
