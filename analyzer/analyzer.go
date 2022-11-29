package analyzer

import (
	"fmt"
	"quoi/ast"
	"quoi/token"
)

type Err struct {
	Line, Column uint
	Msg          string
}

type Analyzer struct {
	program *ast.Program
	curExpr *ast.Expr
	env     *ScopeStack
	Errs    []Err
}

func New(program *ast.Program) *Analyzer {
	return &Analyzer{program: program, curExpr: nil, env: NewScopeStack()}
}

func fnParamTypeRepr(param ast.FunctionParameter) string {
	res := ""
	if param.IsList {
		res = "list-"
		res += param.TypeOfList.Literal
		return res
	}
	res += param.Tok.Literal
	return res
}

func fnReturnTypeRepr(ret ast.FunctionReturnType) string {
	res := ""
	if ret.IsList {
		res = "list-"
		res += ret.TypeOfList.Literal
		return res
	}
	res += ret.Tok.Literal
	return res
}

func (a *Analyzer) errorf(line, col uint, msgf string, args ...interface{}) {
	a.Errs = append(a.Errs, Err{
		Line:   line,
		Column: col,
		Msg:    fmt.Sprintf(msgf, args...),
	})
}

// first pass
func (a *Analyzer) registerFunctionsAndDatatypes() {
	for _, s := range a.program.Stmts {
		switch s := s.(type) {
		case *ast.FunctionDeclarationStatement:
			if err := a.registerFuncSignature(s); err != nil {
				a.errorf(s.Tok.Line, s.Tok.Col, err.Error())
			}
		case *ast.DatatypeDeclaration:
			if err := a.registerDatatype(s); err != nil {
				a.errorf(s.Tok.Line, s.Tok.Col, err.Error())
			}
		}
	}
}

func (a *Analyzer) registerFuncSignature(s *ast.FunctionDeclarationStatement) error {
	ir := &IRFunction{Name: s.Name.String(), TakesCount: len(s.Params), ReturnsCount: len(s.ReturnTypes)}
	for _, v := range s.Params {
		ir.Takes = append(ir.Takes, fnParamTypeRepr(v))
	}
	for _, v := range s.ReturnTypes {
		ir.Returns = append(ir.Returns, fnReturnTypeRepr(v))
	}
	return a.env.AddFunc(ir)
}

func (a *Analyzer) registerDatatype(s *ast.DatatypeDeclaration) error {
	ir := &IRDatatype{Name: s.Name.String(), FieldCount: len(s.Fields)}
	for _, v := range s.Fields {
		field := IRDatatypeField{Type: v.Tok.Literal, Name: v.Ident.String()}
		ir.Fields = append(ir.Fields, field)
	}
	return a.env.AddDatatype(ir)
}

func (a *Analyzer) Analyze() *IRProgram {
	a.registerFunctionsAndDatatypes()
	program := &IRProgram{}
	for _, s := range a.program.Stmts {
		switch s := s.(type) {
		case *ast.VariableDeclarationStatement:
			if v := a.typecheckVarDecl(s); v != nil {
				program.Push(v)
			}
		}
	}
	return nil
}

func (a *Analyzer) typecheck(expr ast.Expr, expectedType string) bool {
	switch expr := expr.(type) {
	case *ast.IntLiteral:
		return expectedType == "int"
	case *ast.BoolLiteral:
		return expectedType == "bool"
	case *ast.StringLiteral:
		return expectedType == "string"
	case *ast.PrefixExpr:
		switch expr.Tok.Type {
		case token.ADD:
			if len(expr.Args) < 2 {
				a.errorf(expr.Tok.Line, expr.Tok.Col, "not enough arguments to '+' operator")
				return false
			}
			_, isint := expr.Args[0].(*ast.IntLiteral)
			_, isstr := expr.Args[0].(*ast.StringLiteral)
			if !(isint) && !(isstr) {
				// TODO return an error
				return false
			}
			expect := "int"
			if isstr {
				expect = "string"
			}
			for _, v := range expr.Args {
				if ok := a.typecheck(v, expect); !(ok) {
					return false
				}
			}
			return true
			// more to come ...
		}
		// more to come ...
	}
	return false
}

func (a *Analyzer) typecheckVarDecl(s *ast.VariableDeclarationStatement) *IRVariable {
	//v := &IRVariable{Name: s.Ident.String(), Type: s.Tok.Literal, Value: s.Value.String()}
	panic("typecheckVarDecl UNIMPLEMENTED")
}
