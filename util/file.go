package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

// CopyFile copies the data from `src` to `dst`.
func CopyFile(src string, dest string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return fmt.Errorf("CopyFile: ReadFile: %v", err)
	}

	if err := ioutil.WriteFile(dest, data, 0600); err != nil {
		return fmt.Errorf("CopyFile: WriteFile: %v", err)
	}

	return nil
}

// Edit opens `filename` in $EDITOR (assuming vim as default)
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

// GuardExists returns an descriptive error for the user if `path` exists already.
func GuardExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("Guard: Destination file already exists (%s). Use --force to ignore this", path)
	}

	return nil
}
