// compile and run
package cmd

import (
	"io"
	"os"
	"os/exec"
	"syscall"
)

const (
	_FILE_NAME = "main.go"
)

func saveFile(source, fname string) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, source)
	return err
}

func deleteFile(fname string) {
	if err := os.Remove(fname); err != nil {
		panic(err)
	}
}

func goCmd(cmd string) {
	bin, err := exec.LookPath("go")
	if err != nil {
		panic("quoi: `go` not found")
	}
	args := []string{"go", cmd, _FILE_NAME}
	env := os.Environ()
	err = syscall.Exec(bin, args, env)
	if err != nil {
		panic("quoi: exec: " + err.Error())
	}
}

func RunProgram(source string) {
	if err := saveFile(source, _FILE_NAME); err != nil {
		panic("quoi: save file: " + err.Error())
	}
	goCmd("run")
	deleteFile(_FILE_NAME)
}

func GetExecutable(source string) {
	if err := saveFile(source, _FILE_NAME); err != nil {
		panic("quoi: save file: " + err.Error())
	}
	goCmd("build")
	deleteFile(_FILE_NAME)
}

func GetGo(source string) {
	if err := saveFile(source, _FILE_NAME); err != nil {
		panic("quoi: save file: " + err.Error())
	}
}
