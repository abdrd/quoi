package lexer

import (
	"fmt"
	"testing"
)

func TestLexWs(t *testing.T) {
	input := "\t\t\r\v\f"
	input += "     	"
	l := New(input)
	ws := l.lexWs()
	fmt.Printf("ws: %+v\n", ws)
	if ws.StartCol != 0 && ws.EndCol != 1 {
		t.Errorf("ws pos wrong: %+v\n", ws)
		return
	}
	fmt.Println("l.pos: ", l.pos)
}
