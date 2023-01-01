package generator

import (
	"fmt"
	"quoi/analyzer"
	"quoi/lexer"
	"quoi/parser"
	"testing"
)

func setup(input string) *Generator {
	l := lexer.New(input)
	if len(l.Errs) > 0 {
		for _, v := range l.Errs {
			fmt.Printf("lexer err: %s\n", v.Msg)
		}
	}
	p := parser.New(l)
	if len(p.Errs) > 0 {
		for _, v := range p.Errs {
			fmt.Printf("parser err: %s\n", v.Msg)
		}
	}
	a := analyzer.New(p.Parse())
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			fmt.Printf("analyzer err: %s\n", v.Msg)
		}
	}
	prg := a.Analyze()
	return New(prg)
}

func Test1(t *testing.T) {
	input := `
		int n = 5.

		;User u = User { name="User 1" }.

		;bool x = (and true (or true (lt 5 6) )).
	`
	fmt.Println(setup(input).Generate())
}
