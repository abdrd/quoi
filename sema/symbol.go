package sema

import (
	"log"
	"quoi/ast"
)

type symbol interface {
	ident() string
}

type symbolTable struct {
	table map[string]symbol
}

func newSymbolTable() *symbolTable {
	return &symbolTable{table: make(map[string]symbol)}
}

type scope struct {
	st *symbolTable
}

func (s *scope) get(ident string) symbol {
	if s, found := s.st.table[ident]; found {
		return s
	}
	return nil
}

func (s *scope) push(sym symbol) bool {
	if s.get(sym.ident()) != nil {
		return false
	}
	s.st.table[sym.ident()] = sym
	return true
}

func newScope(st *symbolTable) *scope {
	return &scope{st: st}
}

type scopeStack struct {
	stack []*scope
}

func newScopeStack() *scopeStack {
	return &scopeStack{stack: []*scope{newScope(newSymbolTable())}}
}

func (s *scopeStack) push(scope *scope) {
	s.stack = append(s.stack, scope)
}

// initialize a new scope on the top of the stack
func (s *scopeStack) enterScope() {
	s.push(newScope(newSymbolTable()))
}

// terminates the current scope
func (s *scopeStack) exitScope() {
	stackLen := len(s.stack)
	if stackLen < 1 {
		return
	}
	s.stack = s.stack[:stackLen-1]
}

// searches the symbol sym in any scope, and returns the first found
func (s *scopeStack) findSymbol(ident string) symbol {
	for i := 0; i < len(s.stack); i++ {
		curScope := s.stack[i]
		if s := curScope.get(ident); s != nil {
			return s
		}
	}
	return nil
}

// adds the new symbol sym to the current top-scope
func (s *scopeStack) addSymbol(sym symbol) {
	stackLen := len(s.stack)
	if stackLen < 1 {
		return
	}
	if ok := s.stack[stackLen-1].push(sym); !(ok) {
		// TODO panicking for now
		log.Fatalf("identifier '%s' already exists\n", sym.ident())
	}
}

// returns true is the symbol 'sym' is defined in the current top-scope.
func (s *scopeStack) checkScope(sym symbol) bool {
	stackLen := len(s.stack)
	if stackLen < 1 {
		return false
	}
	return s.stack[stackLen].get(sym.ident()) != nil
}

type varDecl struct {
	*ast.VariableDeclarationStatement
}

func (v *varDecl) ident() string {
	return v.Ident.String()
}

type listDecl struct {
	*ast.ListVariableDeclarationStatement
}

func (l *listDecl) ident() string {
	return l.Name.String()
}
