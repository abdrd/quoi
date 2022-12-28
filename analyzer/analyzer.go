package analyzer

import (
	"fmt"
	"quoi/ast"
	"quoi/token"
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

var (
	TypeList = func(listType string) string {
		return "list-" + listType
	}
)

func fnParamTypeRepr(param ast.FunctionParameter) string {
	res := ""
	if param.IsList {
		return "list-" + param.TypeOfList.Literal
	}
	res += param.Tok.Literal
	return res
}

func fnReturnTypeRepr(ret ast.FunctionReturnType) string {
	res := ""
	if ret.IsList {
		return "list-" + ret.TypeOfList.Literal
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

type typeLit string

const (
	TypeString typeLit = "string"
	TypeInt    typeLit = "int"
	TypeBool   typeLit = "bool"
	TypeVoid   typeLit = "void"
	// empty lists are of this type (list-any)
	TypeAny typeLit = "any"
)

var (
	TypeList_ = func(t typeLit) typeLit {
		return typeLit("list-" + string(t))
	}
	TypeDatatype_ = func(dt string) typeLit {
		return typeLit(dt)
	}
)

type Type struct {
	typ       typeLit
	line, col uint
	next      *Type
}

func (t *Type) setNext(ty *Type) {
	if t.next == nil {
		t.next = ty
	}
}

func NewType(typ typeLit, line, col uint) *Type {
	return &Type{typ: typ, line: line, col: col}
}

func NewListType(typ typeLit, line, col uint) *Type {
	return NewType(TypeList_(typ), line, col)
}

func (a *Analyzer) match(expr ast.Expr, t *Type) bool {
	// Ignoring the possibility of *t.Next != nil (for now)
	switch expr := expr.(type) {
	case *ast.StringLiteral:
		return t.typ == TypeString
	case *ast.IntLiteral:
		return t.typ == TypeInt
	case *ast.BoolLiteral:
		return t.typ == TypeBool
	case *ast.ListLiteral:
		// learn the type of list literal
		listType, err := a.infer(expr)
		if err != nil {
			a.pushErr(err)
			return false
		}
		if listType.typ == TypeAny {
			return true
		}
		return listType.typ == t.typ
	case *ast.DatatypeLiteral:
		datatypeType, err := a.infer(expr)
		if err != nil {
			a.pushErr(err)
			return false
		}
		return datatypeType.typ == t.typ
	case *ast.FunctionCall:
		fnType, err := a.infer(expr)
		if err != nil {
			a.pushErr(err)
			return false
		}
		return fnType.typ == t.typ
	case *ast.PrefixExpr:
		prefType, err := a.infer(expr)
		if err != nil {
			a.pushErr(err)
			return false
		}
		return prefType.typ == t.typ
	case *ast.Identifier:
		if a.env.IsFailedVar(expr.Tok.Literal) {
			return true
		}
		decl := a.env.GetVar(expr.String())
		if decl == nil {
			a.errorf(expr.Tok.Line, expr.Tok.Col, "reference to non-existent variable '%s'", expr.String())
			return false
		}
		return typeLit(decl.Type) == t.typ
	}
	panic(fmt.Sprintf("--UNREACHABLE--\n*Analyzer.match: unknown expr '%s'\n", expr.String()))
}

// minimal type inference
func (a *Analyzer) infer(expr ast.Expr) (*Type, error) {
	switch expr := expr.(type) {
	case *ast.StringLiteral:
		return NewType(TypeString, expr.Typ.Line, expr.Typ.Col), nil
	case *ast.IntLiteral:
		return NewType(TypeInt, expr.Typ.Line, expr.Typ.Col), nil
	case *ast.BoolLiteral:
		return NewType(TypeBool, expr.Typ.Line, expr.Typ.Col), nil
	case *ast.ListLiteral:
		if len(expr.Elems) < 1 {
			return NewType(TypeAny, expr.Tok.Line, expr.Tok.Col), nil
		}
		firstElem := expr.Elems[0]
		firstElemType, err := a.infer(firstElem)
		if err != nil {
			return nil, err
		}
		for _, el := range expr.Elems {
			if !(a.match(el, firstElemType)) {
				elType, err := a.infer(el)
				if err != nil {
					return nil, err
				}
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "expected '%s', got '%s' in list literal", firstElemType.typ, elType.typ)
			}
		}
		return NewType(TypeList_(firstElemType.typ), firstElemType.line, firstElemType.col), nil
	case *ast.Identifier:
		decl := a.env.GetVar(expr.String())
		if decl == nil {
			return nil, newErr(expr.Tok.Line, expr.Tok.Col, "reference to non-existent variable '%s'", expr.Tok.Literal)
		}
		return NewType(typeLit(decl.Type), expr.Tok.Line, expr.Tok.Col), nil
	case *ast.DatatypeLiteral:
		datatype := a.env.GetDatatype(expr.Tok.Literal)
		if datatype == nil {
			return nil, newErr(expr.Tok.Line, expr.Tok.Col, "initialization of non-existent datatype '%s'", datatype.Name)
		}
		typ, err := a.infer(expr)
		if err != nil {
			return nil, err
		}
		return typ, nil
	case *ast.FunctionCall:
		fn := a.env.GetFunc(expr.Tok.Literal)
		if fn == nil {
			return nil, newErr(expr.Tok.Line, expr.Tok.Col, "invoking of non-existent function '%s'", fn.Name)
		}
		if fn.ReturnsCount == 0 {
			return NewType(TypeVoid, expr.Tok.Line, expr.Tok.Col), nil
		}
		if fn.ReturnsCount == 1 {
			return NewType(typeLit(fn.Returns[0]), expr.Tok.Line, expr.Tok.Col), nil
		}
		// head of the linked list
		// other return types will follow this.
		//
		// for example:
		// [int] --next--> [string] --next--> [User] --next--> *nil*
		// a very simple singly-linked list
		t := NewType(typeLit(fn.Returns[0]), expr.Tok.Line, expr.Tok.Col)
		// copy t; because when we exit the loop, t will no longer be the first type; instead
		// it will be the last type.
		// And we want to return the first type, so that we can, in the future, follow the type chain.
		tOriginal := t
		for _, r := range fn.Returns {
			t2 := NewType(typeLit(r), expr.Tok.Line, expr.Tok.Col)
			t.setNext(t2)
			t = t2
		}
		return tOriginal, nil
	case *ast.PrefixExpr:
		var expectConsecutive = func(what typeLit) error {
			for _, arg := range expr.Args {
				typ, err := a.infer(arg)
				if err != nil {
					return err
				}
				if typ.typ != what {
					return newErr(expr.Tok.Line, expr.Tok.Col, "expected '%s', got '%s'", what, typ.typ)
				}
			}
			return nil
		}
		switch expr.Tok.Type {
		case token.ADD:
			if len(expr.Args) < 2 {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "operator '+' takes at least two operands")
			}
			typ, err := a.infer(expr.Args[0])
			if err != nil {
				return nil, err
			}
			switch typ.typ {
			case TypeInt:
				if err := expectConsecutive(TypeInt); err != nil {
					return nil, err
				}
				return typ, nil
			case TypeString:
				if err := expectConsecutive(TypeString); err != nil {
					return nil, err
				}
				return typ, nil
			default:
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "invalid type of expression '%s' for '+' operator", typ.typ)
			}
		case token.MINUS, token.DIV, token.MUL:
			if len(expr.Args) < 2 {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "operator '%s' takes at least two operands", token.PrefixExprName(expr.Tok.Type))
			}
			typ, err := a.infer(expr.Args[0])
			if err != nil {
				return nil, err
			}
			if typ.typ != TypeInt {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "expected 'int', got '%s'", typ.typ)
			}
			if err := expectConsecutive(TypeInt); err != nil {
				return nil, err
			}
			return typ, nil
		case token.AND, token.OR:
			if len(expr.Args) != 2 {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "operator '%s' expects exactly two arguments", token.PrefixExprName(expr.Tok.Type))
			}
			if err := expectConsecutive(TypeBool); err != nil {
				return nil, err
			}
			return NewType(TypeBool, expr.Tok.Line, expr.Tok.Col), nil
		case token.LT, token.LTE, token.GT, token.GTE, token.EQUAL:
			if len(expr.Args) != 2 {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "operator '%s' expects exactly two arguments", token.PrefixExprName(expr.Tok.Type))
			}
			if err := expectConsecutive(TypeInt); err != nil {
				return nil, err
			}
			return NewType(TypeBool, expr.Tok.Line, expr.Tok.Col), nil
		case token.NOT:
			if len(expr.Args) != 1 {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "operator 'not' expects exactly one argument")
			}
			typ, err := a.infer(expr.Args[0])
			if err != nil {
				return nil, err
			}
			if typ.typ != TypeBool {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "expected 'bool', got '%s'", typ.typ)
			}
			return typ, nil
		// list/string indexing
		case token.SINGLE_QUOTE:
			if len(expr.Args) != 1 {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "operator \"'\" expects exactly one argument")
			}
			typ, err := a.infer(expr.Args[0])
			if err != nil {
				return nil, err
			}
			isStrIndex, isListIndex := typ.typ == TypeString, strings.Contains(string(typ.typ), "list-")
			if !(isStrIndex && isListIndex) {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "invalid type of expression for \"'\"")
			}
			return typ, nil
		}
	}
	panic(fmt.Sprintf("--UNREACHABLE--\n*Analyzer.infer: unknown expr '%s'\n", expr.String()))
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

