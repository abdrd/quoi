package ast

import (
	"fmt"
	"quoi/token"
)

type Program struct {
	Stmts []Statement
}

type Node interface {
	String() string
}

type Expr interface {
	Node
}

type Statement interface {
	Node
	statement()
}

type StringLit struct {
	Typ token.Type
	Val string
}

func (s StringLit) String() string {
	return fmt.Sprintf("\"%s\"", s.Val)
}

type IntLiteral struct {
	Typ token.Type
	Val int64
}

func (i IntLiteral) String() string {
	return fmt.Sprint(i.Val)
}

type BoolLiteral struct {
	Typ token.Type
	Val bool
}

func (b BoolLiteral) String() string {
	return fmt.Sprint(b.Val)
}

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
