package lexer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"quoi/token"
	"strings"
	"unicode"
)

const eof = -1

type Position struct {
	Line, Column uint
}

type Lexer struct {
	buffer bytes.Buffer
	pos    Position
}

func New(input string) *Lexer {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				if errors.Is(e, bytes.ErrTooLarge) {
					fmt.Println("error: byte buffer became too large")
					fmt.Println("aborting...")
					os.Exit(1)
				}
			}
		}
	}()
	l := &Lexer{}
	l.buffer = bytes.Buffer{}
	l.buffer.WriteString(input)
	return l
}

func (l *Lexer) resetPos() {
	l.pos.Line++
	l.pos.Column = 0
}

func (l *Lexer) isSpace(c rune) bool {
	if c == '\n' {
		return false
	}
	return unicode.IsSpace(c)
}

func (l *Lexer) isDigit(c rune) bool {
	return unicode.IsDigit(c)
}

func (l *Lexer) makeToken(typ token.Type, startPos uint, lit string) token.Token {
	t := token.Token{
		Type:     typ,
		StartCol: startPos,
		EndCol:   l.pos.Column,
		Line:     l.pos.Line,
		Literal:  lit,
	}
	if typ == token.WHITESPACE {
		t.EndCol = t.StartCol + 1
	}
	return t
}

func (l *Lexer) readRune() (rune, error) {
	c, _, err := l.buffer.ReadRune()
	if c == '\n' {
		l.resetPos()
	}
	l.pos.Column++
	return c, err
}

func (l *Lexer) peekRune() rune {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("error: peekRune: unsuccessful read operation")
			fmt.Println("aborting...")
			os.Exit(1)
		}
	}()
	c, _, err := l.buffer.ReadRune()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return eof
		}
	}
	err = l.buffer.UnreadRune()
	if err != nil {
		panic(err)
	}
	return c
}

func (l *Lexer) NextToken() token.Token {
	c := l.peekRune()
	if c == eof {
		return l.makeToken(token.EOF, l.pos.Column, "<<<EOF>>>")
	}
	if c == '\n' {
		t := l.makeToken(token.LF, l.pos.Column, "<lf>")
		l.readRune()
		return t
	}
	if c == '\r' {
		start := l.pos.Column
		if p := l.peekRune(); p == '\n' {
			l.readRune()
			t := l.makeToken(token.CRLF, start, "<crlf>")
			l.readRune()
			return t
		}
	}
	if l.isSpace(c) {
		return l.lexWs()
	} else if l.isDigit(c) {
		return l.lexInt()
	}
	return token.Token{}
}

// scan one-length whitespace, eat out the rest
// (returned whitespace token will have a length of 1)
func (l *Lexer) lexWs() token.Token {
	startPos := l.pos.Column
	c, err := l.readRune()
	if err != nil {
		// it is impossible that c is EOF
		panic(err)
	}
	lit := string(c)
	switch lit {
	case "\t":
		lit = "<escape-seq> (t)"
	case "\v":
		lit = "<escape-seq> (v)"
	case "\f":
		lit = "<escape-seq> (f)"
	case "\r":
		lit = "<escape-seq> (r)"
	}
	for {
		p := l.peekRune()
		if !(l.isSpace(p)) {
			break
		}
		_, err := l.readRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
	}
	t := l.makeToken(token.WHITESPACE, startPos, lit)
	return t
}

func (l *Lexer) lexInt() token.Token {
	startPos := l.pos.Column
	var lit strings.Builder
	c, _ := l.readRune()
	lit.WriteRune(c)
	for {
		p := l.peekRune()
		if !(l.isDigit(p)) {
			break
		}
		c, err := l.readRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		lit.WriteRune(c)
	}
	t := l.makeToken(token.INT, startPos, lit.String())
	return t
}
