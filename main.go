package main

import (
	"fmt"
	"log"
	"os"
	"quoi/cmd"
)

func printUsage() { fmt.Println("usage") }

func readFile(filename string) ([]byte, error) {
	bx, err := os.ReadFile(filename)
	if err != nil {
		return bx, err
	}
	return bx, err
}

func main() {
	// TODO emit flags
	switch len(os.Args) {
	case 1:
		printUsage()
	case 2:
		filename := os.Args[1]
		bx, err := readFile(filename)
		if err != nil {
			log.Fatalf("read %s: %s\n", filename, err.Error())
		}
		err = cmd.Run(filename, string(bx))
		if err != nil {
			log.Fatalf("run: %s\n", err.Error())
		}
	}
}
