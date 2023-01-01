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
	nodes := g.prg.IRStatements
	for _, n := range nodes {
		switch n := n.(type) {
		case *analyzer.IRVariable:
			g.vardecl(n)
		}
	}
	g.assemble()
	return g.code()
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
		b.writef("%s{\n\t", e.Name)
		for k, v := range e.FieldsAndValues {
			b.writef("%s: %s,\n\t", k, g.expr(v))
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
			for _, v := range e.Operands {
				b.writef("%s %s", e.Operator, v)
			}
			b.writef(")")
		case "gt", "gte", "lt", "lte", "and", "or":
			m := map[string]string{
				"gt": ">", "gte": ">=", "lt": "<", "lte": "<=", "and": "&&", "or": "||",
			}
			b.writef("(%s %s %s)", e.Operands[0], m[e.Operator], e.Operands[1])
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
	g.w("var %s %s = %s", d.Name, d.Type, g.expr(d.Value))
}
