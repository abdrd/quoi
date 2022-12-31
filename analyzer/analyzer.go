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
	std     *StandardLibrary
	Errs    []Err

	// state
	seenReturn bool
}

func New(program *ast.Program) *Analyzer {
	a := &Analyzer{program: program, curExpr: nil, env: NewScopeStack()}
	a.std = InitStandardLibrary(a)
	return a
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
	return a.env.AddFunc(ir.Name, ir)
}

func (a *Analyzer) registerStdFuncSignature(ns, name string, s *ast.FunctionDeclarationStatement) {
	ir := &IRFunction{Name: s.Name.String(), TakesCount: len(s.Params), ReturnsCount: len(s.ReturnTypes)}
	for _, v := range s.Params {
		ir.Takes = append(ir.Takes, fnParamTypeRepr(v))
	}
	for _, v := range s.ReturnTypes {
		ir.Returns = append(ir.Returns, fnReturnTypeRepr(v))
	}
	a.std.AddFunc(ns, name, ir)
}

func (a *Analyzer) registerDatatype(s *ast.DatatypeDeclaration) error {
	ir := &IRDatatype{Name: s.Name.String(), FieldCount: len(s.Fields)}
	for _, v := range s.Fields {
		field := IRDatatypeField{Type: v.Tok.Literal, Name: v.Ident.String()}
		ir.Fields = append(ir.Fields, field)
	}
	return a.env.AddDatatype(ir.Name, ir)
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

const (
	TypeString = "string"
	TypeInt    = "int"
	TypeBool   = "bool"
	TypeVoid   = "void"
	// empty lists are of this type (list-any)
	TypeAny = "any"
)

var (
	TypeList_ = func(t string) string {
		return "list-" + t
	}
	TypeDatatype_ = func(dt string) string {
		return dt
	}
)

type Type struct {
	typ       string
	line, col uint
	next      *Type
}

func (t *Type) setNext(ty *Type) {
	if t.next == nil {
		t.next = ty
	}
}

func NewType(typ string, line, col uint) *Type {
	return &Type{typ: typ, line: line, col: col}
}

func NewListType(typ string, line, col uint) *Type {
	return NewType(TypeList_(typ), line, col)
}

func NewTypeFromVarType(typ ast.VarType, line, col uint) *Type {
	if typ.IsList {
		return NewListType(typ.TypeOfList.Literal, line, col)
	}
	return NewType(typ.Tok.Literal, line, col)
}

func (a *Analyzer) matchTypes(lhs, rhs *Type) error {
	for lhs.next != nil && rhs.next != nil {
		if lhs.typ != rhs.typ {
			return newErr(lhs.line, lhs.col, "expected '%s', got '%s'", lhs.typ, rhs.typ)
		}
		lhs = lhs.next
		rhs = rhs.next
	}
	lhsExhausted, rhsExhausted := lhs.next == nil, rhs.next == nil
	if lhsExhausted && !(rhsExhausted) {
		return newErr(lhs.line, lhs.col, "unused value of type '%s'", rhs.typ)
	} else if !(lhsExhausted) && rhsExhausted {
		return newErr(lhs.line, lhs.col, "value assigned to nothing")
	} else if lhs.typ != rhs.typ {
		return newErr(lhs.line, lhs.col, "mismatched types '%s', and '%s'", lhs.typ, rhs.typ)
	}
	return nil
}

func (a *Analyzer) match(expr ast.Expr, t *Type) error {
	switch expr := expr.(type) {
	case *ast.StringLiteral:
		if t.typ != TypeString {
			return newErr(expr.Typ.Line, expr.Typ.Col, "expected '%s', got 'string'", t.typ)
		}
		return nil
	case *ast.IntLiteral:
		if t.typ != TypeInt {
			return newErr(expr.Typ.Line, expr.Typ.Col, "expected '%s', got 'int'", t.typ)
		}
		return nil
	case *ast.BoolLiteral:
		if t.typ != TypeBool {
			return newErr(expr.Typ.Line, expr.Typ.Col, "expected '%s', got 'bool'", t.typ)
		}
		return nil
	case *ast.ListLiteral:
		// learn the type of list literal
		listType, err := a.infer(expr)
		if err != nil {
			return err
		}
		if listType.typ == TypeAny {
			return nil
		}
		if t.typ != listType.typ {
			return newErr(expr.Tok.Line, expr.Tok.Col, "expected '%s', got '%s' in list literal", t.typ, listType.typ)
		}
		return nil
	case *ast.DatatypeLiteral:
		datatypeType, err := a.infer(expr)
		if err != nil {
			return err
		}
		if t.typ != datatypeType.typ {
			return newErr(expr.Tok.Line, expr.Tok.Col, "expected '%s', got '%s' in datatype literal", t.typ, datatypeType.typ)
		}
		return nil
	case *ast.FunctionCall:
		fnType, err := a.infer(expr)
		if err != nil {
			return err
		}
		for t.next != nil && fnType.next != nil {
			if t.typ != fnType.typ {
				return newErr(expr.Tok.Line, expr.Tok.Col, "expected '%s', got '%s'", t.typ, fnType.typ)
			}
			t = t.next
			fnType = fnType.next
		}
		tExhausted, fnTypeExhausted := t.next == nil, fnType.next == nil
		if tExhausted && !(fnTypeExhausted) {
			return newErr(expr.Tok.Line, expr.Tok.Col, "unused value from function call '%s'", expr.Ident)
		} else if !(tExhausted) && fnTypeExhausted {
			return newErr(expr.Tok.Line, expr.Tok.Col, "variable assigned to nothing")
		} else if t.typ != fnType.typ {
			return newErr(expr.Tok.Line, expr.Tok.Col, "expected '%s', got '%s'", t.typ, fnType.typ)
		}
		return nil
	case *ast.FunctionCallFromNamespace:
		fnName, ns := expr.Function.Ident.String(), expr.Namespace.Identifier.String()
		line, col := expr.Namespace.Tok.Line, expr.Namespace.Tok.Col
		typ, err := a.infer(expr)
		if err != nil {
			return err
		}
		fn := a.std.GetFunc(ns, fnName)
		if fn == nil {
			return newErr(expr.Namespace.Tok.Line, expr.Namespace.Tok.Col, "unknown function '%s::%s'", ns, fnName)
		}
		for i := 0; i < len(expr.Function.Args); i++ {
			argType, err := a.infer(expr.Function.Args[i])
			if err != nil {
				return err
			}
			if argType.typ != t.typ {
				return newErr(expr.Function.Tok.Line, expr.Namespace.Tok.Col, "expected '%s', got '%s' as argument in '%s::%s'", t.typ, argType.typ, ns, fnName)
			}
		}
		for t.next != nil && typ.next != nil {
			if t.typ != typ.typ {
				return newErr(line, col, "expected '%s', got '%s'", t.typ, typ.typ)
			}
			t = t.next
			typ = typ.next
		}
		tExhausted, fnTypeExhausted := t.next == nil, typ.next == nil
		if tExhausted && !(fnTypeExhausted) {
			return newErr(line, col, "unused value from function call '%s::%s'", ns, fnName)
		} else if !(tExhausted) && fnTypeExhausted {
			return newErr(line, col, "variable assigned to nothing")
		} else if t.typ != typ.typ {
			return newErr(line, col, "expected '%s', got '%s'", t.typ, typ.typ)
		}
		return nil
	case *ast.PrefixExpr:
		prefType, err := a.infer(expr)
		if err != nil {
			return err
		}
		if t.typ != prefType.typ {
			return newErr(expr.Tok.Line, expr.Tok.Col, "expected '%s', got '%s'", t.typ, prefType.typ)
		}
		return nil
	case *ast.Identifier:
		if a.env.IsFailedVar(expr.Tok.Literal) {
			return nil
		}
		typ := a.env.GetVar(expr.String())
		if typ == "" {
			return newErr(expr.Tok.Line, expr.Tok.Col, "reference to non-existent variable '%s'", expr.String())
		}
		if t.typ != typ {
			return newErr(expr.Tok.Line, expr.Tok.Col, "expected '%s', got '%s'", t.typ, typ)
		}
		return nil
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
			if err := a.match(el, firstElemType); err != nil {
				return nil, err
			}
		}
		return NewType(TypeList_(firstElemType.typ), firstElemType.line, firstElemType.col), nil
	case *ast.Identifier:
		typ := a.env.GetVar(expr.String())
		if typ == "" {
			return nil, newErr(expr.Tok.Line, expr.Tok.Col, "reference to non-existent variable '%s'", expr.Tok.Literal)
		}
		return NewType(typ, expr.Tok.Line, expr.Tok.Col), nil
	case *ast.DatatypeLiteral:
		datatype := a.env.GetDatatype(expr.Tok.Literal)
		if datatype == nil {
			return nil, newErr(expr.Tok.Line, expr.Tok.Col, "initialization of non-existent datatype '%s'", expr.Tok.Literal)
		}
		if len(expr.Fields) > datatype.FieldCount {
			unknownField := expr.Fields[datatype.FieldCount+1]
			dtName := datatype.Name
			return nil, newErr(expr.Tok.Line, expr.Tok.Col, "unknown field '%s' in datatype literal '%s'", unknownField, dtName)
		}
		exprFieldsNameTypeMap := map[string]*Type{}
		datatypeFieldsNameTypeMap := map[string]*Type{}
		for _, v := range expr.Fields {
			typ, err := a.infer(v.Value)
			if err != nil {
				return nil, err
			}
			exprFieldsNameTypeMap[v.Name.String()] = typ
		}
		for _, v := range datatype.Fields {
			datatypeFieldsNameTypeMap[v.Name] = NewType(v.Type, expr.Tok.Line, expr.Tok.Col)
		}
		for k, v := range exprFieldsNameTypeMap {
			t, ok := datatypeFieldsNameTypeMap[k]
			if !(ok) {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "unknown field '%s' in datatype literal '%s'", k, datatype.Name)
			}
			if err := a.matchTypes(t, v); err != nil {
				errMsg := fmt.Sprintf("type mismatch for field '%s' in datatype literal '%s' (want=%s got=%s)", k, datatype.Name, t.typ, v.typ)
				if v.next != nil {
					errMsg = fmt.Sprintf("type mismatch for field '%s' in datatype literal '%s'; more value on rhs", k, datatype.Name)
				}
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, errMsg)
			}
		}
		return NewType(expr.Tok.Literal, expr.Tok.Line, expr.Tok.Col), nil
	case *ast.FunctionCall:
		lenArgs := len(expr.Args)
		fn := a.env.GetFunc(expr.Tok.Literal)
		if fn == nil {
			return nil, newErr(expr.Tok.Line, expr.Tok.Col, "invoking of non-existent function '%s'", expr.Ident)
		}
		if fn.TakesCount == 0 && lenArgs != 0 {
			return nil, newErr(expr.Tok.Line, expr.Tok.Col, "function '%s' takes no arguments", expr.Ident)
		}
		if fn.TakesCount > lenArgs {
			return nil, newErr(expr.Tok.Line, expr.Tok.Col, "function '%s' was given insufficient number of arguments (want=%d got=%d)", expr.Ident, fn.TakesCount, lenArgs)
		} else if fn.TakesCount < lenArgs {
			return nil, newErr(expr.Tok.Line, expr.Tok.Col, "function '%s' was given excessive number of arguments (want=%d got=%d)", expr.Ident, fn.TakesCount, lenArgs)
		}
		if fn.ReturnsCount == 0 {
			return NewType(TypeVoid, expr.Tok.Line, expr.Tok.Col), nil
		}
		if fn.ReturnsCount == 1 {
			return NewType(fn.Returns[0], expr.Tok.Line, expr.Tok.Col), nil
		}
		// head of the linked list
		// other return types will follow this.
		//
		// for example:
		// [int] --next--> [string] --next--> [User] --next--> *nil*
		// a very simple singly-linked list
		t := NewType(fn.Returns[0], expr.Tok.Line, expr.Tok.Col)
		// copy t; because when we exit the loop, t will no longer be the first type; instead
		// it will be the last type.
		// And we want to return the first type, so that we can, in the future, follow the type chain.
		tOriginal := t
		for i, r := range fn.Returns {
			if i == 0 {
				// skip the first one to prevent duplicate type at the beginning
				continue
			}
			t2 := NewType(r, expr.Tok.Line, expr.Tok.Col)
			t.setNext(t2)
			t = t2
		}
		return tOriginal, nil
	case *ast.FunctionCallFromNamespace:
		// very similar to above case

		lenArgs := len(expr.Function.Args)
		line, col := expr.Namespace.Tok.Line, expr.Namespace.Tok.Col
		ns, name := expr.Namespace.Tok.Literal, expr.Function.Ident.String()
		fn := a.std.GetFunc(ns, name)
		if fn == nil {
			return nil, newErr(line, col, "unknown function '%s::%s'", ns, name)
		}
		argLen := len(expr.Function.Args)
		if argLen > fn.TakesCount {
			return nil, newErr(line, col, "excessive number of arguments passed to function '%s::%s' (want=%d got=%d)", ns, name, fn.TakesCount, argLen)
		} else if argLen < fn.TakesCount {
			return nil, newErr(line, col, "insufficient number of arguments passed to function '%s::%s' (want=%d got=%d)", ns, name, fn.TakesCount, argLen)
		}
		if fn.TakesCount == 0 && lenArgs != 0 {
			return nil, newErr(line, col, "function '%s::%s' takes no arguments", ns, name)
		}
		if fn.TakesCount > lenArgs {
			return nil, newErr(line, col, "function '%s::%s' was given insufficient number of arguments (want=%d got=%d)", ns, name, fn.ReturnsCount, lenArgs)
		} else if fn.TakesCount < lenArgs {
			return nil, newErr(line, col, "function '%s::%s' was given excessive number of arguments (want=%d got=%d)", ns, name, fn.ReturnsCount, lenArgs)
		}
		if fn.ReturnsCount == 0 {
			return NewType(TypeVoid, line, col), nil
		}
		if fn.ReturnsCount == 1 {
			return NewType(fn.Returns[0], line, col), nil
		}
		t := NewType(fn.Returns[0], line, col)
		tOriginal := t
		for i, r := range fn.Returns {
			if i == 0 {
				continue
			}
			t2 := NewType(r, line, col)
			t.setNext(t2)
			t = t2
		}
		return tOriginal, nil
	case *ast.PrefixExpr:
		var expectConsecutive = func(what string) error {
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
			if len(expr.Args) != 2 {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "operator \"'\" expects exactly two arguments")
			}
			typ, err := a.infer(expr.Args[0])
			if err != nil {
				return nil, err
			}
			isStrIndex, isListIndex := typ.typ == TypeString, strings.Contains(typ.typ, "list-")
			if !(isStrIndex) && !(isListIndex) {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "invalid type of expression for \"'\"")
			}
			if isListIndex {
				typOfList := strings.Split(typ.typ, "list-")[1]
				return NewType(typOfList, expr.Tok.Line, expr.Tok.Col), nil
			}
			return NewType(TypeString, expr.Tok.Line, expr.Tok.Col), nil
		case token.GET:
			if len(expr.Args) != 2 {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "operator 'get' expects exactly two arguments")
			}
			typ, err := a.infer(expr.Args[0])
			if err != nil {
				return nil, err
			}
			dt := a.env.GetDatatype(typ.typ)
			if dt == nil {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "no variable called '%s' that is a datatype", expr.Args[0])
			}
			field := expr.Args[1]
			// field is an identifier
			var found bool
			var retType string
			for _, v := range dt.Fields {
				if v.Name == field.String() {
					found = true
					retType = v.Type
				}
			}
			if !(found) {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "no field named '%s' in datatype '%s'", field, dt.Name)
			}
			return NewType(retType, expr.Tok.Line, expr.Tok.Col), nil
		case token.SET:
			if len(expr.Args) != 3 {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "operator 'set' expects exactly three arguments")
			}
			typ, err := a.infer(expr.Args[0])
			if err != nil {
				return nil, err
			}
			dt := a.env.GetDatatype(typ.typ)
			if dt == nil {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "no variable called '%s' that is a datatype", expr.Args[0])
			}
			field := expr.Args[1]
			// field is an identifier
			var found bool
			var fieldType string
			for _, v := range dt.Fields {
				if v.Name == field.String() {
					found = true
					fieldType = v.Type
				}
			}
			if !(found) {
				return nil, newErr(expr.Tok.Line, expr.Tok.Col, "no field named '%s' in datatype '%s'", field, dt.Name)
			}
			if err := a.match(expr.Args[2], NewType(fieldType, expr.Tok.Line, expr.Tok.Col)); err != nil {
				return nil, err
			}
			return NewType(dt.Name, expr.Tok.Line, expr.Tok.Col), nil
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
	case *ast.FunctionCall:
		return a.typecheckFunCall(s)
	case *ast.FunctionCallFromNamespace:
		return a.typecheckFunCallFromNamespace(s)
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
		typ := a.env.GetVar(expr.Tok.Literal)
		return &IRVariableReference{Name: expr.String(), Type: typ}
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
	case *ast.FunctionCall:
		fnName := expr.Ident.String()
		// this can't be nil
		fn := a.env.GetFunc(fnName)
		ir := &IRFunctionCall{Name: fnName, TakesCount: fn.TakesCount, ReturnsCount: fn.ReturnsCount, Returns: fn.Returns}
		for _, v := range expr.Args {
			ir.Takes = append(ir.Takes, a.toIrExpr(v))
		}
		return ir
	case *ast.FunctionCallFromNamespace:
		fnName, ns := expr.Function.Ident.String(), expr.Namespace.Tok.Literal
		fn := a.std.GetFunc(ns, fnName)
		ir := &IRFunctionCallFromNamespace{Namespace: ns, IRFunctionCall: IRFunctionCall{
			Name:         fnName,
			Returns:      fn.Returns,
			TakesCount:   fn.TakesCount,
			ReturnsCount: fn.ReturnsCount,
		}}
		for _, v := range expr.Function.Args {
			ir.Takes = append(ir.Takes, a.toIrExpr(v))
		}
		return ir
	case *ast.DatatypeLiteral:
		ir := &IRDatatypeLiteral{Name: expr.Tok.Literal, FieldsAndValues: make(map[string]IRExpression)}
		for _, v := range expr.Fields {
			ir.FieldsAndValues[v.Name.String()] = a.toIrExpr(v.Value)
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
	varType := NewType(s.Tok.Literal, s.Tok.Line, s.Tok.Col)
	if err := a.match(s.Value, varType); err != nil {
		a.pushErr(err)
		a.env.AddFailedVar(s.Ident.String())
		return nil
	}
	ir := &IRVariable{Name: s.Ident.String(), Type: s.Tok.Literal, Value: a.toIrExpr(s.Value)}
	if err := a.env.AddVar(ir.Name, ir.Type); err != nil {
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
	listType := NewType(TypeList_(s.Typ.Literal), s.Tok.Line, s.Tok.Col)
	if err := a.match(s.List, listType); err != nil {
		a.pushErr(err)
		a.env.AddFailedVar(s.Name.String())
		return nil
	}
	ir := &IRVariable{Name: s.Name.String(), Type: TypeList(s.Typ.Literal), Value: a.toIrExpr(s.List, s.Typ.Literal)}
	if err := a.env.AddVar(ir.Name, ir.Type); err != nil {
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
	if err := a.match(s.Cond, NewType(TypeBool, s.Tok.Line, s.Tok.Col)); err != nil {
		a.pushErr(err)
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
	if err := a.match(s.Cond, NewType(TypeBool, s.Tok.Line, s.Tok.Col)); err != nil {
		a.pushErr(err)
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
	names, types := s.Names, s.Types
	var buildTypeChain = func(varTypes []ast.VarType, line, col uint) *Type {
		t := NewTypeFromVarType(varTypes[0], line, col)
		if len(varTypes) == 1 {
			return t
		}
		tCopy := t
		for i := 1; i < len(varTypes); i++ {
			newT := NewTypeFromVarType(varTypes[i], line, col)
			tCopy.setNext(newT)
			tCopy = newT
		}
		return t
	}
	var buildRhsTypeChain = func() (*Type, error) {
		t, err := a.infer(s.Values[0])
		if err != nil {
			return nil, err
		}
		tCopy := t
		for i := 1; i < len(s.Values); i++ {
			newT, err := a.infer(s.Values[i])
			if err != nil {
				return nil, err
			}
			tCopy.setNext(newT)
			tCopy = newT
		}
		return t, nil
	}
	lhs := buildTypeChain(types, s.Tok.Line, s.Tok.Col)
	rhs, err := buildRhsTypeChain()
	if err != nil {
		a.pushErr(err)
		return nil
	}
	var setAllVarsFailed = func() {
		for _, v := range names {
			a.env.AddFailedVar(v.String())
		}
	}
	if err := a.matchTypes(lhs, rhs); err != nil {
		a.pushErr(err)
		setAllVarsFailed()
		return nil
	}
	for _, n := range names {
		ir.Names = append(ir.Names, n.String())
	}
	for _, n := range types {
		var t string
		if n.IsList {
			t = TypeList_(n.TypeOfList.Literal)
		} else {
			t = n.Tok.Literal
		}
		ir.Types = append(ir.Types, t)
	}
	for _, v := range s.Values {
		if d, ok := v.(*ast.Identifier); ok {
			if a.env.IsFailedVar(d.String()) {
				setAllVarsFailed()
				return nil
			}
		}
		ir.Values = append(ir.Values, a.toIrExpr(v))
	}
	// add variables
	for i := 0; ; i++ {
		name := names[i].String()
		a.env.AddVar(name, rhs.typ)
		if rhs.next == nil {
			break
		}
		rhs = rhs.next
	}
	return ir
}

func (a *Analyzer) typecheckReassignment(s *ast.ReassignmentStatement) *IRReassigment {
	ir := &IRReassigment{Name: s.Ident.String()}
	typOfOldVal := NewType(a.env.GetVar(ir.Name), s.Tok.Line, s.Tok.Col)
	newVal := s.NewValue
	if err := a.match(newVal, typOfOldVal); err != nil {
		a.pushErr(err)
		return nil
	}
	ir.NewValue = a.toIrExpr(newVal)
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
	if err := a.match(cond, boolType); err != nil {
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
		if err := a.env.AddVar(param.Name, param.Type); err != nil {
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
				retType := NewType(ir.Returns[i], s.Tok.Line, s.Tok.Col)
				if err := a.match(v, retType); err != nil {
					a.pushErr(err)
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
		typ := r.types[i]
		typType := NewType(typ, line, col)
		if err := a.match(val, typType); err != nil {
			return err
		}
	}
	return nil
}

func (a *Analyzer) typecheckFunCall(s *ast.FunctionCall) *IRFunctionCall {
	fnName := s.Ident.String()
	ir := &IRFunctionCall{Name: fnName}
	fr := a.env.GetFunc(fnName)
	if fr == nil {
		a.errorf(s.Tok.Line, s.Tok.Col, "invoking of non-existent function '%s'", fnName)
		return nil
	}
	if fr.ReturnsCount > 0 {
		a.errorf(s.Tok.Line, s.Tok.Col, "unused value from function call '%s'", fnName)
		return nil
	}
	lenArgs := len(s.Args)
	if fr.TakesCount > lenArgs {
		a.errorf(s.Tok.Line, s.Tok.Col, "missing arguments to function call '%s'", fnName)
		return nil
	} else if fr.TakesCount < lenArgs {
		a.errorf(s.Tok.Line, s.Tok.Col, "excessive number of arguments to function call '%s'", fnName)
		return nil
	}
	for i := 0; i < len(s.Args); i++ {
		t, err := a.infer(s.Args[i])
		if err != nil {
			a.pushErr(err)
			return nil
		}
		paramType := NewType(fr.Takes[i], s.Tok.Line, s.Tok.Col)
		if err := a.matchTypes(paramType, t); err != nil {
			if strings.HasPrefix(err.Error(), "unused") {
				a.errorf(s.Tok.Line, s.Tok.Col, "excessive number of arguments passed to function call '%s'", fnName)
			} else if strings.HasPrefix(err.Error(), "mismatched") {
				a.errorf(s.Tok.Line, s.Tok.Col, "wrong type of argument passed to function '%s'", fnName)
			} else {
				a.pushErr(err)
			}
			return nil
		}
	}
	ir.Returns = fr.Returns
	for _, v := range s.Args {
		ir.Takes = append(ir.Takes, a.toIrExpr(v))
	}
	ir.ReturnsCount = len(ir.Returns)
	ir.TakesCount = len(ir.Takes)
	return ir
}

func (a *Analyzer) typecheckFunCallFromNamespace(s *ast.FunctionCallFromNamespace) *IRFunctionCallFromNamespace {
	fnName := s.Function.Ident.String()
	irFnCall := IRFunctionCall{Name: fnName}
	ir := &IRFunctionCallFromNamespace{Namespace: s.Namespace.Tok.Literal, IRFunctionCall: irFnCall}
	line, col := s.Namespace.Tok.Line, s.Namespace.Tok.Col
	fr := a.std.GetFunc(ir.Namespace, fnName)
	if fr == nil {
		a.errorf(line, col, "invoking of non-existent function '%s::%s'", ir.Namespace, fnName)
		return nil
	}
	if fr.ReturnsCount > 0 {
		a.errorf(line, col, "unused value from function call '%s::%s'", ir.Namespace, fnName)
		return nil
	}
	lenArgs := len(s.Function.Args)
	if fr.TakesCount > lenArgs {
		a.errorf(line, col, "missing arguments to function call '%s::%s'", ir.Namespace, fnName)
		return nil
	} else if fr.TakesCount < lenArgs {
		a.errorf(line, col, "excessive number of arguments to function call '%s::%s'", ir.Namespace, fnName)
		return nil
	}
	for i := 0; i < len(s.Function.Args); i++ {
		t, err := a.infer(s.Function.Args[i])
		if err != nil {
			a.pushErr(err)
			return nil
		}
		paramType := NewType(fr.Takes[i], line, col)
		if err := a.matchTypes(paramType, t); err != nil {
			if strings.HasPrefix(err.Error(), "unused") {
				a.errorf(line, col, "excessive number of arguments passed to function call '%s::%s'", ir.Namespace, fnName)
			} else if strings.HasPrefix(err.Error(), "mismatched") {
				a.errorf(line, col, "wrong type of argument passed to function '%s::%s'", ir.Namespace, fnName)
			} else {
				a.pushErr(err)
			}
			return nil
		}
	}
	ir.Returns = fr.Returns
	for _, v := range s.Function.Args {
		ir.Takes = append(ir.Takes, a.toIrExpr(v))
	}
	ir.ReturnsCount = len(ir.Returns)
	ir.TakesCount = len(ir.Takes)
	return ir
}
