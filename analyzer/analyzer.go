package analyzer

import (
	"fmt"
	"quoi/ast"
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
	ir := &IRDatatype{}
	return a.env.AddDatatype(ir)
}

func (a *Analyzer) Analyze() *IRProgram {
	a.registerFunctionsAndDatatypes()
	return nil
}
