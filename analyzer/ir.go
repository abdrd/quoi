package analyzer

import (
	"fmt"
	"strings"
)

type IRProgram struct {
	Stmts []IRStatement
}

func (i *IRProgram) Push(stmt IRStatement) {
	i.Stmts = append(i.Stmts, stmt)
}

type IRStatement interface {
	irStmt()
	String() string
}

type IRExpression interface {
	irExpr()
	String() string
}

type IRVariable struct {
	Name, Type string
	Value      IRExpression
}

type IRSubseq struct {
	Names, Types []string
	Values       []IRExpression
}

type IRFunction struct {
	Name                     string
	Takes, Returns           []string
	TakesCount, ReturnsCount int
	Block                    []IRStatement
}

type IRIf struct {
	Cond        IRExpression
	Block       []IRStatement
	Alternative *IRElseIf
	Default     *IRElse
}

type IRElseIf struct {
	Cond        IRExpression
	Block       []IRStatement
	Alternative *IRElseIf
	Default     *IRElse
}

type IRElse struct {
	Block []IRStatement
}

type IRReturn struct {
	ReturnTypes  []string
	ReturnValues []IRExpression
	ReturnCount  int
}

type IRDatatypeField struct {
	Type, Name string
}

type IRDatatype struct {
	Name       string
	FieldCount int
	Fields     []IRDatatypeField
}

type IRReassigment struct {
	Name     string
	NewValue IRExpression
}

// EXPRESSIONS

type IRVariableReference struct {
	Name, Type string
}

type IRInt struct {
	Value string // to avoid converting int to string in codegen.
}

type IRString struct {
	Value string
}

type IRBoolean struct {
	Value string
}

type IRList struct {
	Type   string
	Length int
	Value  []IRExpression
}

type IRFunctionCall struct {
	Name                     string
	Takes                    []IRExpression
	Returns                  []string
	TakesCount, ReturnsCount int
}

type IRFunctionCallFromNamespace struct {
	Namespace string
	IRFunctionCall
}

type IRDatatypeLiteral struct {
	Name            string
	FieldsAndValues map[string]IRExpression
}

type IRPrefExpr struct {
	Operator string
	Operands []IRExpression
}

type IRNot struct {
	Expr IRExpression
}

type IRIndex struct {
	Expr IRExpression
}

type IRBlock struct {
	Stmts []IRStatement
}

type IRLoop struct {
	Cond  IRExpression
	Stmts []IRStatement
}

/* ********** IR STATEMENTS ***************** */
func (IRVariable) irStmt()                  {}
func (IRSubseq) irStmt()                    {}
func (IRFunction) irStmt()                  {}
func (IRIf) irStmt()                        {}
func (IRElseIf) irStmt()                    {}
func (IRElse) irStmt()                      {}
func (IRReturn) irStmt()                    {}
func (IRFunctionCallFromNamespace) irStmt() {}
func (IRFunctionCall) irStmt()              {}
func (IRDatatype) irStmt()                  {}
func (IRPrefExpr) irStmt()                  {}
func (IRReassigment) irStmt()               {}
func (IRBlock) irStmt()                     {}
func (IRLoop) irStmt()                      {}

/* ********** IR EXPRESSIONS **************** */
func (IRVariableReference) irExpr()         {}
func (IRInt) irExpr()                       {}
func (IRString) irExpr()                    {}
func (IRBoolean) irExpr()                   {}
func (IRList) irExpr()                      {}
func (IRFunctionCall) irExpr()              {}
func (IRFunctionCallFromNamespace) irExpr() {}
func (IRPrefExpr) irExpr()                  {}
func (IRNot) irExpr()                       {}
func (IRIndex) irExpr()                     {}
func (IRDatatypeLiteral) irExpr()           {}

/* ************ */
// ADD String methods on IR nodes for debugging.

func (p *IRProgram) String() string {
	res := "PROGRAM!(\n"
	for _, v := range p.Stmts {
		res += v.String() + "\t\n"
	}
	res += ")"
	return res
}

func (i *IRVariable) String() string {
	if i == nil {
		return "<nil_var>"
	}
	return fmt.Sprintf("var!(name:%s type:%s value:%s)", i.Name, i.Type, i.Value)
}

func (s *IRSubseq) String() string {
	if s == nil {
		return "<nil_subseq>"
	}
	var res string = "subseq!("
	for i := 0; i < len(s.Names); i++ {
		res += fmt.Sprintf("name:%s type:%s", s.Names[i], s.Types[i])
		if i != len(s.Names)-1 {
			res += " "
		}
	}
	res += " value:"
	for _, v := range s.Values {
		res += v.String()
	}
	res += ")"
	return res
}

func (f *IRFunction) String() string {
	if f == nil {
		return "<nil_fun>"
	}
	var res = "fun!("
	res += fmt.Sprintf("name:%s takes:#%d[", f.Name, f.TakesCount)
	for i, v := range f.Takes {
		res += fmt.Sprintf("%s ", v)
		if !(i == f.TakesCount-1) {
			res += " "
		}
	}
	res += fmt.Sprintf("] returns:#%d[", f.ReturnsCount)
	for i, v := range f.Returns {
		res += v
		if !(i == f.ReturnsCount-1) {
			res += " "
		}
	}
	res += "]\n{"
	for _, v := range f.Block {
		res += fmt.Sprintf("\t%s", v)
	}
	res += "\n}"
	return res
}

