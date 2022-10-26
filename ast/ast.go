package ast

import (
	"fmt"
	"quoi/token"
	"strings"
)

type Program struct {
	Stmts []Statement
}

func (p *Program) PushStmt(stmt Statement) {
	p.Stmts = append(p.Stmts, stmt)
}

type Node interface {
	String() string
}

type Expr interface {
	Node
}

type ExprStmt interface {
	Expr
}

type Statement interface {
	Node
	statement()
}

type StringLiteral struct {
	Typ token.Type
	Val string
}

func (s StringLiteral) String() string {
	return fmt.Sprintf("\"%s\"", s.Val)
}
func (StringLiteral) statement() {}

type IntLiteral struct {
	Typ token.Type
	Val int64
}

func (i IntLiteral) String() string {
	return fmt.Sprint(i.Val)
}
func (IntLiteral) statement() {}

type BoolLiteral struct {
	Typ token.Type
	Val bool
}

func (b BoolLiteral) String() string {
	return fmt.Sprint(b.Val)
}
func (BoolLiteral) statement() {}

type Identifier struct {
	Tok token.Token
}

func (i Identifier) String() string {
	return i.Tok.Literal
}
func (i Identifier) statement() {}

type VariableDeclarationStatement struct {
	Tok        token.Token // variable type
	IsList     bool
	TypeOfList token.Token
	Ident      *Identifier // variable name
	Value      Expr        // variable value
}

func (v VariableDeclarationStatement) String() string {
	if v.Value == nil {
		return "VALUE (Expr) IS NIL"
	}
	name := ""
	if v.Ident != nil {
		name = v.Ident.String()
	}
	return fmt.Sprintf("%s %s = %s.", v.Tok.Literal, name, v.Value.String())
}
func (VariableDeclarationStatement) statement() {}

type SubsequentVariableDeclarationStatement struct {
	Types  []token.Token
	Names  []*Identifier
	Values []Expr
}

func (s SubsequentVariableDeclarationStatement) String() string {
	var res strings.Builder
	for i, v := range s.Types {
		putComma := i != len(s.Names)-1
		// there are as many names as types.
		res.WriteString(v.Literal)
		res.WriteByte(' ')
		res.WriteString(s.Names[i].String())
		if putComma {
			res.WriteString(", ")
		}
	}
	res.WriteString(" = ")
	for i, v := range s.Values {
		putComma := i != len(s.Values)-1
		res.WriteString(v.String())
		if putComma {
			res.WriteString(", ")
		}
	}
	res.WriteByte('.')
	return res.String()
}
func (SubsequentVariableDeclarationStatement) statement() {}

type ReassignmentStatement struct {
	Tok      token.Token // IDENT token
	Ident    *Identifier
	NewValue Expr
}

func (r ReassignmentStatement) String() string {
	if r.NewValue == nil {
		return "NEW VALUE (Expr) IS NIL"
	}
	name := ""
	if r.Ident != nil {
		name = r.Ident.String()
	}
	return fmt.Sprintf("%s = %s.", name, r.NewValue.String())
}
func (ReassignmentStatement) statement() {}

type BlockStatement struct {
	Tok   token.Token
	Stmts []Statement
}

func (b BlockStatement) String() string {
	res := "block"
	for _, v := range b.Stmts {
		res += "\n\t" + v.String()
	}
	res += "\nend"
	return res
}
func (BlockStatement) statement() {}

type ReturnStatement struct {
	Tok          token.Token
	ReturnValues []Expr
}

func (r ReturnStatement) String() string {
	var res strings.Builder
	res.WriteString("return ")
	for i, v := range r.ReturnValues {
		putComma := i != len(r.ReturnValues)-1
		res.WriteString(v.String())
		if putComma {
			res.WriteString(", ")
		}
	}
	res.WriteByte('.')
	return res.String()
}

func (ReturnStatement) statement() {}

type BreakStatement struct {
	Tok token.Token // token.BREAK
}

func (b BreakStatement) String() string {
	return b.Tok.Literal + "."
}
func (BreakStatement) statement() {}

