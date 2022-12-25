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

	// state
	seenReturn bool
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
		case *ast.BreakStatement:
			a.errorf(s.Tok.Line, s.Tok.Col, "top-level break statement")
		case *ast.ContinueStatement:
			a.errorf(s.Tok.Line, s.Tok.Col, "top-level continue statement")
		case *ast.PrefixExpr:
			a.errorf(s.Tok.Line, s.Tok.Col, "top-level prefix-expression")
		case *ast.StringLiteral:
			a.errorf(s.Typ.Line, s.Typ.Col, "unused string literal")
		case *ast.IntLiteral:
			a.errorf(s.Typ.Line, s.Typ.Col, "unused integer literal")
		case *ast.BoolLiteral:
			a.errorf(s.Typ.Line, s.Typ.Col, "unused boolean literal")
		case *ast.DatatypeLiteral:
			a.errorf(s.Tok.Line, s.Tok.Col, "unused datatype literal")
		case *ast.ReturnStatement:
			a.errorf(s.Tok.Line, s.Tok.Col, "return statement outside a function body")
		default:
			if ir := a.typecheckStatement(s, nil); ir != nil {
				program.Push(ir)
			}
		}
	}
	return program
}

func (a *Analyzer) is(expr ast.Expr, type_ string) error {
	switch expr := expr.(type) {
	case *ast.StringLiteral:
		if type_ == TypeString {
			return nil
		}
		return newErr(expr.Typ.Line, expr.Typ.Col, "expected '%s' got 'string'", type_)
	case *ast.IntLiteral:
		if type_ == TypeInt {
			return nil
		}
		return newErr(expr.Typ.Line, expr.Typ.Col, "expected '%s' got 'int'", type_)
	case *ast.BoolLiteral:
		if type_ == TypeBool {
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
			return newErr(expr.Tok.Line, expr.Tok.Col, "reference to non-existent variable '%s'", expr.Tok.Literal)
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

func (a *Analyzer) typecheckStatement(s ast.Statement, returnWanted *returnWanted) IRStatement {
	switch s := s.(type) {
	case *ast.VariableDeclarationStatement:
		return a.typecheckVarDecl(s)
	case *ast.ListVariableDeclarationStatement:
		return a.typecheckListDecl(s)
	case *ast.IfStatement:
		return a.typecheckIfStmt(s, returnWanted)
	case *ast.DatatypeDeclaration:
		return a.typecheckDatatypeDecl(s)
	case *ast.SubsequentVariableDeclarationStatement:
		return a.typecheckSubseqVarDecl(s)
	case *ast.ReassignmentStatement:
		return a.typecheckReassignment(s)
	case *ast.BlockStatement:
		return a.typecheckBlock(s, returnWanted)
	case *ast.LoopStatement:
		return a.typecheckLoop(s, returnWanted)
	case *ast.FunctionDeclarationStatement:
		return a.typecheckFunDecl(s)
	}
	return nil
}

func (a *Analyzer) toIrExpr(expr ast.Expr, typeOfList ...string) IRExpression {
	// IMPORTANT
	// typeOfList variable is a variadic parameter, because I want to be able to skip it if I want to.
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
	case *ast.ListLiteral:
		if len(typeOfList) != 1 {
			panic("toIrExpr: len(typeOfList) != 1")
		}
		ir := &IRList{Type: typeOfList[0], Length: len(expr.Elems)}
		for _, v := range expr.Elems {
			ir.Value = append(ir.Value, a.toIrExpr(v))
		}
		return ir
	}
	panic("toIrExpr : unhandled expr " + expr.String())
}

func (a *Analyzer) typecheckPrefExpr(s *ast.PrefixExpr, expectedType string) error {
	switch s.Tok.Literal {
	case "+":
		if len(s.Args) < 2 {
			return newErr(s.Tok.Line, s.Tok.Col, "operator '+' needs at least 2 arguments")
		}
		isStrConcat := a.is(s.Args[0], TypeString) == nil
		isIntAddition := a.is(s.Args[0], TypeInt) == nil
		if !(isStrConcat) && !(isIntAddition) {
			return newErr(s.Tok.Line, s.Tok.Col, "operator '+' takes string or int values")
		}
		if isStrConcat {
			for _, v := range s.Args {
				if err := a.is(v, TypeString); err != nil {
					return err
				}
			}
			if expectedType != TypeString {
				return newErr(s.Tok.Line, s.Tok.Col, "expected '%s', got 'string'", expectedType)
			}
			return nil
		}
		for _, v := range s.Args {
			if err := a.is(v, TypeInt); err != nil {
				return err
			}
		}
		if expectedType != TypeInt {
			return newErr(s.Tok.Line, s.Tok.Col, "expected '%s', got 'int'", expectedType)
		}
		return nil
	case "-", "/", "*":
		if len(s.Args) < 2 {
			return newErr(s.Tok.Line, s.Tok.Col, "operator '%s' needs at least 2 arguments", s.Tok.Literal)
		}
		for _, v := range s.Args {
			if err := a.is(v, TypeInt); err != nil {
				return err
			}
		}
		if expectedType != TypeInt {
			return newErr(s.Tok.Line, s.Tok.Col, "expected '%s', got 'int'", expectedType)
		}
		return nil
	case "lt", "lte", "gt", "gte":
		if len(s.Args) < 2 {
			return newErr(s.Tok.Line, s.Tok.Col, "operator '%s' needs exactly 2 arguments", s.Tok.Literal)
		}
		for _, v := range s.Args {
			if err := a.is(v, TypeInt); err != nil {
				return err
			}
		}
		if expectedType != TypeBool {
			return newErr(s.Tok.Line, s.Tok.Col, "expected '%s', got 'bool'", expectedType)
		}
		return nil
	case "not":
		if len(s.Args) != 1 {
			return newErr(s.Tok.Line, s.Tok.Col, "operator 'not' needs exactly 1 argument")
		}
		if err := a.is(s.Args[0], TypeBool); err != nil {
			return err
		}
		if expectedType != TypeBool {
			return newErr(s.Tok.Line, s.Tok.Col, "expected '%s', got 'bool'", expectedType)
		}
		return nil
	case "=":
		if len(s.Args) != 2 {
			return newErr(s.Tok.Line, s.Tok.Col, "operator '=' needs exactly 2 arguments")
		}
		isStrEq := a.is(s.Args[0], TypeString) == nil
		isIntEq := a.is(s.Args[0], TypeInt) == nil
		if !(isStrEq) && !(isIntEq) {
			return newErr(s.Tok.Line, s.Tok.Col, "operator '=' takes string or int values")
		}
		if isStrEq {
			if err := a.is(s.Args[1], TypeString); err != nil {
				return err
			}
			if expectedType != TypeBool {
				return newErr(s.Tok.Line, s.Tok.Col, "expected '%s', got 'bool'", expectedType)
			}
			return nil
		}
		if err := a.is(s.Args[1], TypeInt); err != nil {
			return err
		}
		if expectedType != TypeBool {
			return newErr(s.Tok.Line, s.Tok.Col, "expected '%s', got 'bool'", expectedType)
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
	ir := &IRVariable{Name: s.Ident.String(), Type: s.Tok.Literal, Value: a.toIrExpr(s.Value)}
	a.env.AddVar(ir.Name, ir)
	return ir
}

func (a *Analyzer) typecheckListDecl(s *ast.ListVariableDeclarationStatement) *IRVariable {
	listTyp := TypeList(s.Typ.Literal)
	if err := a.is(s.List, listTyp); err != nil {
		a.pushErr(err)
		return nil
	}
	ir := &IRVariable{Name: s.Name.String(), Type: listTyp, Value: a.toIrExpr(s.List, s.Typ.Literal)}
	a.env.AddVar(ir.Name, ir)
	return ir
}

func (a *Analyzer) funAndDatatypeDeclOnlyInGlobalScope(s ast.Statement) error {
	switch s := s.(type) {
	case *ast.FunctionDeclarationStatement:
		return newErr(s.Tok.Line, s.Tok.Col, "function declarations are only allowed at global scope")
	case *ast.DatatypeDeclaration:
		return newErr(s.Tok.Line, s.Tok.Col, "datatype declarations are only allowed at global scope")
	}
	return nil
}

func (a *Analyzer) returnCountAndTypeMustMatch(v ast.Statement, returnWanted *returnWanted) error {
	if r, ok := v.(*ast.ReturnStatement); ok {
		a.seenReturn = true
		if err := returnWanted.checkCountError(r.Tok.Line, r.Tok.Col, len(r.ReturnValues)); err != nil {
			return err
		}
		if err := returnWanted.checkTypeError(a, r.Tok.Line, r.Tok.Col, r.ReturnValues); err != nil {
			return err
		}
	}
	return nil
}

func (a *Analyzer) typecheckIfStmt(s *ast.IfStatement, returnWanted *returnWanted) *IRIf {
	a.env.EnterScope()
	defer a.env.ExitScope()
	if err := a.is(s.Cond, TypeBool); err != nil {
		a.pushErr(err)
		return nil
	}
	ir := &IRIf{Cond: a.toIrExpr(s.Cond)}
	for _, v := range s.Stmts {
		if err := a.returnCountAndTypeMustMatch(v, returnWanted); err != nil {
			a.pushErr(err)
			return nil
		}
		if err := a.funAndDatatypeDeclOnlyInGlobalScope(v); err != nil {
			a.pushErr(err)
			return nil
		}
		if stmtIr := a.typecheckStatement(v, returnWanted); stmtIr != nil {
			ir.Block = append(ir.Block, stmtIr)
		}
	}
	if s.Alternative != nil {
		ir.Alternative = a.typecheckElseIfStmt(s.Alternative, returnWanted)
	}
	if s.Default != nil {
		ir.Default = a.typecheckElseStmt(s.Default, returnWanted)
	}
	return ir
}

func (a *Analyzer) typecheckElseStmt(s *ast.ElseStatement, returnWanted *returnWanted) *IRElse {
	ir := &IRElse{}
	for _, v := range s.Stmts {
		if err := a.returnCountAndTypeMustMatch(v, returnWanted); err != nil {
			a.pushErr(err)
			return nil
		}
		if err := a.funAndDatatypeDeclOnlyInGlobalScope(v); err != nil {
			a.pushErr(err)
			return nil
		}
		if stmtIr := a.typecheckStatement(v, returnWanted); stmtIr != nil {
			ir.Block = append(ir.Block, stmtIr)
		}
	}
	return ir
}

// this is going to be mostly the same as typecheckIfStmt, but I don't want to create workarounds to prevent
// entering a new scope when using typecheckIfStmt to typecheck an elseif statement.
func (a *Analyzer) typecheckElseIfStmt(s *ast.IfStatement, returnWanted *returnWanted) *IRElseIf {
	if err := a.is(s.Cond, TypeBool); err != nil {
		a.pushErr(err)
		return nil
	}
	ir := &IRElseIf{Cond: a.toIrExpr(s.Cond)}
	for _, v := range s.Stmts {
		if err := a.returnCountAndTypeMustMatch(v, returnWanted); err != nil {
			a.pushErr(err)
			return nil
		}
		if err := a.funAndDatatypeDeclOnlyInGlobalScope(v); err != nil {
			a.pushErr(err)
			return nil
		}
		if stmtIr := a.typecheckStatement(v, returnWanted); stmtIr != nil {
			ir.Block = append(ir.Block, stmtIr)
		}
	}
	if s.Alternative != nil {
		ir.Alternative = a.typecheckElseIfStmt(s.Alternative, returnWanted)
	}
	if s.Default != nil {
		ir.Default = a.typecheckElseStmt(s.Default, returnWanted)
	}
	return ir
}

func (a *Analyzer) typecheckDatatypeDecl(s *ast.DatatypeDeclaration) *IRDatatype {
	ir := &IRDatatype{Name: s.Name.String(), FieldCount: len(s.Fields)}
	fields := map[string]bool{} // to prevent two fields with the same name
	for _, v := range s.Fields {
		fieldName := v.Ident.String()
		if fields[fieldName] {
			a.errorf(v.Tok.Line, v.Tok.Col, "duplicate field name '%s' in datatype '%s'", fieldName, ir.Name)
			return nil
		}
		ir.Fields = append(ir.Fields, IRDatatypeField{Type: v.Tok.Literal, Name: v.Ident.String()})
		fields[fieldName] = true
	}
	return ir
}

func (a *Analyzer) typecheckSubseqVarDecl(s *ast.SubsequentVariableDeclarationStatement) *IRSubseq {
	ir := &IRSubseq{}
	lenTypes, lenNames, lenValues := len(s.Types), len(s.Names), len(s.Values)
	// lenTypes, and lenNames are guaranteed -by the parser- to be equal.
	if lenTypes != lenValues || lenNames != lenValues {
		a.errorf(s.Tok.Line, s.Tok.Col, "missing value")
		return nil
	}
	for _, v := range s.Names {
		ir.Names = append(ir.Names, v.String())
	}
	for _, v := range s.Types {
		if v.IsList {
			ir.Types = append(ir.Types, TypeList(v.TypeOfList.Literal))
			continue
		}
		ir.Types = append(ir.Types, v.Tok.Literal)
	}
	for i := 0; i < len(s.Values); i++ {
		typ, name := ir.Types[i], ir.Names[i]
		if err := a.is(s.Values[i], typ); err != nil {
			a.pushErr(err)
			return nil
		}
		valToConv := s.Values[i]
		isList := s.Types[i].IsList
		typOfList := s.Types[i].TypeOfList
		var irExpr IRExpression
		if isList {
			irExpr = a.toIrExpr(valToConv, typOfList.Literal)
		} else {
			irExpr = a.toIrExpr(valToConv)
		}
		ir.Values = append(ir.Values, irExpr)
		if err := a.env.AddVar(name, &IRVariable{Name: name, Type: typ, Value: irExpr}); err != nil {
			a.errorf(s.Tok.Line, s.Tok.Col, err.Error())
			return nil
		}
	}
	return ir
}

func (a *Analyzer) typecheckReassignment(s *ast.ReassignmentStatement) *IRReassigment {
	ir := &IRReassigment{Name: s.Ident.String()}
	typOfOldVal := a.env.GetVar(ir.Name).Type
	if err := a.is(s.NewValue, typOfOldVal); err != nil {
		a.pushErr(err)
		return nil
	}
	ir.NewValue = a.toIrExpr(s.NewValue)
	if err := a.env.UpdateVar(ir.Name, ir.NewValue); err != nil {
		a.errorf(s.Tok.Line, s.Tok.Col, err.Error())
		return nil
	}
	return ir
}

func (a *Analyzer) illegalFunDatatypeBreakAndContinueIn(what string, s ast.Statement) error {
	if err := a.funAndDatatypeDeclOnlyInGlobalScope(s); err != nil {
		return err
	}
	switch s := s.(type) {
	case *ast.BreakStatement:
		return newErr(s.Tok.Line, s.Tok.Col, "break is not allowed inside %ss", what)
	case *ast.ContinueStatement:
		return newErr(s.Tok.Line, s.Tok.Col, "continue is not allowed inside %ss", what)
	}
	return nil
}

func (a *Analyzer) typecheckBlock(s *ast.BlockStatement, returnWanted *returnWanted) *IRBlock {
	a.env.EnterScope()
	defer a.env.ExitScope()
	ir := &IRBlock{}
	for _, v := range s.Stmts {
		if err := a.returnCountAndTypeMustMatch(v, returnWanted); err != nil {
			a.pushErr(err)
			return nil
		}
		if err := a.illegalFunDatatypeBreakAndContinueIn("block", v); err != nil {
			a.pushErr(err)
			return nil
		}
		if stmt := a.typecheckStatement(v, returnWanted); stmt != nil {
			ir.Stmts = append(ir.Stmts, stmt)
		}
	}
	return ir
}

func (a *Analyzer) typecheckLoop(s *ast.LoopStatement, returnWanted *returnWanted) *IRLoop {
	ir := &IRLoop{}
	if err := a.is(s.Cond, TypeBool); err != nil {
		a.pushErr(err)
		return nil
	}
	ir.Cond = a.toIrExpr(s.Cond)
	for _, v := range s.Stmts {
		if err := a.returnCountAndTypeMustMatch(v, returnWanted); err != nil {
			a.pushErr(err)
			return nil
		}
		if err := a.funAndDatatypeDeclOnlyInGlobalScope(v); err != nil {
			a.pushErr(err)
			return nil
		}
		if stmt := a.typecheckStatement(v, returnWanted); stmt != nil {
			ir.Stmts = append(ir.Stmts, stmt)
		}
	}
	return ir
}

func (a *Analyzer) typecheckFunDecl(s *ast.FunctionDeclarationStatement) *IRFunction {
	a.env.EnterScope()
	defer a.env.ExitScope()
	defer func() { a.seenReturn = false }()
	ir := &IRFunction{Name: s.Name.String(), TakesCount: len(s.Params), ReturnsCount: len(s.ReturnTypes)}
	for _, v := range s.Params {
		if v.IsList {
			ir.Takes = append(ir.Takes, TypeList(v.TypeOfList.Literal))
		} else {
			ir.Takes = append(ir.Takes, v.Tok.Literal)
		}
		param := &IRVariable{Name: v.Name.String()} // value is non-significant.
		if v.IsList {
			param.Type = TypeList(v.TypeOfList.Literal)
		} else {
			param.Type = v.Tok.Literal
		}
		if err := a.env.AddVar(param.Name, param); err != nil {
			a.errorf(v.Tok.Line, v.Tok.Col, "duplicate parameter '%s' name in function '%s'", param.Name, ir.Name)
			return nil
		}
	}
	for _, v := range s.ReturnTypes {
		if v.IsList {
			ir.Returns = append(ir.Returns, TypeList(v.TypeOfList.Literal))
			continue
		}
		ir.Returns = append(ir.Returns, v.Tok.Literal)
	}
	for _, v := range s.Stmts {
		returnWanted := &returnWanted{count: ir.ReturnsCount, types: ir.Returns}
		if r, ok := v.(*ast.ReturnStatement); ok {
			// this is a return statement
			lenReturn := len(r.ReturnValues)
			if ir.ReturnsCount == 0 && lenReturn > 0 {
				// the function wasn't supposed to return anything, but we have got a return statement
				// here.
				a.errorf(r.Tok.Line, r.Tok.Col, "unwanted return value in function '%s'", ir.Name)
				return nil
			}
			if err := returnWanted.checkCountError(r.Tok.Line, r.Tok.Col, lenReturn); err != nil {
				a.pushErr(err)
				return nil
			}
			// equal
			for i, v := range r.ReturnValues {
				if err := a.is(v, ir.Returns[i]); err != nil {
					a.pushErr(err)
					return nil
				}
			}
			a.seenReturn = true
		}
		if err := a.illegalFunDatatypeBreakAndContinueIn("function declaration", v); err != nil {
			a.pushErr(err)
			return nil
		}
		if stmt := a.typecheckStatement(v, returnWanted); stmt != nil {
			ir.Block = append(ir.Block, stmt)
		}
	}
	if !(a.seenReturn) && ir.ReturnsCount > 0 {
		a.errorf(s.Tok.Line, s.Tok.Col, "missing return statement")
	}
	return ir
}

type returnWanted struct {
	count int
	types []string
}

func (r *returnWanted) checkCountError(line, col uint, gotCount int) error {
	if r.count < gotCount {
		return newErr(line, col, "excessive return value (want=%d got=%d)", r.count, gotCount)
	} else if r.count > gotCount {
		return newErr(line, col, "missing return value (want=%d got=%d)", r.count, gotCount)
	}
	return nil
}

func (r *returnWanted) checkTypeError(a *Analyzer, line, col uint, vals []ast.Expr) error {
	// counts are equal. guaranteed.
	for i := 0; i < len(vals); i++ {
		if err := a.is(vals[i], r.types[i]); err != nil {
			return err
		}
	}
	return nil
}