func (a *Analyzer) typecheckVarDecl(s *ast.VariableDeclarationStatement) *IRVariable {
	if d, ok := s.Value.(*ast.Identifier); ok {
		if a.env.IsFailedVar(d.Tok.Literal) {
			return nil
		}
	}
	t, err := a.infer(s.Value)
	if err != nil {
		a.pushErr(err)
		a.env.AddFailedVar(s.Tok.Literal)
		return nil
	}
	varType := NewType(typeLit(s.Tok.Literal), s.Tok.Line, s.Tok.Col)
	if ok := a.match(s.Value, varType); !(ok) {
		a.errorf(s.Tok.Line, s.Tok.Col, "expected '%s', got '%s'", varType.typ, t.typ)
		a.env.AddFailedVar(s.Ident.String())
		return nil
	}
	ir := &IRVariable{Name: s.Ident.String(), Type: s.Tok.Literal, Value: a.toIrExpr(s.Value)}
	if err := a.env.AddVar(ir.Name, ir); err != nil {
		a.errorf(s.Tok.Line, s.Tok.Col, err.Error())
		return nil
	}
	return ir
}

func (a *Analyzer) typecheckListDecl(s *ast.ListVariableDeclarationStatement) *IRVariable {
	if d, ok := s.List.(*ast.Identifier); ok {
		if a.env.IsFailedVar(d.Tok.Literal) {
			return nil
		}
	}
	listExprType, err := a.infer(s.List)
	if err != nil {
		a.pushErr(err)
		return nil
	}
	listType := NewType(TypeList_(typeLit(s.Typ.Literal)), s.Tok.Line, s.Tok.Col)
	if ok := a.match(s.List, listType); !(ok) {
		a.errorf(s.Tok.Line, s.Tok.Col, "expected '%s', got '%s'", listType.typ, listExprType.typ)
		a.env.AddFailedVar(s.Name.String())
		return nil
	}
	ir := &IRVariable{Name: s.Name.String(), Type: TypeList(s.Typ.Literal), Value: a.toIrExpr(s.List, s.Typ.Literal)}
	if err := a.env.AddVar(ir.Name, ir); err != nil {
		a.errorf(s.Tok.Line, s.Tok.Col, err.Error())
		return nil
	}
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
	condType, err := a.infer(s.Cond)
	if err != nil {
		a.pushErr(err)
		return nil
	}
	if ok := a.match(s.Cond, NewType(TypeBool, s.Tok.Line, s.Tok.Col)); !ok {
		a.errorf(s.Tok.Line, s.Tok.Col, "expected 'bool', got '%s'", condType.typ)
		return nil
	}
	ir := &IRIf{Cond: a.toIrExpr(s.Cond)}
	// enter a new scope here
	a.env.EnterScope()
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
	a.env.ExitScope()
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
	a.env.EnterScope()
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
	a.env.ExitScope()
	return ir
}