type ContinueStatement struct {
	Tok token.Token
}

func (c ContinueStatement) String() string {
	return c.Tok.Literal + "."
}
func (ContinueStatement) statement() {}

type LoopStatement struct {
	Tok   token.Token
	Cond  Expr
	Stmts []Statement
}

func (l LoopStatement) String() string {
	res := "loop "
	if l.Cond == nil {
		res = "loop {"
	}
	res += l.Cond.String() + " {"
	if len(l.Stmts) == 0 {
		res += " }"
	} else {
		for _, s := range l.Stmts {
			res += "\n\t" + s.String()
		}
		res += "\n}"
	}
	return res
}
func (LoopStatement) statement() {}

type DatatypeField struct {
	Tok   token.Token
	Ident *Identifier
}

func (d DatatypeField) String() string {
	return fmt.Sprintf("%s %s", d.Tok.Literal, d.Ident.String())
}

type DatatypeDeclaration struct {
	Tok    token.Token
	Name   *Identifier
	Fields []*DatatypeField
}

func (d DatatypeDeclaration) String() string {
	res := "datatype "
	if d.Name != nil {
		res += d.Name.String() + " {"
	}
	if len(d.Fields) == 0 {
		res += " }"
		return res
	}
	for _, v := range d.Fields {
		field := v.String()
		field = "\n\t" + field
		res += field
	}
	res += "\n}"
	return res
}
func (DatatypeDeclaration) statement() {}

type PrefixExpr struct {
	Tok  token.Token // operator (e.g. +, -, ', and, ...)
	Args []Expr
}

func (p PrefixExpr) String() string {
	res := "(" + p.Tok.Literal
	for _, v := range p.Args {
		res += " " + v.String()
	}
	res = res + ")"
	return res
}
func (PrefixExpr) statement() {}

type FunctionCall struct {
	Tok   token.Token
	Ident *Identifier
	Args  []Expr
}

func (f FunctionCall) String() string {
	if f.Ident == nil {
		f.Ident = &Identifier{Tok: token.New(token.IDENT, "<<NIL_IDENT>>", 0, 0)}
	}
	res := f.Ident.String() + "("
	for i, v := range f.Args {
		putComma := i != len(f.Args)-1
		res += v.String()
		if putComma {
			res += ", "
		}
	}
	res += ")"
	return res
}
func (FunctionCall) statement() {}

type Namespace struct {
	Tok        token.Token
	Identifier *Identifier // namespace identifier (e.g. Stdout)
}

type FunctionCallFromNamespace struct {
	Namespace *Namespace
	Function  *FunctionCall
}

func (f FunctionCallFromNamespace) String() string {
	var res string
	if f.Namespace == nil {
		res += "<nil_namespace>"
	} else {
		res += f.Namespace.Tok.Literal
	}
	res += "::"
	if f.Function == nil {
		res += "<nil_function>()"
	} else {
		res += f.Function.String()
	}
	return res
}
func (FunctionCallFromNamespace) statement() {}

type ListLiteral struct {
	Tok   token.Token // [
	Elems []Expr
}

func (l ListLiteral) String() string {
	var res strings.Builder
	res.WriteString("[")
	for i, v := range l.Elems {
		putComma := i != len(l.Elems)-1
		res.WriteString(v.String())
		if putComma {
			res.WriteString(", ")
		}
	}
	res.WriteString("]")
	return res.String()
}
func (ListLiteral) statement() {}

type ListVariableDeclarationStatement struct {
	Tok  token.Token
	Typ  token.Token // types of elements in the list
	Name *Identifier
	List *ListLiteral
}

func (l ListVariableDeclarationStatement) String() string {
	var res strings.Builder
	ident := "<nil_varname>"
	if l.Name != nil {
		ident = l.Name.String()
	}
	list := "<nil_list>"
	if l.List != nil {
		list = l.List.String()
	}
	res.WriteString(fmt.Sprintf("listof %s %s = %s.", l.Typ.Literal, ident, list))
	return res.String()
}
func (ListVariableDeclarationStatement) statement() {}

type ElseStatement struct {
	Tok   token.Token // token.ELSE
	Stmts []Statement
}

