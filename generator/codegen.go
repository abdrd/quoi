package generator

import (
	"fmt"
	"quoi/ast"
	"quoi/token"
	"strings"
)

// Quoi -> Go
type Generator struct {
	program      *ast.Program
	header, body strings.Builder
}

func New(program *ast.Program) *Generator {
	g := &Generator{program: program}
	g.header.WriteString("package main\n\nimport(\n\t")
	return g
}

func (g *Generator) ws() {
	g.body.WriteByte(' ')
}

func (g *Generator) newline() {
	g.body.WriteByte('\n')
}

func (g *Generator) newlTab() {
	g.w("\n\t")
}

func (g *Generator) commaif(cond bool) {
	if cond {
		g.w(", ")
	}
}

func (g *Generator) brackets() {
	g.w("[]")
}

func (g *Generator) sym(char byte) {
	g.body.WriteByte(char)
}

func shouldPutComma(idx, length int) bool {
	return idx < length && idx != length-1
}

// g.body.WriteString
func (g *Generator) w(str string) {
	g.body.WriteString(str)
}

func (g *Generator) Generate() {
	for _, v := range g.program.Stmts {
		g.genStatement(v)
	}
	g.header.WriteByte(')')
	g.header.WriteByte('\n')
}

func (g *Generator) Code() string {
	g.header.WriteString(g.body.String())
	return g.header.String()
}

func (g *Generator) genImport(pkg string) {
	g.header.WriteString(fmt.Sprintf("\"%s\"", pkg))
}

func (g *Generator) genParam(param ast.FunctionParameter) {
	g.w(param.Name.String())
	g.ws()
	if param.IsList {
		g.brackets()
		g.w(param.TypeOfList.Literal)
	} else {
		g.w(param.Tok.Literal)
	}
}

func (g *Generator) genFuncDecl(fn *ast.FunctionDeclarationStatement) {
	g.w("func ")
	g.w(fn.Name.String())
	g.ws()
	if len(fn.Params) > 0 {
		g.sym('(')
		for i, v := range fn.Params {
			g.genParam(v)
			g.commaif(shouldPutComma(i, len(fn.Params)))
		}
		g.sym(')')
	}
	g.ws()
	if fn.ReturnCount > 0 {
		needParens := fn.ReturnCount > 1
		if needParens {
			g.sym('(')
		}
		for i, v := range fn.ReturnTypes {
			if v.IsList {
				g.brackets()
				g.w(v.TypeOfList.Literal)
			} else {
				g.w(v.Tok.Literal)
			}
			g.commaif(shouldPutComma(i, len(fn.ReturnTypes)))
		}
		if needParens {
			g.sym(')')
		}
	}
	g.ws()
	g.sym('{')
	g.newline()
	for _, v := range fn.Stmts {
		g.genStatement(v)
	}
	g.newline()
	g.sym('}')
	g.newline()
}

func (g *Generator) genStatement(s ast.Statement) {
	switch s := s.(type) {
	case *ast.VariableDeclarationStatement:
		g.genVarDecl(s)
	case *ast.FunctionCall:
		g.genFunCall(s)
	case *ast.FunctionDeclarationStatement:
		g.genFuncDecl(s)
	case *ast.IfStatement:
		g.genIfStmt(s)
	case *ast.LoopStatement:
		g.genLoop(s)
	case *ast.ReassignmentStatement:
		g.genReassignment(s)
	case *ast.ReturnStatement:
		g.genReturn(s)
	case *ast.SubsequentVariableDeclarationStatement:
		g.genSubseq(s)
	case *ast.BreakStatement:
		g.genBreak()
	case *ast.ContinueStatement:
		g.genContinue()
	case *ast.DatatypeDeclaration:
		g.genDatatypeDecl(s)
	case *ast.ListVariableDeclarationStatement:
		g.genListDecl(s)
	}
}

func (g *Generator) genExpr(expr ast.Expr) {
	switch expr := expr.(type) {
	case *ast.IntLiteral:
		g.genInt(expr)
	case *ast.StringLiteral:
		g.genString(expr)
	case *ast.BoolLiteral:
		g.w(expr.String())
	case *ast.DatatypeLiteral:
		g.genDatatypeLit(expr)
	case *ast.FunctionCall:
		g.genFunCall(expr)
	case *ast.Identifier:
		g.genIdent(expr)
	case *ast.PrefixExpr:
		g.genPrefExpr(expr)
		// TODO list literal
		/*
			case *ast.ListLiteral:
				g.genListLit(expr)
		*/
	}
}

func (g *Generator) genInt(lit *ast.IntLiteral) {
	g.w(lit.String())
}

func (g *Generator) genString(lit *ast.StringLiteral) {
	g.w(lit.String())
}

func (g *Generator) genListLit(lit *ast.ListLiteral, type_ string) {
	g.brackets()
	g.w(type_)
	g.sym('{')
	g.genExprList(lit.Elems)
	g.sym('}')
}

