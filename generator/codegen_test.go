package generator

import (
	"fmt"
	"quoi/lexer"
	"quoi/parser"
	"testing"
)

func _generate(input string) {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.Parse()
	for _, v := range p.Errs {
		fmt.Printf("Parser err: %d:%d -- %s\n", v.Line, v.Column, v.Msg)
	}
	g := New(program)
	g.Generate()
	fmt.Println(g.Code())
}

func Test1(t *testing.T) {
	input := `
		fun add(int num1, int num2) -> int {
			return (+ num1 num2).
		}

		;print( (+ 1 2) ).
	`
	_generate(input)
}

func Test2(t *testing.T) {
	input := `
		PRINT( (+ 1 2 3 4 5 6 7 8 ) ).
		PRINT( (* 1 2 3 4 5 6 7 8 ) ).
		PRINT( (/ 1 2 3 4 5 6 7 8 ) ).
		PRINT( (- 1 2 3 4 5 6 7 8 ) ).
	`
	_generate(input)
}
