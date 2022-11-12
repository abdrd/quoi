package analyzer

import (
	"fmt"
	"quoi/ast"
)

type SymbolTable struct {
	vars map[string]*ast.VariableDeclarationStatement
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		vars: make(map[string]*ast.VariableDeclarationStatement),
	}
}

// return nil if not found
func (s *SymbolTable) getVar(ident string) *ast.VariableDeclarationStatement {
	if v, found := s.vars[ident]; found {
		return v
	}
	return nil
}

func (s *SymbolTable) addVar(ident string, decl *ast.VariableDeclarationStatement) error {
	if v := s.getVar(ident); v != nil {
		return fmt.Errorf("variable '%s' is already defined", ident)
	}
	s.vars[ident] = decl
	return nil
}

type Scope struct {
	symbolTable *SymbolTable
}

func NewScope() *Scope {
	return &Scope{symbolTable: NewSymbolTable()}
}

type ScopeStack struct {
	Scopes []*Scope
}

func NewScopeStack() *ScopeStack {
	// create the global scope
	gs := NewScope()
	scopes := []*Scope{}
	scopes = append(scopes, gs)
	return &ScopeStack{Scopes: scopes}
}

// append/push to the end
func (ss *ScopeStack) push(scope *Scope) {
	ss.Scopes = append(ss.Scopes, scope)
}

// pop from the end
func (ss *ScopeStack) pop() *Scope {
	scopesLen := len(ss.Scopes)
	if scopesLen == 0 {
		return nil
	}
	popped := ss.Scopes[scopesLen-1]
	ss.Scopes = ss.Scopes[:scopesLen-1]
	return popped
}

func (ss *ScopeStack) EnterScope() {
	ss.push(NewScope())
}

func (ss *ScopeStack) ExitScope() {
	ss.pop()
}

func (ss *ScopeStack) GetVar(ident string) *ast.VariableDeclarationStatement {
	for _, s := range ss.Scopes {
		if v := s.symbolTable.getVar(ident); v != nil {
			return v
		}
	}
	return nil
}

// add variable to the symbol table of the scope that is at the top of ss.Scopes
func (ss *ScopeStack) AddVar(ident string, decl *ast.VariableDeclarationStatement) error {
	return ss.Scopes[len(ss.Scopes)-1].symbolTable.addVar(ident, decl)
}