func (g *Generator) genPrefExpr(expr *ast.PrefixExpr) {
	if expr.Tok.Type == token.SINGLE_QUOTE {
		g.genListIndex(expr)
		return
	}
	if expr.Tok.Type == token.NOT {
		g.genNotOp(expr)
		return
	}
	g.sym('(')
	op := ""
	switch expr.Tok.Type {
	case token.LT:
		op = "<"
	case token.LTE:
		op = "<="
	case token.GT:
		op = ">"
	case token.GTE:
		op = ">="
	case token.AND:
		op = "&&"
	case token.OR:
		op = "||"
	default:
		op = expr.Tok.Literal
	}
	for i, v := range expr.Args {
		isLast := i == len(expr.Args)-1
		g.genExpr(v)
		if !(isLast) {
			g.w(op)
		}
	}
	g.sym(')')
}

func (g *Generator) genListIndex(expr *ast.PrefixExpr) {}

func (g *Generator) genNotOp(expr *ast.PrefixExpr) {
	g.sym('!')
	g.sym('(')
	g.genExpr(expr.Args[0])
	g.sym(')')
}

func (g *Generator) genVarDecl(decl *ast.VariableDeclarationStatement) {
	g.w("var ")
	g.w(decl.Ident.String())
	g.ws()
	g.w(decl.Tok.Literal)
	g.ws()
	g.sym('=')
	g.ws()
	g.genExpr(decl.Value)
	g.newline()
}

func (g *Generator) genStmtList(stmts []ast.Statement) {
	stmtsLen := len(stmts)
	for i, v := range stmts {
		g.genStatement(v)
		g.commaif(shouldPutComma(i, stmtsLen))
	}
}

func (g *Generator) genExprList(exprs []ast.Expr) {
	exprsLen := len(exprs)
	for i, v := range exprs {
		g.genExpr(v)
		g.commaif(shouldPutComma(i, exprsLen))
	}
}

func (g *Generator) genFunCall(call *ast.FunctionCall) {
	g.w(call.Ident.String())
	g.sym('(')
	g.genExprList(call.Args)
	g.sym(')')
	g.newline()
}

func (g *Generator) genIfStmt(stmt *ast.IfStatement) {
	g.w("if ")
	g.genExpr(stmt.Cond)
	g.ws()
	g.sym('{')
	g.newlTab()
	g.genStmtList(stmt.Stmts)
	if stmt.Alternative != nil {
		g.genElseIf(stmt.Alternative)
	}
	g.ws()
	if stmt.Default != nil {
		g.genElse(stmt.Default)
	}
}

func (g *Generator) genElseIf(stmt *ast.IfStatement) {
	g.w(" else if ")
	g.genExpr(stmt.Cond)
	g.ws()
	g.sym('{')
	g.newlTab()
	g.genStmtList(stmt.Stmts)
	if stmt.Alternative != nil {
		g.genElseIf(stmt.Alternative)
	}
	g.sym('}')
}

func (g *Generator) genElse(stmt *ast.ElseStatement) {
	g.w(" else {")
	g.newlTab()
	g.genStmtList(stmt.Stmts)
	g.sym('}')
}

func (g *Generator) genLoop(loop *ast.LoopStatement) {
	g.w("for ")
	g.genExpr(loop.Cond)
	g.ws()
	g.sym('{')
	g.newlTab()
	g.genStmtList(loop.Stmts)
	g.sym('}')
}

func (g *Generator) genReassignment(reas *ast.ReassignmentStatement) {
	g.w(reas.Tok.Literal)
	g.ws()
	g.sym('=')
	g.ws()
	g.genExpr(reas.NewValue)
}

func (g *Generator) genReturn(ret *ast.ReturnStatement) {
	g.w("\treturn ")
	g.genExprList(ret.ReturnValues)
}

func (g *Generator) genSubseq(subseq *ast.SubsequentVariableDeclarationStatement) {
	namesLen := len(subseq.Names)
	for i, v := range subseq.Names {
		g.genExpr(v)
		g.commaif(shouldPutComma(i, namesLen))
	}
	g.ws()
	g.sym(':')
	g.sym('=')
	g.ws()
	g.genExprList(subseq.Values)
}

func (g *Generator) genDatatypeLit(lit *ast.DatatypeLiteral) {
	g.w(lit.Tok.Literal)
	g.ws()
	g.sym('{')
	lenFields := len(lit.Fields)
	for i, v := range lit.Fields {
		g.genExpr(v.Value)
		g.commaif(shouldPutComma(i, lenFields))
	}
	g.sym('}')
}

func (g *Generator) genIdent(ident *ast.Identifier) {
	g.w(ident.Tok.Literal)
}

func (g *Generator) genBreak() {
	g.w("break")
}

func (g *Generator) genContinue() {
	g.w("continue")
}

func (g *Generator) genDatatypeDecl(decl *ast.DatatypeDeclaration) {
	g.w("type ")
	g.w(decl.Name.Tok.Literal)
	g.w(" struct {")
	g.newlTab()
	for _, v := range decl.Fields {
		g.w(v.Ident.String())
		g.w(v.Tok.Literal)
		g.newline()
	}
	g.sym('}')
}

func (g *Generator) genListDecl(decl *ast.ListVariableDeclarationStatement) {
	g.w("var ")
	g.w(decl.Name.String())
	g.ws()
	g.brackets()
	g.w(decl.Tok.Literal)
	g.ws()
	g.sym('=')
	g.ws()
	g.genListLit(decl.List, decl.Typ.Literal)
}
