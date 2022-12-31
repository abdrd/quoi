package analyzer

import (
	"fmt"
)

type SymbolTable struct {
	// name: type
	vars      map[string]string
	funcs     map[string]*IRFunction
	datatypes map[string]*IRDatatype

	// this field exists because:
	// if typechecking of a variable fails,
	// we don't add that variable to 'vars';
	// and if, in the future, we'd like to access
	// that variable, we get an error.
	//
	// suppose a situation like this:
	// 	int a = (+ "Hello " "world!").
	//  int b = a.
	//
	// the analyzer reports two errors:
	//	1- expected 'int', got 'string'
	//	2- reference to non-existent variable 'a'
	//
	// but, I don't want the second error to appear, because
	// it is redundant.
	// to prevent that, I am declaring this field here.
	//
	failedVars map[string]bool
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		vars:      make(map[string]string),
		funcs:     make(map[string]*IRFunction),
		datatypes: make(map[string]*IRDatatype),

		failedVars: make(map[string]bool),
	}
}

func (s *SymbolTable) getVar(ident string) string {
	return s.vars[ident]
}

func (s *SymbolTable) addVar(ident, type_ string) error {
	d := s.getVar(ident)
	if len(d) > 0 {
		return fmt.Errorf("variable '%s' is already defined", ident)
	}
	s.vars[ident] = type_
	return nil
}

func (s *SymbolTable) isFailedVar(ident string) bool {
	return s.failedVars[ident]
}

func (s *SymbolTable) addFailedVar(ident string) {
	s.failedVars[ident] = true
}

func (s *SymbolTable) getFunc(ident string) *IRFunction {
	return s.funcs[ident]
}

func (s *SymbolTable) addFunc(ident string, rec *IRFunction) error {
	if v := s.getFunc(ident); v != nil {
		return fmt.Errorf("function '%s' is already declared", ident)
	}
	s.funcs[ident] = rec
	return nil
}

func (s *SymbolTable) getDatatype(ident string) *IRDatatype {
	return s.datatypes[ident]
}

func (s *SymbolTable) addDatatype(ident string, rec *IRDatatype) error {
	if v := s.getDatatype(ident); v != nil {
		return fmt.Errorf("datatype '%s' is already declared", ident)
	}
	s.datatypes[ident] = rec
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

func (ss *ScopeStack) GetVar(ident string) string {
	for i := len(ss.Scopes) - 1; i >= 0; i-- {
		if v := ss.Scopes[i].symbolTable.getVar(ident); v != "" {
			return v
		}
	}
	return ""
}

func (ss *ScopeStack) AddVar(ident, type_ string) error {
	err := ss.Scopes[len(ss.Scopes)-1].symbolTable.addVar(ident, type_)
	if err != nil {
		ss.AddFailedVar(ident)
	}
	return err
}

func (ss *ScopeStack) IsFailedVar(ident string) bool {
	return ss.Scopes[len(ss.Scopes)-1].symbolTable.isFailedVar(ident)
}

func (ss *ScopeStack) AddFailedVar(ident string) {
	ss.Scopes[len(ss.Scopes)-1].symbolTable.addFailedVar(ident)
}

func (ss *ScopeStack) GetFunc(ident string) *IRFunction {
	// look at only the global scope, because function can only be declared at the top level.
	return ss.Scopes[0].symbolTable.getFunc(ident)
}

func (ss *ScopeStack) AddFunc(ident string, rec *IRFunction) error {
	// look at only the global scope, because function can only be declared at the top level.
	return ss.Scopes[0].symbolTable.addFunc(ident, rec)
}

func (ss *ScopeStack) AddDatatype(ident string, rec *IRDatatype) error {
	// look at only the global scope, because function can only be declared at the top level.
	return ss.Scopes[0].symbolTable.addDatatype(ident, rec)
}

func (ss *ScopeStack) GetDatatype(ident string) *IRDatatype {
	return ss.Scopes[0].symbolTable.getDatatype(ident)
}
