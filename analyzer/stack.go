package analyzer

import (
	"fmt"
)

type SymbolTable struct {
	vars      map[string]*IRVariable
	funcs     map[string]*IRFunction
	datatypes map[string]*IRDatatype
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		vars:      make(map[string]*IRVariable),
		funcs:     make(map[string]*IRFunction),
		datatypes: make(map[string]*IRDatatype),
	}
}

// return nil if not found
func (s *SymbolTable) getVar(ident string) *IRVariable {
	if v, found := s.vars[ident]; found {
		return v
	}
	return nil
}

func (s *SymbolTable) addVar(ident string, decl *IRVariable) error {
	if v := s.getVar(ident); v != nil {
		return fmt.Errorf("variable '%s' is already defined", ident)
	}
	s.vars[ident] = decl
	return nil
}

func (s *SymbolTable) getFunc(ident string) *IRFunction {
	if v, found := s.funcs[ident]; found {
		return v
	}
	return nil
}

func (s *SymbolTable) addFunc(decl *IRFunction) error {
	ident := decl.Name
	if v := s.getFunc(ident); v != nil {
		return fmt.Errorf("function '%s' is already declared", ident)
	}
	s.funcs[ident] = decl
	return nil
}

func (s *SymbolTable) getDatatype(ident string) *IRDatatype {
	if v, found := s.datatypes[ident]; found {
		return v
	}
	return nil
}

func (s *SymbolTable) addDatatype(decl *IRDatatype) error {
	ident := decl.Name
	if v := s.getDatatype(ident); v != nil {
		return fmt.Errorf("datatype '%s' is already declared", ident)
	}
	s.datatypes[ident] = decl
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

func (ss *ScopeStack) GetVar(ident string) *IRVariable {
	for _, s := range ss.Scopes {
		if v := s.symbolTable.getVar(ident); v != nil {
			return v
		}
	}
	return nil
}

// add variable to the symbol table of the scope that is at the top of ss.Scopes
func (ss *ScopeStack) AddVar(ident string, decl *IRVariable) error {
	return ss.Scopes[len(ss.Scopes)-1].symbolTable.addVar(ident, decl)
}

func (ss *ScopeStack) GetFunc(ident string) *IRFunction {
	// look at only the global scope, because function can only be declared at the top level.
	return ss.Scopes[0].symbolTable.getFunc(ident)
}

func (ss *ScopeStack) AddFunc(decl *IRFunction) error {
	return ss.Scopes[0].symbolTable.addFunc(decl)
}

func (ss *ScopeStack) AddDatatype(decl *IRDatatype) error {
	return ss.Scopes[0].symbolTable.addDatatype(decl)
}

func (ss *ScopeStack) GetDatatype(ident string) *IRDatatype {
	return ss.Scopes[0].symbolTable.getDatatype(ident)
}
