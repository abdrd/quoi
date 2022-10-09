package parser

import (
	"strconv"
)

// little utility functions to convert strings to some literal types (integers, booleans, etc.)

// convert p.tok.Literal to int64.
// if we encounter an error (that's very unlikely), we append push an error to parser.
func atoi(p *Parser) int64 {
	n, err := strconv.ParseInt(p.tok.Literal, 10, 64)
	if err != nil {
		p.errorf(p.tok.Line, p.tok.Col, "invalid integer: unable to convert '%s' to an integer", p.tok.Literal)
	}
	return n
}

// ascii to bool
func atob(p *Parser) bool {
	var lit bool
	switch p.tok.Literal {
	case "true":
		lit = true
	case "false":
		lit = false
	default:
		p.errorf(p.tok.Line, p.tok.Col, "invalid boolean: unable to convert '%s' to a boolean", p.tok.Literal)
	}
	return lit
}
