package lexer

import (
	"fmt"
	"os"
	"testing"
)

func TestReadFile(t *testing.T) {
	fmt.Println(readFile("main.quoi"))
}

func Test1(t *testing.T) {
	// get environment variable accessible to this process, to get the source file's name to tokenize
	fileName := os.Getenv("FILE")
	if fileName == "" {
		fileName = "main.quoi"
	}
	// run the test like this:
	// FILE=<file> go test . -v -run Test1
	runTest(fileName)
}