func (i *IRIf) String() string {
	if i == nil {
		return "<nil_if>"
	}
	res := "if cond:"
	res += i.Cond.String()
	res += " \n{"
	for _, v := range i.Block {
		res += fmt.Sprintf("\t%s", v)
	}
	res += "} "
	if i.Alternative != nil {
		res += (&IRIf{Cond: i.Alternative.Cond, Block: i.Alternative.Block,
			Alternative: i.Alternative.Alternative, Default: i.Alternative.Default}).String()
	}
	if i.Default != nil {
		res += i.Default.String()
	}
	return res
}

func (e *IRElse) String() string {
	if e == nil {
		return "<nil_else>"
	}
	res := " else {"
	for _, v := range e.Block {
		res += fmt.Sprintf("\t%s", v)
	}
	res += "}"
	return res
}

func (*IRElseIf) String() string {
	return "ELSEIF STRING"
}

func (r *IRReturn) String() string {
	if r == nil {
		return "<nil_return>"
	}
	res := fmt.Sprintf("return!(types#%d:[", r.ReturnCount)
	for i, v := range r.ReturnTypes {
		res += v
		if i != r.ReturnCount-1 {
			res += " "
		}
	}
	res += "] returns:["
	for i, v := range r.ReturnValues {
		res += v.String()
		if i != r.ReturnCount-1 {
			res += " "
		}
	}
	res += "])"
	return res
}

func (f *IRFunctionCallFromNamespace) String() string {
	if f == nil {
		return "<nil_fcfn>"
	}
	res := fmt.Sprintf("fcfn!(name:%s", f.Namespace+"::"+f.Name)
	res += fmt.Sprintf(" takes:#%d[", f.TakesCount)
	for i, v := range f.Takes {
		res += v.String()
		if i != f.TakesCount-1 {
			res += " "
		}
	}
	res += fmt.Sprintf("] returns:#%d[", f.ReturnsCount)
	for i, v := range f.Returns {
		res += v
		if i != f.ReturnsCount-1 {
			res += " "
		}
	}
	res += "])"
	return res
}

func (f *IRFunctionCall) String() string {
	if f == nil {
		return "<nil_fc>"
	}
	// silly
	fcfn := &IRFunctionCallFromNamespace{Namespace: "", IRFunctionCall: *f}
	return strings.Replace(fcfn.String(), "fcfn!(", "fc!(", 1)
}

func (d *IRDatatype) String() string {
	if d == nil {
		return "<nil_datatype>"
	}
	res := fmt.Sprintf("datatype!(name:%s fields:#%d[ ", d.Name, d.FieldCount)
	for i, v := range d.Fields {
		res += fmt.Sprintf("name:%s type:%s", v.Name, v.Type)
		if i != d.FieldCount-1 {
			res += " "
		}
	}
	res += "])"
	return res
}

func (p *IRPrefExpr) String() string {
	if p == nil {
		return "<nil_prefexpr>"
	}
	res := "prefexpr!("
	res += p.Operator + " "
	for i, v := range p.Operands {
		res += v.String()
		if i != len(p.Operands)-1 {
			res += " "
		}
	}
	res += ")"
	return res
}

func (r *IRReassigment) String() string {
	if r == nil {
		return "<nil_reas>"
	}
	res := fmt.Sprintf("reas!(name:%s newval:%s)", r.Name, r.NewValue)
	return res
}

func (b *IRBlock) String() string {
	if b == nil {
		return "<nil_bloc>"
	}
	res := "bloc!(\n"
	for i, v := range b.Stmts {
		res += fmt.Sprintf("\t%s", v)
		if i != len(b.Stmts)-1 {
			res += "\n"
		}
	}
	res += ")"
	return res
}

func (l *IRLoop) String() string {
	if l == nil {
		return "<nil_loop>"
	}
	res := fmt.Sprintf("loop!(cond:%s ", l.Cond)
	for i, v := range l.Stmts {
		res += fmt.Sprintf("\t%s", v)
		if i != len(l.Stmts)-1 {
			res += "\n"
		}
	}
	res += ")"
	return res
}

func (v *IRVariableReference) String() string {
	if v == nil {
		return "<nil_varref>"
	}
	return fmt.Sprintf("varref!(name:%s type:%s)", v.Name, v.Type)
}

func (i *IRInt) String() string {
	if i == nil {
		return "<nil_int>"
	}
	return i.Value
}

func (b *IRBoolean) String() string {
	if b == nil {
		return "<nil_bool>"
	}
	return b.Value
}

func (s *IRString) String() string {
	if s == nil {
		return "<nil_str>"
	}
	return fmt.Sprintf("\"%s\"", s.Value)
}

func (l *IRList) String() string {
	if l == nil {
		return "<nil_list>"
	}
	res := fmt.Sprintf("list!(type:%s elems:#%d[", l.Type, l.Length)
	for i, v := range l.Value {
		res += v.String()
		if i != l.Length-1 {
			res += " "
		}
	}
	res += "])"
	return res
}

func (n *IRNot) String() string {
	if n == nil {
		return "<nil_not>"
	}
	return fmt.Sprintf("not!(%s)", n.Expr)
}

func (i *IRIndex) String() string {
	if i == nil {
		return "<nil_index>"
	}
	return fmt.Sprintf("index!(%s)", i.Expr)
}

func (d *IRDatatypeLiteral) String() string {
	if d == nil {
		return "<nil_dtlit>"
	}
	res := fmt.Sprintf("dtlit!(name:%s fields:{ ", d.Name)
	for k, v := range d.FieldsAndValues {
		res += fmt.Sprintf("%s=%s ", k, v)
	}
	res += "})"
	return res
}
