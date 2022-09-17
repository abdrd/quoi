package token

type Type int

const (
	EOF Type = iota
	ILLEGAL
	WHITESPACE
	IDENT
	INT
	STRING
	BOOL
	PRINT
	PRINTF
	DATATYPE
	FUN
	BLOCK
	END
	IF
	ELSEIF
	ELSE
	LOOP
	RETURN
	AT
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
)

func (t Type) String() string {
	tt := map[Type]string{
		EOF: "EOF", ILLEGAL: "ILLEGAL", WHITESPACE: "WHITESPACE",
		IDENT: "IDENTIFIER", INT: "INTEGER", STRING: "STRING", BOOL: "BOOLEAN",
		PRINT: "PRINT", PRINTF: "PRINTF", DATATYPE: "DATATYPE", FUN: "FUN",
		BLOCK: "BLOCK", END: "END", IF: "IF", ELSEIF: "ELSEIF", ELSE: "ELSE",
		LOOP: "LOOP", RETURN: "RETURN", AT: "AT", SEMICOLON: "SEMICOLON",
		OPENING_PAREN: "OPENING_PAREN", CLOSING_PAREN: "CLOSING_PAREN",
		BACKSLASH: "BACKSLASH", DOUBLE_QUOTE: "DOUBLE_QUOTE", UNDERSCORE: "UNDERSCORE",
		DOT: "DOT", OPENING_CURLY: "OPENING_CURLY", CLOSING_CURLY: "CLOSING_CURLY", ARROW: "ARROW",
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
