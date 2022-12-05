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

func (e Err) Error() string { return e.Msg }

func newErr(line, col uint, msgf string, args ...interface{}) Err {
	return Err{
		Line:   line,
		Column: col,
		Msg:    fmt.Sprintf(msgf, args...),
	}
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

const (
	TypeInt    = "int"
	TypeString = "string"
	TypeBool   = "bool"
)

var (
	TypeDatatype = func(dt string) string {
		return dt
	}
	TypeList = func(listType string) string {
		return "list-" + listType
	}
)

func fnParamTypeRepr(param ast.FunctionParameter) string {
	res := ""
	if param.IsList {
		return TypeList(param.TypeOfList.Literal)
	}
	res += param.Tok.Literal
	return res
}

func fnReturnTypeRepr(ret ast.FunctionReturnType) string {
	res := ""
	if ret.IsList {
		return TypeList(ret.TypeOfList.Literal)
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
	return program
}

func (a *Analyzer) typecheckExpr(expr ast.Expr, expectedType string) error {
	switch expr := expr.(type) {
	case *ast.IntLiteral:
		if ok := expectedType == TypeInt; !(ok) {
			return newErr(expr.Typ.Line, expr.Typ.Col, "expected %s, not int", expectedType)
		}
		return nil
	case *ast.BoolLiteral:
		if ok := expectedType == TypeBool; !(ok) {
			return newErr(expr.Typ.Line, expr.Typ.Col, "expected %s, not bool", expectedType)
		}
		return nil
	case *ast.StringLiteral:
		if ok := expectedType == TypeString; !(ok) {
			return newErr(expr.Typ.Line, expr.Typ.Col, "expected %s, not string", expectedType)
		}
		return nil
	case *ast.PrefixExpr:
		if err := a.assurePrefExprReturnType(expr, expectedType); err != nil {
			return err
		}
		return a.typecheckOperator(expr)
	}
	panic("typecheckExpr UNIMPLEMENTED " + expr.String() + expectedType)
	//return nil
}

func (a *Analyzer) typecheckOperator(expr *ast.PrefixExpr) error {
	switch expr.Tok.Type {
	case token.ADD, token.EQUAL:
		if len(expr.Args) < 2 {
			return newErr(expr.Tok.Line, expr.Tok.Col, "not enough arguments to '%s' operator", token.PrefixExprName(expr.Tok.Type))
		}
		err := a.typecheckExpr(expr.Args[0], TypeInt)
		isInt := err == nil
		if isInt {
			// expect all args to be of type 'int'
			for _, arg := range expr.Args {
				if pref, isPref := arg.(*ast.PrefixExpr); isPref {
					if err := a.assurePrefExprReturnType(pref, TypeInt); err != nil {
						return err
					}
				}
				if err := a.typecheckExpr(arg, TypeInt); err != nil {
					return err
				}
			}
			return nil
		}
		err = a.typecheckExpr(expr.Args[0], TypeString)
		isStr := err == nil
		if isStr {
			// expect all args to be of type 'string'
			for _, arg := range expr.Args {
				if pref, isPref := arg.(*ast.PrefixExpr); isPref {
					if err := a.assurePrefExprReturnType(pref, TypeString); err != nil {
						return err
					}
				}
				if err := a.typecheckExpr(arg, TypeString); err != nil {
					return err
				}
			}
			return nil
		}
		return newErr(expr.Tok.Line, expr.Tok.Col, "illegal type of operand type for '%s' operator", token.PrefixExprName(expr.Tok.Type))
	case token.MINUS, token.MUL, token.DIV, token.LT, token.LTE, token.GT, token.GTE:
		if len(expr.Args) < 2 {
			return newErr(expr.Tok.Line, expr.Tok.Col, "not enough arguments to '%s' operator", token.PrefixExprName(expr.Tok.Type))
		}
		if err := a.typecheckExpr(expr.Args[0], TypeInt); err != nil {
			return newErr(expr.Tok.Line, expr.Tok.Col, "expected an 'int' as operand type for '%s' operator", token.PrefixExprName(expr.Tok.Type))
		}
		for _, arg := range expr.Args {
			if err := a.typecheckExpr(arg, TypeInt); err != nil {
				return err
			}
		}
		return nil
	case token.AND, token.OR:
		if len(expr.Args) < 2 {
			return newErr(expr.Tok.Line, expr.Tok.Col, "not enough arguments to '%s' operator", token.PrefixExprName(expr.Tok.Type))
		}
		if err := a.typecheckExpr(expr.Args[0], TypeBool); err != nil {
			return newErr(expr.Tok.Line, expr.Tok.Col, "expected a 'bool' as operand type for '%s' operator", token.PrefixExprName(expr.Tok.Type))
		}
		for _, arg := range expr.Args {
			if err := a.typecheckExpr(arg, TypeBool); err != nil {
				return err
			}
		}
		return nil
	case token.NOT:
		if len(expr.Args) != 1 {
			return newErr(expr.Tok.Line, expr.Tok.Col, "'not' operator needs exactly one (1) argument")
		}
		return a.typecheckExpr(expr.Args[0], TypeBool)
	}
	return nil
}

func (a *Analyzer) assurePrefExprReturnType(expr *ast.PrefixExpr, retType string) error {
	for _, arg := range expr.Args {
		if err := a.typecheckExpr(arg, retType); err != nil {
			return err
		}
	}
	return nil
}

func (a *Analyzer) typecheckVarDecl(s *ast.VariableDeclarationStatement) *IRVariable {
	v := &IRVariable{Name: s.Ident.String(), Type: s.Tok.Literal, Value: s.Value.String()}
	if err := a.typecheckExpr(s.Value, v.Type); err != nil {
		a.Errs = append(a.Errs, err.(Err))
		return nil
	}
	return v
}
