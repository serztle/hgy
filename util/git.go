package util

import (
	"fmt"
	"os"
	"os/exec"
)

type Git struct {
	Dir string
}

func GitNew(dir string) Git {
	return Git{Dir: dir}
}

func (g *Git) Exec(command string, args ...string) error {
	repackArgs := []string{
		"-C", g.Dir,
		fmt.Sprintf("--git-dir=%s", ".git"),
		fmt.Sprintf("--work-tree=%s", "."),
		command,
	}
	repackArgs = append(repackArgs, args...)
	if err := exec.Command("git", repackArgs...).Run(); err != nil {
		return fmt.Errorf("Git command '%s' failed in '%s' (%v)", command, g.Dir, err)
	} else {
		return nil
	}
}

func (g *Git) New(dir string) error {
	g.Dir = dir
	if info, err := os.Stat(g.Dir); err != nil {
		return err
	} else if info.IsDir() {
		return fmt.Errorf("%s is not a directory!", g.Dir)
	}
	return nil
}

func (g *Git) Fail(err error) {
	if err == nil {
		return
	}

	if errReset := g.Exec("reset"); errReset != nil {
		fmt.Printf(
			"Error: Git reset failed (%v). Something went horribly wrong!\nCause: %v. Abort.\n",
			errReset,
			err,
		)
		os.Exit(1)
	} else {
		fmt.Printf("Error: %v. Abort.\n", err)
		os.Exit(1)
	}
}

func (g *Git) Exists() bool {
	if err := g.Exec("status"); err != nil {
		return false
	} else {
		return true
	}
}

func (g *Git) Init() error {
	return g.Exec("init")
}

func (g *Git) Add(filename string) error {
	return g.Exec("add", "-A", filename)
}

func (g *Git) Remove(filename string) error {
	return g.Exec("rm", filename)
}

func (g *Git) HasChanges(cached bool) bool {
	var args []string
	if cached {
		args = []string{"--exit-code", "--cached"}
	} else {
		args = []string{"--exit-code"}
	}
	if err := g.Exec("diff", args...); err != nil {
		return true
	} else {
		return false
	}
}

func (g *Git) Rm(filename string) error {
	return g.Exec("rm", filename)
}

func (g *Git) Commit(message string) error {
	return g.Exec("commit", "-m", message)
}