func (e ElseStatement) String() string {
	var res strings.Builder
	res.WriteString(" else {\n")
	for _, v := range e.Stmts {
		res.WriteByte('\t')
		res.WriteString(v.String())
		res.WriteByte('\n')
	}
	res.WriteByte('}')
	return res.String()
}
func (ElseStatement) statement() {}

type IfStatement struct {
	Tok         token.Token // token.IF
	Cond        Expr
	Stmts       []Statement
	Alternative *IfStatement
	Default     *ElseStatement
}

func (i IfStatement) String() string {
	var res strings.Builder
	res.WriteString("if ")
	if i.Cond != nil {
		res.WriteString(i.Cond.String())
		res.WriteString(" {\n")
	} else {
		res.WriteString("<nil_cond> {\n")
	}
	for _, v := range i.Stmts {
		res.WriteByte('\t')
		res.WriteString(v.String())
		res.WriteByte('\n')
	}
	res.WriteByte('}')
	if i.Alternative != nil {
		res.WriteString(" else")
		res.WriteString(i.Alternative.String())
	}
	if i.Default != nil {
		res.WriteString(i.Default.String())
	}
	return res.String()
}
func (IfStatement) statement() {}

type FunctionParameter struct {
	Tok        token.Token // type of parameter (int, string, User, ...)
	IsList     bool
	TypeOfList token.Token
	Name       *Identifier // name of parameter
}

type FunctionReturnType struct {
	Tok    token.Token // actual type (token.INTKW, token.STRINGKW, token.IDENT, etc.)
	IsList bool        // since listof token is one token, and types of lists are composed of two tokens, ...
	// listof int, listof string, listof City, ...
	TypeOfList token.Token // int, string, City, ...
}

type FunctionDeclarationStatement struct {
	Tok         token.Token // token.FUN
	Name        *Identifier // function name
	Params      []FunctionParameter
	ReturnCount int // how many things does this return ?
	ReturnTypes []FunctionReturnType
	Stmts       []Statement
}

func (f FunctionDeclarationStatement) String() string {
	var res strings.Builder
	res.WriteString("fun ")
	if f.Name != nil {
		res.WriteString(f.Name.String())
	} else {
		res.WriteString("<nil_name>")
	}
	res.WriteByte('(')
	for i, v := range f.Params {
		putComma := i != len(f.Params)-1
		res.WriteString(v.Tok.Literal)
		if v.IsList {
			res.WriteByte(' ')
			res.WriteString(v.TypeOfList.Literal)
		}
		res.WriteByte(' ')
		res.WriteString(v.Name.String())
		if putComma {
			res.WriteString(", ")
		}
	}
	res.WriteString(") -> ")
	for i, v := range f.ReturnTypes {
		putComma := i != len(f.ReturnTypes)-1
		res.WriteString(v.Tok.Literal)
		if putComma {
			res.WriteString(", ")
		}
	}
	res.WriteString(" {")
	res.WriteByte('\n')
	for _, v := range f.Stmts {
		res.WriteByte('\t')
		res.WriteString(v.String())
		res.WriteByte('\n')
	}
	res.WriteByte('}')
	return res.String()
}
func (FunctionDeclarationStatement) statement() {}

// <ident>=<value>
type DataypeLiteralField struct {
	Name  *Identifier
	Value Expr
}

func (d DataypeLiteralField) String() string {
	return fmt.Sprintf("%s=%s", d.Name.String(), d.Value.String())
}

type DatatypeLiteral struct {
	Tok    token.Token // token.IDENT
	Fields []*DataypeLiteralField
}

func (d DatatypeLiteral) String() string {
	var res strings.Builder
	res.WriteString(d.Tok.Literal)
	res.WriteByte('{')
	if len(d.Fields) > 0 {
		res.WriteByte('\n')
	}
	for _, v := range d.Fields {
		res.WriteByte('\t')
		res.WriteString(v.String())
		res.WriteByte('\n')
	}
	res.WriteByte('}')
	return res.String()
}
func (DatatypeLiteral) statement() {}
