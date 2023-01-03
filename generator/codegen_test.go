package generator

import (
	"fmt"
	"os"
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
		os.Exit(1)
	}
	p := parser.New(l)

	parsed := p.Parse()
	if len(p.Errs) > 0 {
		for _, v := range p.Errs {
			fmt.Printf("parser err: %s\n", v.Msg)
		}
		os.Exit(1)
	}
	a := analyzer.New(parsed)
	prg := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			fmt.Printf("analyzer err: %s\n", v.Msg)
		}
		os.Exit(1)
	}
	return New(prg)
}

func Test1(t *testing.T) {
	input := `
		int n = 5.

		datatype User {
			string name
			int age
			bool is_alive
		}

		User u = User { name="User 1" age=44 is_alive=true }.

		bool x = (and true (or true (lt 5 6) )).

		int nn = -15.

		int n2 = (+ 1 2 4 5 6 176 nn (* 7 8 9 (/ 1290 65))).

		if true {
			int x = 5.
		} elseif false { 
			string y = "Hello".
		} elseif (= 5 (+ 1 1)) {
			User fACXAQ_1351_x = User {}.
		} else {
			if (not (lt 5 6)) { bool z = true. }
		}
		
	`
	fmt.Println(setup(input).Generate())
}
