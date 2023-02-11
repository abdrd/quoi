package generator

import (
	"fmt"
	"math/rand"
	"quoi/analyzer"
	"strings"
	"time"
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
	prg                  *analyzer.IRProgram
	header, global, body *stringBuilder
	addedImports         map[string]bool
}

func New(prg *analyzer.IRProgram) *Generator {
	g := &Generator{
		prg: prg,

		header: newStringBuilder(),
		body:   newStringBuilder(),
		// declarations
		global:       newStringBuilder(),
		addedImports: make(map[string]bool),
	}
	g.header.writef("package main\n\nimport(\n")
	g.body.writef("func main() {\n")
	return g
}

func (g *Generator) addImport(pkg string) {
	if g.addedImports[pkg] {
		return
	}
	g.addedImports[pkg] = true
	g.header.writef("\t\"%s\"\n", pkg)
}

// body
func (g *Generator) w(strf string, args ...interface{}) {
	g.body.writef(strf, args...)
}

// function, and datatype declarations
func (g *Generator) wd(strf string, args ...interface{}) {
	g.global.writef(strf, args...)
}

func (g *Generator) assemble() {
	g.header.writef(")\n\n")
	g.body.writef("\n}\n")
	g.header.writef(g.global.b.String())
	g.header.writef(g.body.b.String())
}

func (g *Generator) code() string {
	g.addRuntimeFunctions()
	return g.header.b.String()
}

func (g *Generator) Generate() string {
	for _, n := range g.prg.Stmts {
		g.stmt(n)
	}
	g.assemble()
	return g.code()
}

func (g *Generator) addRuntimeFunctions() {
	// add function definitions for stdlib functions. mangle their names to
	// avoid redefinitions by the user.
	rand.Seed(time.Now().UnixNano())
	var randomFn = func(fnName string) string {
		return fmt.Sprintf("%s_%d", fnName, rand.Intn(10000))
	}
	// TODO
	_ = randomFn
}

func (g *Generator) stmt1(s analyzer.IRStatement) string {
	switch s := s.(type) {
	case *analyzer.IRVariable:
		return g.vardecl(s)
	case *analyzer.IRIf:
		return g.if_(s)
	case *analyzer.IRBlock:
		return g.block(s)
	case *analyzer.IRDatatype:
		return g.dt(s)
	case *analyzer.IRFunction:
		return g.fun(s)
	case *analyzer.IRFunctionCall:
		return g.funcall(s)
	case *analyzer.IRFunctionCallFromNamespace:
		return g.funcallns(s)
	case *analyzer.IRLoop:
		return g.loop(s)
	case *analyzer.IRReassigment:
		return g.reas(s)
	case *analyzer.IRReturn:
		return g.ret(s)
	case *analyzer.IRBreak:
		return "break\n"
	case *analyzer.IRContinue:
		return "continue\n"
	}
	panic("unknown statement " + s.String())
}

func (g *Generator) stmt(s analyzer.IRStatement) {
	switch s.(type) {
	case *analyzer.IRFunction, *analyzer.IRDatatype:
		g.wd(g.stmt1(s))
	default:
		g.w(g.stmt1(s))
	}
}

