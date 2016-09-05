package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func CopyFile(src string, dest string) error {
	if data, err := ioutil.ReadFile(src); err != nil {
		return fmt.Errorf("CopyFile: ReadFile: %v", err)
	} else {
		if err := ioutil.WriteFile(dest, data, 0600); err != nil {
			return fmt.Errorf("CopyFile: WriteFile: %v", err)
		}
	}
	return nil
}

func Edit(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func GuardExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("Guard: Destination file already exists (%s). Use --force to ignore this", path)
	}
	return nil
}
