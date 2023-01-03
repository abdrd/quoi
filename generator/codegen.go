package generator

import (
	"fmt"
	"quoi/analyzer"
	"strings"
)

type stringBuilder struct {
	b strings.Builder
}

func newStringBuilder() *stringBuilder {
	return &stringBuilder{b: strings.Builder{}}
}

func (s *stringBuilder) writef(strf string, args ...interface{}) {
	s.b.WriteString(fmt.Sprintf(strf, args...))
}

func (s *stringBuilder) String() string {
	return s.b.String()
}

// Go code producer
type Generator struct {
	prg          *analyzer.IRProgram
	header, body *stringBuilder
}

func New(prg *analyzer.IRProgram) *Generator {
	g := &Generator{
		prg: prg,

		header: newStringBuilder(),
		body:   newStringBuilder(),
	}
	g.header.writef("package main\n\nimport(\n")
	return g
}

func (g *Generator) addImport(pkg string) {
	g.header.writef("\t\"%s\"\n", pkg)
}

func (g *Generator) closeImport() {
	g.header.writef(")\n")
}

func (g *Generator) w(strf string, args ...interface{}) {
	g.body.writef(strf, args...)
}

func (g *Generator) assemble() {
	g.header.writef(")\n\n")
	g.header.writef(g.body.b.String())
}

func (g *Generator) code() string {
	return g.header.b.String()
}

func (g *Generator) Generate() string {
	for _, n := range g.prg.Stmts {
		g.stmt(n)
	}
	g.assemble()
	return g.code()
}

func (g *Generator) stmt(s analyzer.IRStatement) {
	switch s := s.(type) {
	case *analyzer.IRVariable:
		g.vardecl(s)
	case *analyzer.IRIf:
		g.if_(s)
	}
}

func (g *Generator) exprList(ex []analyzer.IRExpression, lenArgs int) string {
	b := newStringBuilder()
	for i, v := range ex {
		putComma := i != lenArgs
		b.writef(g.expr(v))
		if putComma {
			b.writef(", ")
		}
	}
	return b.String()
}

func (g *Generator) expr(e analyzer.IRExpression) string {
	switch e := e.(type) {
	case *analyzer.IRString:
		return fmt.Sprintf("\"%s\"", e.Value)
	case *analyzer.IRBoolean:
		return e.Value
	case *analyzer.IRInt:
		return e.Value
	case *analyzer.IRDatatypeLiteral:
		b := newStringBuilder()
		b.writef("%s{\n", e.Name)
		for k, v := range e.FieldsAndValues {
			b.writef("\t%s: %s,\n", k, g.expr(v))
		}
		b.writef("}")
		return b.String()
	case *analyzer.IRList:
		b := newStringBuilder()
		b.writef("[]%s{ ", e.Type)
		b.writef(g.exprList(e.Value, len(e.Value)))
		b.writef(" }")
		return b.String()
	case *analyzer.IRFunctionCall:
		b := newStringBuilder()
		b.writef("%s(", e.Name)
		b.writef(g.exprList(e.Takes, len(e.Takes)))
		b.writef(")")
		return b.String()
	case *analyzer.IRFunctionCallFromNamespace:
		b := newStringBuilder()
		var ns string
		switch e.Namespace {
		case "Stdout":
			ns = "fmt"
		default:
			ns = "TODO: UNKNOWN NAMESPACE"
		}
		g.addImport(ns)
		b.writef("%s.%s", ns, g.expr(&e.IRFunctionCall))
		return b.String()
	case *analyzer.IRIndex:
		b := newStringBuilder()
		// TODO bounds-checking
		var idx string
		if v, ok := e.Expr.(*analyzer.IRPrefExpr); ok {
			idx = g.expr(v.Operands[1])
		}
		b.writef("%s[%s]", e.Expr, idx)
		return b.String()
	case *analyzer.IRNot:
		b := newStringBuilder()
		b.writef("!(%s)", g.expr(e.Expr))
		return b.String()
	case *analyzer.IRPrefExpr:
		b := newStringBuilder()
		switch e.Operator {
		case "+", "-", "/", "*":
			b.writef("(")
			for i, v := range e.Operands {
				b.writef("%s", g.expr(v))
				if i != len(e.Operands)-1 {
					b.writef(" %s ", e.Operator)
				}
			}
			b.writef(")")
		case "gt", "gte", "lt", "lte", "and", "or", "=":
			m := map[string]string{
				"gt": ">", "gte": ">=", "lt": "<", "lte": "<=", "and": "&&", "or": "||", "=": "==",
			}
			b.writef("(%s %s %s)", g.expr(e.Operands[0]), m[e.Operator], g.expr(e.Operands[1]))
		case "not":
			b.writef("!(%s)", g.expr(e.Operands[0]))
		default:
			b.writef("UNKNOWN OPERATOR %s", e.Operator)
		}
		return b.String()
	case *analyzer.IRVariableReference:
		b := newStringBuilder()
		b.writef("%s", e.Name)
		return b.String()
	}
	return "NOT_IMPLEMENTED: " + e.String()
}

func (g *Generator) vardecl(d *analyzer.IRVariable) {
	g.w("var %s %s = %s\n", d.Name, d.Type, g.expr(d.Value))
}

func (g *Generator) if_(d *analyzer.IRIf) {
	g.w("if %s {\n\t", g.expr(d.Cond))
	for _, v := range d.Block {
		if v, ok := v.(*analyzer.IRElseIf); ok {
			g.elseif(v)
			continue
		}
		g.stmt(v)
	}
	g.w("}")
	if d.Alternative != nil {
		g.elseif(d.Alternative)
	}
	if d.Default != nil {
		g.else_(d.Default)
	}
}

func (g *Generator) elseif(d *analyzer.IRElseIf) {
	g.w(" else if %s {\n\t", g.expr(d.Cond))
	for _, v := range d.Block {
		if v, ok := v.(*analyzer.IRElseIf); ok {
			g.elseif(v)
			continue
		}
		g.stmt(v)
	}
	g.w("\n}")
	if d.Alternative != nil {
		g.elseif(d.Alternative)
	}
	if d.Default != nil {
		g.else_(d.Default)
	}
}

func (g *Generator) else_(d *analyzer.IRElse) {
	g.w(" else {\n\t")
	for _, v := range d.Block {
		if v, ok := v.(*analyzer.IRElseIf); ok {
			g.elseif(v)
			continue
		}
		g.stmt(v)
	}
	g.w("\n}")
}