func (g *Generator) exprList(ex []analyzer.IRExpression, lenArgs int) string {
	b := newStringBuilder()
	for i, v := range ex {
		b.writef(g.expr(v))
		if i == lenArgs-1 {
			continue
		}
		b.writef(", ")
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
		case "'":
			var idx string
			idx = g.expr(e.Operands[1])
			b.writef("%s[%s]", g.expr(e.Operands[0]), idx)
			// bounds checking
			b.writef("\nif %s > len(%s)-1 { panic(\"index '%s' is out of range\") }\n", idx, g.expr(e.Operands[0]), idx)
		case "set":
			b.writef("%s.%s = %s\n", g.expr(e.Operands[0]), g.expr(e.Operands[1]), g.expr(e.Operands[2]))
		case "get":
			b.writef("%s.%s\n", g.expr(e.Operands[0]), g.expr(e.Operands[1]))
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

func (g *Generator) vardecl(d *analyzer.IRVariable) string {
	typ := d.Type
	if strings.Contains(typ, "list-") {
		typ = "[]" + strings.Split(typ, "list-")[1]
	}
	return fmt.Sprintf("\nvar %s %s = %s\n", d.Name, typ, g.expr(d.Value))
}

func (g *Generator) if_(d *analyzer.IRIf) string {
	b := newStringBuilder()
	b.writef("if %s {\n\t", g.expr(d.Cond))
	for _, v := range d.Block {
		if v, ok := v.(*analyzer.IRElseIf); ok {
			b.writef(g.elseif(v))
			continue
		}
		b.writef(g.stmt1(v))
	}
	b.writef("}")
	if d.Alternative != nil {
		b.writef(g.elseif(d.Alternative))
	}
	if d.Default != nil {
		b.writef(g.else_(d.Default))
	}
	return b.String()
}

func (g *Generator) elseif(d *analyzer.IRElseIf) string {
	b := newStringBuilder()
	b.writef(" else if %s {\n\t", g.expr(d.Cond))
	for _, v := range d.Block {
		if v, ok := v.(*analyzer.IRElseIf); ok {
			b.writef(g.elseif(v))
			continue
		}
		b.writef(g.stmt1(v))
	}
	b.writef("\n}")
	if d.Alternative != nil {
		b.writef(g.elseif(d.Alternative))
	}
	if d.Default != nil {
		b.writef(g.else_(d.Default))
	}
	return b.String()
}

func (g *Generator) else_(d *analyzer.IRElse) string {
	b := newStringBuilder()
	b.writef(" else {\n\t")
	for _, v := range d.Block {
		if v, ok := v.(*analyzer.IRElseIf); ok {
			b.writef(g.elseif(v))
			continue
		}
		b.writef(g.stmt1(v))
	}
	b.writef("\n}")
	return b.String()
}

func (g *Generator) block(d *analyzer.IRBlock) string {
	b := newStringBuilder()
	b.writef("{\n\t")
	for _, v := range d.Stmts {
		b.writef(g.stmt1(v))
	}
	b.writef("\n}\n")
	return b.String()
}

func (g *Generator) dt(d *analyzer.IRDatatype) string {
	b := newStringBuilder()
	b.writef("type %s struct {\n", d.Name)
	for _, v := range d.Fields {
		b.writef("\t%s %s\n", v.Name, v.Type)
	}
	b.writef("}\n")
	return b.String()
}

func (g *Generator) fun(d *analyzer.IRFunction) string {
	b := newStringBuilder()
	b.writef("func %s(", d.Name)
	for i, v := range d.Takes {
		b.writef("%s %s", d.ParamNames[i], v)
		if i != len(d.Takes)-1 {
			b.writef(", ")
		}
	}
	b.writef(") ")
	if d.ReturnsCount > 0 {
		b.writef("(")
		for i, v := range d.Returns {
			b.writef(v)
			if i != len(d.Takes)-1 {
				b.writef(", ")
			}
		}
		b.writef(") ")
	}
	b.writef("{\n")
	for _, v := range d.Block {
		b.writef(g.stmt1(v))
	}
	b.writef("\n}\n")
	return b.String()
}

func (g *Generator) funcall(d *analyzer.IRFunctionCall) string {
	b := newStringBuilder()
	b.writef("%s(", d.Name)
	b.writef(g.exprList(d.Takes, d.TakesCount))
	b.writef(")\n")
	return b.String()
}

func (g *Generator) funcallns(d *analyzer.IRFunctionCallFromNamespace) string {
	b := newStringBuilder()
	ns := d.Namespace
	nsim := map[string]string{
		"Stdout": "fmt",
		"Math":   "math",
		"String": "strings",
	}
	nsfm := map[string]string{
		"println": "Println",
		"print":   "Print",
		"mod":     "Mod10",
		"pow":     "Pow",
		"sqrt":    "Sqrt",
	}
	pkg := nsim[ns]
	g.addImport(pkg)
	b.writef("%s.", pkg)
	d.IRFunctionCall.Name = nsfm[d.Name]
	b.writef(g.funcall(&d.IRFunctionCall))
	return b.String()
}

func (g *Generator) loop(d *analyzer.IRLoop) string {
	b := newStringBuilder()
	b.writef("for %s {\n", g.expr(d.Cond))
	for _, v := range d.Stmts {
		b.writef(g.stmt1(v))
	}
	b.writef("}\n")
	return b.String()
}

func (g *Generator) ret(d *analyzer.IRReturn) string {
	b := newStringBuilder()
	b.writef("return %s\n", g.exprList(d.ReturnValues, d.ReturnCount))
	return b.String()
}

func (g *Generator) reas(d *analyzer.IRReassigment) string {
	b := newStringBuilder()
	b.writef("%s = %s\n", d.Name, g.expr(d.NewValue))
	return b.String()
}