// this is going to be mostly the same as typecheckIfStmt, but I don't want to create workarounds to prevent
// entering a new scope when using typecheckIfStmt to typecheck an elseif statement.
func (a *Analyzer) typecheckElseIfStmt(s *ast.IfStatement, returnWanted *returnWanted) *IRElseIf {
	condType, err := a.infer(s.Cond)
	if err != nil {
		a.pushErr(err)
		return nil
	}
	if ok := a.match(s.Cond, NewType(TypeBool, s.Tok.Line, s.Tok.Col)); !ok {
		a.errorf(s.Tok.Line, s.Tok.Col, "expected 'bool', got '%s'", condType.typ)
		return nil
	}
	ir := &IRElseIf{Cond: a.toIrExpr(s.Cond)}
	a.env.EnterScope()
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
	a.env.ExitScope()
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
		isDatatypeType := v.Tok.Type != token.INTKW && v.Tok.Type != token.STRINGKW && v.Tok.Type != token.BOOLKW
		if isDatatypeType {
			dt := a.env.GetDatatype(v.Tok.Literal)
			if dt == nil {
				// no such datatype
				a.errorf(v.Tok.Line, v.Tok.Col, "no datatype named '%s'", v.Tok.Literal)
				return nil
			}
		}
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
	var setAllVarsFailed = func() {
		for _, n := range s.Names {
			a.env.AddFailedVar(n.String())
		}
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
		curVal := s.Values[i]
		if d, ok := curVal.(*ast.Identifier); ok {
			if a.env.IsFailedVar(d.String()) {
				setAllVarsFailed()
				return nil
			}
		}
		typ, name := NewType(typeLit(ir.Types[i]), s.Tok.Line, s.Tok.Col), ir.Names[i]
		gotType, err := a.infer(curVal)
		if err != nil {
			a.pushErr(err)
			return nil
		}
		if ok := a.match(curVal, typ); !(ok) {
			a.errorf(s.Tok.Line, s.Tok.Col, "expected '%s', got '%s'", typ.typ, gotType.typ)
			setAllVarsFailed()
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
		if err := a.env.AddVar(name, &IRVariable{Name: name, Type: string(typ.typ), Value: irExpr}); err != nil {
			a.errorf(s.Tok.Line, s.Tok.Col, err.Error())
			return nil
		}
	}
	return ir
}

func (a *Analyzer) typecheckReassignment(s *ast.ReassignmentStatement) *IRReassigment {
	ir := &IRReassigment{Name: s.Ident.String()}
	typOfOldVal := NewType(typeLit(a.env.GetVar(ir.Name).Type), s.Tok.Line, s.Tok.Col)
	newVal := s.NewValue
	newValType, err := a.infer(newVal)
	if err != nil {
		a.pushErr(err)
		return nil
	}
	if ok := a.match(newVal, typOfOldVal); !(ok) {
		a.errorf(s.Tok.Line, s.Tok.Col, "invalid type of expression '%s' for variable '%s'", newValType.typ, ir.Name)
		return nil
	}
	ir.NewValue = a.toIrExpr(newVal)
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
	boolType := NewType(TypeBool, s.Tok.Line, s.Tok.Col)
	cond := s.Cond
	condType, err := a.infer(cond)
	if err != nil {
		a.pushErr(err)
		return nil
	}
	if ok := a.match(cond, boolType); !(ok) {
		a.errorf(s.Tok.Line, s.Tok.Col, "expected 'bool', got '%s'", condType.typ)
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
				retType := NewType(typeLit(ir.Returns[i]), s.Tok.Line, s.Tok.Col)
				vType, err := a.infer(v)
				if err != nil {
					a.pushErr(err)
					return nil
				}
				if ok := a.match(v, retType); !(ok) {
					a.errorf(s.Tok.Line, s.Tok.Col, "expected '%s', got '%s'", retType.typ, vType.typ)
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
		val := vals[i]
		valType, err := a.infer(val)
		if err != nil {
			return err
		}
		typ := r.types[i]
		typType := NewType(typeLit(typ), line, col)
		if ok := a.match(val, typType); !(ok) {
			return newErr(line, col, "expected '%s', got '%s'", typType.typ, valType.typ)
		}
	}
	return nil
}
