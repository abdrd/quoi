package generator

import (
	"fmt"
	"quoi/ast"
	"strings"
)

// Quoi -> Go
type Generator struct {
	program       *ast.Program
	imports, rest strings.Builder
}

func New(program *ast.Program) *Generator {
	g := &Generator{program: program}
	g.imports.WriteString("import(\n\t")
	return g
}

func (g *Generator) ws() {
	g.rest.WriteByte(' ')
}

func (g *Generator) newline() {
	g.rest.WriteByte('\n')
}

func (g *Generator) newlTab() {
	g.rest.WriteString("\n\t")
}

func (g *Generator) commaif(cond bool) {
	if cond {
		g.rest.WriteString(", ")
	}
}

func (g *Generator) brackets() {
	g.rest.WriteString("[]")
}

func (g *Generator) sym(char byte) {
	g.rest.WriteByte(char)
}

func shouldPutComma(idx, length int) bool {
	return idx < length && idx != length-1
}

func (g *Generator) genImport(pkg string) {
	g.imports.WriteString(fmt.Sprintf("\"%s\"\n", pkg))
}

func (g *Generator) genParam(param ast.FunctionParameter) {
	g.rest.WriteString(param.Name.String())
	g.ws()
	if param.IsList {
		g.brackets()
		g.rest.WriteString(param.TypeOfList.Literal)
	} else {
		g.rest.WriteString(param.Tok.Literal)
	}
}

func (g *Generator) genFuncDecl(fn *ast.FunctionDeclarationStatement) {
	g.rest.WriteString("func ")
	g.ws()
	g.rest.WriteString(fn.Name.String())
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
				g.rest.WriteString(v.TypeOfList.Literal)
			} else {
				g.rest.WriteString(v.Tok.Literal)
			}
			g.commaif(shouldPutComma(i, len(fn.Params)))
		}
		if needParens {
			g.sym(')')
		}
	}
	g.sym('{')
	g.newline()
	for _, v := range fn.Stmts {
		g.genStatement(v)
	}
}

func (g *Generator) genStatement(s ast.Statement) {
	switch s := s.(type) {
	case *ast.VariableDeclarationStatement:
		g.genVarDecl(s)
	case *ast.FunctionCall:
		g.genFunCall(s)
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
		/* ??
		case *ast.BlockStatement:
			g.genBlock(s)

		*/
	}
}

func (g *Generator) genVarDecl(decl *ast.VariableDeclarationStatement) {
	g.rest.WriteString("var ")
	g.rest.WriteString(decl.Ident.String())
	g.ws()
	g.rest.WriteString(decl.Tok.Literal)
	g.ws()
	g.sym('=')
	g.ws()
	g.genExpr(decl.Value)
	g.newline()
}

func (g *Generator) genFunCall(call *ast.FunctionCall)                            {}
func (g *Generator) genIfStmt(stmt *ast.IfStatement)                              {}
func (g *Generator) genLoop(loop *ast.LoopStatement)                              {}
func (g *Generator) genReassignment(reas *ast.ReassignmentStatement)              {}
func (g *Generator) genReturn(ret *ast.ReturnStatement)                           {}
func (g *Generator) genSubseq(subseq *ast.SubsequentVariableDeclarationStatement) {}

func (g *Generator) genExpr(expr ast.Expr) {
	switch expr := expr.(type) {
	case *ast.BoolLiteral:
		g.rest.WriteString(expr.String())
	case *ast.DatatypeLiteral:
		g.genDatatypeLit(expr)
	case *ast.FunctionCall:
		g.genFunCall(expr)
	case *ast.Identifier:
		g.genIdent(expr)
	}

}

func (g *Generator) genDatatypeLit(lit *ast.DatatypeLiteral) {

}

func (g *Generator) genIdent(ident *ast.Identifier) {

}
