package token

type Type int

const (
	EOF Type = iota
	ILLEGAL
	WHITESPACE
	IDENT
	INT      // a literal int
	INTKW    // keyword int
	STRING   // a string literal
	STRINGKW // string keyword
	BOOL
	BOOLKW
	DATATYPE
	FUN
	BLOCK
	END
	IF
	ELSEIF
	ELSE
	LOOP
	RETURN
	SEMICOLON
	OPENING_PAREN
	CLOSING_PAREN
	BACKSLASH
	DOUBLE_QUOTE
	UNDERSCORE
	DOT
	OPENING_CURLY
	CLOSING_CURLY
	ARROW
	EQUAL
	COMMA
	DOUBLE_COLON
	ADD
	MUL
	MINUS
	DIV
	AND
	OR
	NOT
	LT
	LTE
	GT
	GTE
)

func (t Type) String() string {
	tt := map[Type]string{
		EOF: "EOF", ILLEGAL: "ILLEGAL", WHITESPACE: "WHITESPACE",
		IDENT: "IDENTIFIER", INT: "INTEGER", STRING: "STRING", BOOL: "BOOLEAN",
		DATATYPE: "DATATYPE", FUN: "FUN",
		BLOCK: "BLOCK", END: "END", IF: "IF", ELSEIF: "ELSEIF", ELSE: "ELSE",
		LOOP: "LOOP", RETURN: "RETURN", SEMICOLON: "SEMICOLON",
		OPENING_PAREN: "OPENING_PAREN", CLOSING_PAREN: "CLOSING_PAREN",
		BACKSLASH: "BACKSLASH", DOUBLE_QUOTE: "DOUBLE_QUOTE", UNDERSCORE: "UNDERSCORE",
		DOT: "DOT", OPENING_CURLY: "OPENING_CURLY", CLOSING_CURLY: "CLOSING_CURLY", ARROW: "ARROW",
		INTKW: "INT_KEYWORD", STRINGKW: "STRING_KEYWORD", BOOLKW: "BOOL_KEYWORD",
		COMMA: ",", EQUAL: "EQUAL", DOUBLE_COLON: "DOUBLE_COLON", ADD: "ADD", MUL: "MUL",
		MINUS: "MINUS", DIV: "DIV", AND: "AND", OR: "OR", NOT: "NOT", LT: "LESS_THAN", GT: "GREATER_THAN",
		LTE: "LESS_THAN_OR_EQUAL_TO", GTE: "GREATER_THAN_OR_EQUAL_TO",
	}
	return tt[t]
}

type Token struct {
	Type      Type
	Literal   string
	Line, Col uint
}

// return new token
func New(typ Type, lit string, line, col uint) Token {
	return Token{
		Type:    typ,
		Literal: lit,
		Line:    line,
		Col:     col,
	}
}

// return human friendly name for an arithmetic operator. very limited use case. used for error reporting.
func PrefixExprName(t Type) string {
	switch t {
	case ADD:
		return "+"
	case MINUS:
		return "-"
	case DIV:
		return "/"
	case MUL:
		return "*"
	case AND:
		return "and"
	case OR:
		return "or"
	case NOT:
		return "not"
	case LT:
		return "lt"
	case GT:
		return "gt"
	case LTE:
		return "lte"
	case GTE:
		return "gte"
	}
	return "UNKNOWN"
}
