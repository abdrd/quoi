package analyzer

import (
	"fmt"
	"quoi/ast"
	"strings"
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

func (a *Analyzer) pushErr(err error) {
	a.Errs = append(a.Errs, err.(Err))
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
	return a.typecheck()
}

func (a *Analyzer) typecheck() *IRProgram {
	program := &IRProgram{}
	for _, s := range a.program.Stmts {
		switch s := s.(type) {
		case *ast.PrefixExpr:
			a.errorf(s.Tok.Line, s.Tok.Col, "top-level prefix-expression")
		case *ast.VariableDeclarationStatement:
			ir := a.typecheckVarDecl(s)
			if ir != nil {
				program.Push(ir)
			}
		}
	}
	return program
}

func (a *Analyzer) is(expr ast.Expr, type_ string) error {
	switch expr := expr.(type) {
	case *ast.StringLiteral:
		if type_ == "string" {
			return nil
		}
		return newErr(expr.Typ.Line, expr.Typ.Col, "expected '%s' got 'string'", type_)
	case *ast.IntLiteral:
		if type_ == "int" {
			return nil
		}
		return newErr(expr.Typ.Line, expr.Typ.Col, "expected '%s' got 'int'", type_)
	case *ast.BoolLiteral:
		if type_ == "bool" {
			return nil
		}
		return newErr(expr.Typ.Line, expr.Typ.Col, "expected '%s' got 'bool'", type_)
	case *ast.ListLiteral:
		typeOfList := strings.Split(type_, "-")[1]
		for _, v := range expr.Elems {
			if err := a.is(v, typeOfList); err != nil {
				return err
			}
		}
		return nil
	case *ast.Identifier:
		variable := a.env.GetVar(expr.Tok.Literal)
		if variable == nil {
			return newErr(expr.Tok.Line, expr.Tok.Col, "reference non-existent variable '%s'", expr.Tok.Literal)
		}
		if variable.Type != type_ {
			return newErr(expr.Tok.Line, expr.Tok.Col, "expected '%s' got '%s'", type_, variable.Type)
		}
		return nil
	case *ast.PrefixExpr:
		return a.typecheckPrefExpr(expr, type_)
	}
	panic("is : unknown || " + type_ + " || " + expr.String())
}

func (a *Analyzer) toIrExpr(expr ast.Expr) IRExpression {
	switch expr := expr.(type) {
	case *ast.StringLiteral:
		return &IRString{Value: expr.Val}
	case *ast.IntLiteral:
		return &IRInt{Value: expr.String()}
	case *ast.BoolLiteral:
		return &IRBoolean{Value: expr.String()}
	case *ast.Identifier:
		v := a.env.GetVar(expr.Tok.Literal)
		return &IRVariableReference{Name: v.Name, Type: v.Type, Value: v.Value}
	case *ast.PrefixExpr:
		ir := &IRPrefExpr{Operator: expr.Tok.Literal}
		for _, v := range expr.Args {
			ir.Operands = append(ir.Operands, a.toIrExpr(v))
		}
		return ir
	}
	panic("toIrExpr : unhandled expr " + expr.String())
}

func (a *Analyzer) typecheckPrefExpr(s *ast.PrefixExpr, expectedType string) error {
	if s.Tok.Literal != "not" && s.Tok.Literal != "'" && len(s.Args) < 2 {
		return newErr(s.Tok.Line, s.Tok.Col, "'%s' operator expects at least 2 arguments", s.Tok.Literal)
	}
	switch s.Tok.Literal {
	case "+":
		isStrConcat := a.is(s.Args[0], TypeString) == nil
		isIntAddition := a.is(s.Args[0], TypeInt) == nil
		if isStrConcat {
			for _, v := range s.Args {
				if err := a.is(v, TypeString); err != nil {
					return err
				}
			}
			if expectedType != "string" {
				return newErr(s.Tok.Line, s.Tok.Col, "expected '%s', got 'string'", expectedType)
			}
			return nil
		} else if isIntAddition {
			for _, v := range s.Args {
				if err := a.is(v, TypeInt); err != nil {
					return err
				}
			}
			if expectedType != "int" {
				return newErr(s.Tok.Line, s.Tok.Col, "expected '%s', got 'int'", expectedType)
			}
			return nil
		}
		return newErr(s.Tok.Line, s.Tok.Col, "illegal type of expression passed to '+' operator")
	case "-", "/", "*":
		for _, v := range s.Args {
			if err := a.is(v, TypeInt); err != nil {
				return err
			}
		}
		return nil
	}
	panic("unhandled operator " + s.Tok.Literal)
}

func (a *Analyzer) typecheckVarDecl(s *ast.VariableDeclarationStatement) *IRVariable {
	if err := a.is(s.Value, s.Tok.Literal); err != nil {
		a.pushErr(err)
		return nil
	}
	return &IRVariable{Name: s.Ident.String(), Type: s.Tok.Literal, Value: a.toIrExpr(s.Value)}
}
