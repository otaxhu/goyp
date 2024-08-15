package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/version"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Get Go Toolchain version, using $GOROOT/VERSION file,
// where $GOROOT is the value of GOROOT env variable returned by
// call to 'go env GOROOT' command
func goToolchainVersion() (string, error) {
	b := bytes.Buffer{}
	errB := bytes.Buffer{}
	c := exec.Command("go", "env", "GOROOT")
	c.Stderr = &errB
	c.Stdout = &b

	err := c.Run()
	if x, ok := err.(*exec.ExitError); ok {
		return "", fmt.Errorf("goToolchainVersion: command 'go env GOROOT' exited with code '%d': %s", x.ExitCode(), errB.Bytes())
	} else if err != nil {
		return "", fmt.Errorf("goToolchainVersion: %w", err)
	}

	if len(errB.Bytes()) > 0 {
		return "", fmt.Errorf("goToolchainVersion: %s", errB.Bytes())
	}

	if !filepath.IsAbs(b.String()) {
		return "", errors.New("goToolchainVersion: could not determine GOROOT using 'go env GOROOT' command")
	}

	pathGOROOT := strings.TrimSpace(b.String())

	vPath := filepath.Join(pathGOROOT, "VERSION")

	vFile, err := os.Open(vPath)
	if err != nil {
		return "", err
	}
	defer vFile.Close()

	var goVersion string

	_, err = fmt.Fscanf(vFile, "%s\n", &goVersion)
	if err != nil {
		return "", err
	}

	v := strings.TrimSpace(goVersion)
	if !version.IsValid(v) {
		return "", fmt.Errorf("goToolchainVersion: version '%s' obtained from '%s' file is not a valid version", v, vPath)
	}

	return v, nil
}

var stdPackages map[string]struct{}

func initStdPackagesList() error {
	stdPackages = map[string]struct{}{}

	b := bytes.Buffer{}
	errB := bytes.Buffer{}

	c := exec.Command("go", "list", "std")
	c.Stderr = &errB
	c.Stdout = &b

	err := c.Run()
	if x, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("initStdPackagesList: command 'go list std' exited with code '%d': %s", x.ExitCode(), errB.Bytes())
	} else if err != nil {
		return err
	}

	sc := bufio.NewScanner(&b)
	for sc.Scan() {
		p := strings.TrimSpace(sc.Text())
		if p == "" {
			continue
		}
		stdPackages[p] = struct{}{}
	}
	return nil
}

func isStdPackage(importPath string) bool {
	if stdPackages == nil {
		err := initStdPackagesList()
		if err != nil {
			log.Fatal(err)
		}
	}
	_, ok := stdPackages[importPath]
	return ok
}
