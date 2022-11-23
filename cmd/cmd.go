package cmd

import (
	"quoi/generator"
	"quoi/lexer"
	"quoi/parser"
	"quoi/runner"
)

func Run(filename, quoiCode string) error {
	l := lexer.New(quoiCode)
	p := parser.New(l)
	// TODO analyzer
	g := generator.New(p.Parse())
	r, err := runner.New(l, p, g, runner.Opts{Filename: filename})
	if err != nil {
		return err
	}
	r.Run()
	return nil
}
