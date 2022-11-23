package runner

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"quoi/generator"
	"quoi/lexer"
	"quoi/parser"
	"quoi/token"
	"strings"
)

type Runner struct {
	l    *lexer.Lexer
	p    *parser.Parser
	g    *generator.Generator
	opts Opts
}

type Opts struct {
	EmitGo, EmitTokens, EmitAST bool // if all of these are false, then just run 'code'
	Filename                    string
}

func New(l *lexer.Lexer, p *parser.Parser, g *generator.Generator, opts Opts) (*Runner, error) {
	nilComponent := ""
	if l == nil {
		nilComponent = "lexer"
	}
	if p == nil {
		nilComponent = "parser"
	}
	if g == nil {
		nilComponent = "generator"
	}
	if nilComponent != "" {
		return nil, fmt.Errorf("runner.New:  nil %s", nilComponent)
	}
	if len(opts.Filename) == 0 {
		opts.Filename = "default.quoi"
	}
	r := &Runner{l: l, p: p, g: g, opts: opts}
	return r, nil
}

func formatToken(tok token.Token) string {
	return fmt.Sprintf("@%d:%d(%s %s)\n", tok.Line, tok.Col, tok.Type.String(), tok.Literal)
}

func writeFile(filename, contents string) error {
	return os.WriteFile(filename, []byte(contents), 0777)
}

func (r *Runner) Run() {
	if r.opts.EmitTokens {
		var buf strings.Builder
		for {
			tok := r.l.Next()
			if tok.Type == token.EOF {
				break
			}
			buf.WriteString(formatToken(tok))
		}
		filename := fmt.Sprintf("%s.tokens", r.opts.Filename)
		if err := writeFile(filename, buf.String()); err != nil {
			log.Fatalf("*Runner.Run: %s\n", err.Error())
		}
		return
	}
	if r.opts.EmitAST {
		// TODO emit AST
		fmt.Println("TODO emit AST")
		return
	}
	// TODO access stderr (error messages) of `go` process
	r.g.Generate()
	code := r.g.Code()
	filename := fmt.Sprintf("%s_transpiled.go", r.opts.Filename)
	if err := writeFile(filename, code); err != nil {
		log.Fatalf("*Runner.Run: write go code: %s\n", err.Error())
	}
	if r.opts.EmitGo {
		return
	}
	cmd := exec.Command("go", "run", filename)
	rc, err2 := cmd.StderrPipe()
	err := cmd.Start()
	if err2 != nil {
		log.Fatalf("StderrPipe: %s\n", err2.Error())
	}
	if err != nil {
		log.Print("*Runner.Run: run go code: ")
		sc := bufio.NewScanner(rc)
		for sc.Scan() {
			log.Printf("%s\n", sc.Text())
		}
		return
	}
}
