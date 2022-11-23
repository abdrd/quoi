package analyzer

import (
	"fmt"
	"quoi/ast"
)

type Err struct {
	Line, Column int
	Msg          string
}

type Analyzer struct {
	program *ast.Program
	Errs    []Err
}

func New(program *ast.Program) *Analyzer {
	return &Analyzer{program: program}
}

func (a *Analyzer) errorf(line, col int, msgf string, args ...interface{}) {
	a.Errs = append(a.Errs, Err{
		Line:   line,
		Column: col,
		Msg:    fmt.Sprintf(msgf, args...),
	})
}

/*
	IR (Intermediate Representation)
-----------------------------------------
	declarations

		globals :
			a		int						5
			b 		string  				"hello"
			c 		bool					true
			greet 	fn (string -- string) 1 :
				name string $1
				x_51f string String::concat("Hello ", name)
				if (lt 5 6) :
					y_T61 string "Hello"
					return y_T61.
				return x_51f

			d list-string 3		["A", "B", "C"]

			Int::from_string fn (string -- int) 1 :

			x 	fn (list-string -- string) 1 :
				@listindex $1 0
			add fn (int int -- int) 1 :
				gyF_1 int	 @add $1 $2
				return gyF_1
*/

func (a *Analyzer) Analyze() *IRProgram {
	program := &IRProgram{}
	for _, s := range a.program.Stmts {
		switch s := s.(type) {
		case *ast.VariableDeclarationStatement:
			if ir := a.analyzeVarDecl(s); ir != nil {
				program.Push(ir)
			}
		case *ast.ListVariableDeclarationStatement:
			if ir := a.analyzeListVarDecl(s); ir != nil {
				program.Push(ir)
			}
		}
	}
}

func (a *Analyzer) analyzeVarDecl(s *ast.VariableDeclarationStatement) *IRVariable {
	ir := &IRVariable{}
	ir.Name = s.Ident.String()
	panic(":)")
}

func (a *Analyzer) analyzeListVarDecl(s *ast.ListVariableDeclarationStatement) *IRVariable {
	panic(":))")
}
