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
	OPERATOR // psedo-functions	 (other than print, and printf)
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
	COLON
	COMMA
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
		INTKW: "INT_KEYWORD", STRINGKW: "STRING_KEYWORD", BOOLKW: "BOOL_KEYWORD", OPERATOR: "OPERATOR",
		COLON: "COLON", COMMA: ",", EQUAL: "EQUAL",
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
