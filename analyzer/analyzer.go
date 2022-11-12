package analyzer

import (
	"fmt"
	"quoi/ast"
)

type Err struct {
	Line, Col uint
	Msg       string
}

func newErr(line, col uint, msgf string, args ...interface{}) Err {
	return Err{Line: line, Col: col, Msg: fmt.Sprintf(msgf, args...)}
}

type Analyzer struct {
	program *ast.Program
	Errs    []Err
}

func NewAnalyzer(program *ast.Program) *Analyzer {
	return &Analyzer{program: program}
}

func (a *Analyzer) errorf(line, col uint, msgf string, args ...interface{}) {
	a.Errs = append(a.Errs, newErr(line, col, msgf, args...))
}
