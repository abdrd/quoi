package lexer

import (
	"fmt"
	"quoi/token"
	"strings"
	"unicode"
)

const eof = rune(-1)

type (
	state   int
	ErrCode int
	// for little syntax errors to pass to the parser
	Err struct {
		ErrCode      ErrCode
		Msg          string
		Column, Line int
	}
)

const (
	stateStart state = iota

	stateLexWs
	stateLexInt
	stateLexString
	stateLexIdentKw
	stateLexSymbol // lexing symbols like {, ., ), etc.
)

const (
	ErrUnclosedString ErrCode = iota
	ErrInvalidNegativeInteger
	ErrUnknownSymbol
)

type lexFn func(*Lexer) token.Token

type Lexer struct {
	src           []rune // source code
	lenSrc        uint   // source string length
	pointer       uint   // index of the current character
	line, col     uint
	ch            rune // current character
	hasReachedEOF bool
	state         state
	lexFns        map[state]lexFn // which function to call when in state
	Errs          []Err
}

func New(input string) *Lexer {
	var lexFns = map[state]lexFn{
		stateLexWs:      lexWs,
		stateLexInt:     lexInt,
		stateLexString:  lexString,
		stateLexIdentKw: lexIdentOrKw,
		stateLexSymbol:  lexSymbol,
	}
	if len(input) == 0 {
		panic("lexer.New: empty input string")
	}
	l := &Lexer{
		src:     []rune(input),
		pointer: 0,
		col:     0,
		state:   stateStart,
		lexFns:  lexFns,
	}
	l.lenSrc = uint(len(l.src))
	l.ch = l.src[l.pointer]
	l.hasReachedEOF = l.pointer == l.lenSrc-1
	l.line = 1
	return l
}

func (l *Lexer) errorf(errCode ErrCode, col, line int, formatMsg string, elems ...interface{}) {
	l.Errs = append(l.Errs, Err{
		ErrCode: errCode,
		Msg:     fmt.Sprintf(formatMsg, elems...),
		Column:  col,
		Line:    line,
	})
}

func (l *Lexer) peek() rune {
	if l.hasReachedEOF {
		return eof
	}
	return l.src[l.pointer+1]
}

func (l *Lexer) advance() {
	if l.hasReachedEOF {
		l.ch = eof
		return
	}
	l.pointer++
	l.col++
	l.ch = l.src[l.pointer]
	l.hasReachedEOF = l.pointer+1 == l.lenSrc
	if l.ch == '\n' {
		l.line++
		l.col = 0
	}
}

