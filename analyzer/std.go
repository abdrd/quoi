package analyzer

import (
	"quoi/ast"
	"quoi/lexer"
	"quoi/parser"
	"strings"
)

// The standard library consisting of different namespaces

const STDOUT = `
		fun Stdout_println(string s) -> {}
		fun Stdout_print(string s) -> {}
	`

const MATH = `
		fun Math_mod(int n, int n2) -> int {}
		fun Math_pow(int n, int n2) -> int {}
		fun Math_sqrt(int n) -> int {}
	`

const STRING = `
		fun String_from_int(int n) -> string {}
		fun String_from_bool(bool b) -> string {}
		fun String_concat(string s, string s2) -> string {}
		fun String_index(string s, string ch) -> int {}
	`

const INT = `
		fun Int_from_string(string s) -> int {}
	`

const LIST = `
		fun List_replace_int(listof int nx, int idx, int new_val) -> listof int {}
		fun List_replace_string(listof string strx, int idx, string new_val) -> listof string {}
		fun List_replace_bool(listof bool bx, int idx, bool new_val) -> listof bool {}
	`

type StandardLibrary struct {
	STDOUT, MATH, STRING, INT, LIST map[string]*IRFunction
}

func InitStandardLibrary(a *Analyzer) *StandardLibrary {
	s := &StandardLibrary{
		STDOUT: make(map[string]*IRFunction),
		MATH:   make(map[string]*IRFunction),
		STRING: make(map[string]*IRFunction),
		INT:    make(map[string]*IRFunction),
		LIST:   make(map[string]*IRFunction),
	}
	a.std = s

	std := STDOUT + MATH + STRING + INT + LIST
	l := lexer.New(std)
	p := parser.New(l)
	prg := p.Parse()

	for _, v := range prg.Stmts {
		switch v := v.(type) {
		case *ast.FunctionDeclarationStatement:
			splitted := strings.SplitN(v.Name.String(), "_", 2)
			ns, name := splitted[0], splitted[1]
			a.registerStdFuncSignature(ns, name, v)
		}
	}
	return s
}

func (s *StandardLibrary) GetFunc(namespace, name string) *IRFunction {
	switch namespace {
	case "Stdout":
		return s.STDOUT[name]
	case "Math":
		return s.MATH[name]
	case "String":
		return s.STRING[name]
	case "List":
		return s.LIST[name]
	case "Int":
		return s.INT[name]
	}
	return nil
}

func (s *StandardLibrary) AddFunc(namespace string, name string, decl *IRFunction) {
	switch namespace {
	case "Stdout":
		s.STDOUT[name] = decl
	case "Math":
		s.MATH[name] = decl
	case "String":
		s.STRING[name] = decl
	case "List":
		s.LIST[name] = decl
	case "Int":
		s.INT[name] = decl
	}
}
