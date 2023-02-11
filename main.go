package main

import (
	"fmt"
	"log"
	"os"
	"quoi/analyzer"
	"quoi/cmd"
	"quoi/generator"
	"quoi/lexer"
	"quoi/parser"
)

func readFile(fname string) ([]byte, error) {
	return os.ReadFile(fname)
}

func compile(src string) string {
	l := lexer.New(src)
	p := parser.New(l)
	prg := p.Parse()
	if len(p.Errs) > 0 {
		for _, v := range p.Errs {
			fmt.Println(v)
		}
		os.Exit(1)
	}
	a := analyzer.New(prg)
	irprg := a.Analyze()
	if len(a.Errs) > 0 {
		for _, v := range a.Errs {
			fmt.Println(v)
		}
		os.Exit(1)
	}
	g := generator.New(irprg)
	return g.Generate()
}

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatalln("qc: not enough arguments")
	}
	fname := os.Args[1]
	src, err := readFile(fname)
	if err != nil {
		log.Fatalf("qc: read file '%s': %s\n", fname, err.Error())
	}
	switch len(args) {
	case 2:
		cmd.RunProgram(compile(string(src)))
	case 3:
		command := os.Args[2]
		switch command {
		case "-go":
			cmd.GetGo(string(src))
		case "-exe":
			cmd.GetExecutable(string(src))
		case "-stdout":
			fmt.Println(compile(string(src)))
		default:
			log.Fatalf("qc: unknown option `%s`\n", command)
		}
	}
}
