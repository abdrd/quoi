package analyzer

type IRProgram struct {
	IRStatements []IRStatement
}

func (i *IRProgram) Push(stmt IRStatement) {
	i.IRStatements = append(i.IRStatements, stmt)
}

type IRStatement interface {
	irStmt()
}

type IRExpression interface {
	irExpr()
}

type IRVariable struct {
	Name, Type string
	Value      IRExpression
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

// EXPRESSIONS

type IRVariableReference struct {
	Name, Type string
	Value      IRExpression
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
	Takes, Returns           []string
	TakesCount, ReturnsCount int
}

type IRFunctionCallFromNamespace struct {
	Namespace string
	IRFunctionCall
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

/* ********** IR STATEMENTS ***************** */
func (IRVariable) irStmt()                  {}
func (IRFunction) irStmt()                  {}
func (IRIf) irStmt()                        {}
func (IRElseIf) irStmt()                    {}
func (IRElse) irStmt()                      {}
func (IRReturn) irStmt()                    {}
func (IRFunctionCallFromNamespace) irStmt() {}
func (IRFunctionCall) irStmt()              {}
func (IRDatatype) irStmt()                  {}
func (IRPrefExpr) irStmt()                  {}

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
