package ast

import (
	"fmt"
	"quoi/token"
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

type VariableDeclaration struct {
	Tok   token.Token // variable type
	Ident *Identifier // variable name
	Value Expr        // variable value
}

func (v VariableDeclaration) String() string {
	if v.Value == nil {
		return "VALUE (Expr) IS NIL"
	}
	name := ""
	if v.Ident != nil {
		name = v.Ident.String()
	}
	return fmt.Sprintf("%s %s = %s.", v.Tok.Literal, name, v.Value.String())
}
func (VariableDeclaration) statement() {}

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

type PrintStatement struct {
	Tok token.Token
	Arg Expr
}

func (p PrintStatement) String() string {
	return fmt.Sprintf("print %s.", p.Arg.String())
}
func (PrintStatement) statement() {}
