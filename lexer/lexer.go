package lexer

import (
	"fmt"
	"quoi/token"
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
)

const (
	ErrUnclosedString ErrCode = iota
	ErrInvalidNegativeInteger
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
		stateLexWs:     lexWs,
		stateLexInt:    lexInt,
		stateLexString: lexString,
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
	return token.Token{
		Type:    token.WHITESPACE,
		Literal: lit,
		Line:    l.line,
		Col:     start,
	}
}

func lexInt(l *Lexer) token.Token {
	start := l.pointer
	if l.ch == '-' {
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
	return token.Token{
		Type:    token.INT,
		Literal: lit,
		Line:    l.line,
		Col:     start,
	}
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
	return token.Token{
		Type:    token.STRING,
		Literal: lit,
		Line:    l.line,
		Col:     start,
	}
}

func ignoreComment(l *Lexer) {
	for is(semicolon, l.ch) {
		l.advance()
	}
	l.state = stateStart
}

func (l *Lexer) Next() token.Token {
	if l.state == stateStart {
		if l.ch == eof {
			return token.Token{
				Type:    token.EOF,
				Literal: "<<<EOF>>>",
				Line:    l.line,
				Col:     l.col,
			}
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
		}
	}
	fn := l.lexFns[l.state]
	if fn != nil {
		return fn(l)
	}
	return token.Token{Type: token.ILLEGAL, Literal: string(l.ch)}
}
