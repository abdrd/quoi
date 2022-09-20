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

type VariableDeclaration struct {
	Tok   token.Token // variable type
	Name  string      // variable name
	Value Expr        // variable value
}

func (v VariableDeclaration) String() string {
	if v.Value == nil {
		return "VALUE (Expr) IS NIL"
	}
	return fmt.Sprintf("%s %s = %s.", v.Tok.Literal, v.Name, v.Value.String())
}
func (VariableDeclaration) statement() {}