func canBeAnIdentifierName(ch rune) bool {
	return ch != '@' && (((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')) || ch == '_')
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isWhitespace(ch rune) bool {
	return unicode.IsSpace(ch)
}

func isSymbol(ch rune) bool {
	str := string(ch)
	symbols := ".={}()-"
	return strings.Contains(symbols, str)
}

type char byte

const (
	doubleQuote char = '"'
	semicolon   char = ';'
)

func is(char char, ch rune) bool {
	return byte(ch) == byte(char)
}

func lexWs(l *Lexer) token.Token {
	start := l.pointer
	for isWhitespace(l.ch) {
		l.advance()
		if l.hasReachedEOF {
			break
		}
	}
	end := l.pointer
	if l.hasReachedEOF {
		end++
	}
	lit := string(l.src[start:end])
	// set state to stateStart, to determine the next lexFn in *Lexer.Next.
	l.state = stateStart
	return token.New(token.WHITESPACE, lit, l.line, start)
}

func lexInt(l *Lexer) token.Token {
	start := l.pointer
	if l.ch == '-' {
		if p := l.peek(); p == '>' {
			// this is an arrow symbol.
			// we come across this here, because
			// in Next, the 'else if' clause that checks if it is a
			// integer comes before the clause that checks whether this
			// is a symbol. and both integers, and the arrow symbol has minus
			// at the beginning.
			l.state = stateLexSymbol
			return l.Next()
		}
		l.advance()
	}
	if !(isDigit(l.ch)) {
		l.errorf(ErrInvalidNegativeInteger, int(l.col), int(l.line), "no value after minus")
	}
	for isDigit(l.ch) {
		if l.hasReachedEOF {
			break
		}
		l.advance()
	}
	end := l.pointer
	if l.hasReachedEOF {
		end++
	}
	lit := string(l.src[start:end])
	l.state = stateStart
	return token.New(token.INT, lit, l.line, start)
}

func lexString(l *Lexer) token.Token {
	// eat '"'
	l.advance()
	start := l.pointer
	for !(is(doubleQuote, l.ch)) {
		if l.hasReachedEOF {
			l.errorf(ErrUnclosedString, int(l.pointer), int(l.line), "unexpected end-of-file: unclosed string")
		}
		l.advance()
	}
	end := l.pointer
	if l.hasReachedEOF {
		end++
		// last character of the file is "
		// don't pick it up
		if l.ch == '"' {
			end--
		}
	}
	lit := string(l.src[start:end])
	l.state = stateStart
	return token.New(token.STRING, lit, l.line, start)
}

func ignoreComment(l *Lexer) {
	for is(semicolon, l.ch) {
		l.advance()
	}
	l.state = stateStart
}

func lexIdentOrKw(l *Lexer) token.Token {
	var kw = map[string]token.Type{
		"print": token.PRINT, "printf": token.PRINTF, "datatype": token.DATATYPE, "fun": token.FUN,
		"int": token.INT, "string": token.STRING, "bool": token.BOOL, "block": token.BLOCK,
		"end": token.END, "if": token.IF, "elseif": token.ELSEIF, "else": token.ELSE,
		"loop": token.LOOP, "return": token.RETURN,
	}
	start := l.pointer
	for canBeAnIdentifierName(l.ch) || isDigit(l.ch) {
		l.advance()
	}
	end := l.pointer
	if l.hasReachedEOF {
		end++
		if l.ch != eof {
			end--
		}
	}
	lit := string(l.src[start:end])
	keyword, isKw := kw[lit]
	tok := token.New(token.IDENT, lit, l.line, start)
	if isKw {
		tok.Type = keyword
	}
	l.state = stateStart
	return tok
}

func lexSymbol(l *Lexer) token.Token {
	var symbols = map[byte]token.Type{
		'.': token.DOT,
		'=': token.EQUAL,
		'{': token.OPENING_CURLY,
		'}': token.CLOSING_CURLY,
		'(': token.OPENING_PAREN,
		')': token.CLOSING_PAREN,
	}
	start := l.col
	if l.ch == '-' {
		oldLit := string(l.ch)
		l.advance()
		if l.ch == eof {
			l.errorf(ErrUnknownSymbol, int(start), int(l.line), "unknown symbol '%s'. did you mean '%s'?", string(l.ch), "->")
			l.state = stateStart
			l.advance() // just in case
			return token.New(token.ILLEGAL, oldLit, l.line, start)
		}
		if l.ch == '>' {
			l.advance()
			l.state = stateStart
			return token.New(token.ARROW, "->", l.line, start)
		}
	}
	tok, found := symbols[byte(l.ch)]
	lit := string(l.ch)
	l.advance()
	l.state = stateStart
	if !(found) {
		l.errorf(ErrUnknownSymbol, int(start), int(l.line), "unknown symbol '%s'", lit)
		return token.New(token.ILLEGAL, lit, l.line, l.col)
	}
	return token.New(tok, lit, l.line, l.col)
}

func (l *Lexer) Next() token.Token {
	if l.state == stateStart {
		if l.ch == eof {
			return token.New(token.EOF, "<<<EOF>>>", l.line, l.col)
		}
		if isWhitespace(l.ch) {
			l.state = stateLexWs
		} else if isDigit(l.ch) || l.ch == '-' {
			l.state = stateLexInt
		} else if is(doubleQuote, l.ch) {
			l.state = stateLexString
		} else if is(semicolon, l.ch) {
			ignoreComment(l)
			return l.Next()
		} else if canBeAnIdentifierName(l.ch) {
			l.state = stateLexIdentKw
		} else if isSymbol(l.ch) {
			l.state = stateLexSymbol
		}
	}
	fn := l.lexFns[l.state]
	if fn != nil {
		return fn(l)
	}
	ill := token.New(token.ILLEGAL, string(l.ch), l.line, l.col)
	l.advance()
	return ill
}
