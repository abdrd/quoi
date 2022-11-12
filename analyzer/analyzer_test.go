package analyzer

import (
	"quoi/lexer"
	"quoi/parser"
	"testing"
)

func Test1(t *testing.T) {
	input := `
		;fun a() -> int, bool {
		;	return true, true.
		;}

		;fun b() -> bool {
		;	;return 1.
		;}

		fun c() -> string, bool {
			if (lt 5 6) {
				if (= 1 1) { return "true", true. }
				return "hello", true.
			}
			block 
				if (= 5 5) {
					if (not false) {
						return "Hehe", "hehe", 1.
					}
					return 1.
				}
			end
		}
	`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.Parse()
	for _, v := range p.Errs {
		t.Logf("Parser err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
	}
	a := NewAnalyzer(program)
	a.FirstPass()
	for _, v := range a.Errs {
		t.Logf("Err: %d:%d -- %s\n", v.Line, v.Col, v.Msg)
	}
}
