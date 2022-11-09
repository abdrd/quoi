package sema

import (
	"quoi/ast"
)

type VarType interface {
	vartype()
}

type lineColStruct struct {
	Line, Col uint
}

// int keyword
type TypeInt struct{ lineColStruct }

type TypeString struct{ lineColStruct }

type TypeBool struct{ lineColStruct }

type TypeDatatype struct {
	lineColStruct
	Datatype string
}

type TypeList struct {
	lineColStruct
	TypeOfList VarType
}

func (TypeInt) vartype()      {}
func (TypeString) vartype()   {}
func (TypeBool) vartype()     {}
func (TypeDatatype) vartype() {}
func (TypeList) vartype()     {}

type CheckedExpr interface {
	ce()
}

// integer literal expression
type IntExpr struct {
	lineColStruct
	Value int64
}

type StringExpr struct {
	lineColStruct
	Value string
}

type BoolExpr struct {
	lineColStruct
	Value bool
}

type PrefixExpr struct {
	*ast.PrefixExpr
}

func (IntExpr) ce()    {}
func (StringExpr) ce() {}
func (BoolExpr) ce()   {}
func (PrefixExpr) ce() {}

type CheckedDecl interface {
	cd()
}

type CheckedVarDecl struct {
	Name  *ast.Identifier
	Type  VarType
	Value CheckedExpr
}

type CheckedProgram struct {
	*ast.Program
}

func (CheckedVarDecl) cd() {}
func (CheckedProgram) cd() {}
