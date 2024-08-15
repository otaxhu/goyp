/*
   Copyright 2024 Oscar Pernia

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

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

// Get $GOMODCACHE path, using 'go env GOMODCACHE' command
func goModCache() (string, error) {
	b := bytes.Buffer{}
	errB := bytes.Buffer{}
	c := exec.Command("go", "env", "GOMODCACHE")

	c.Stderr = &errB
	c.Stdout = &b

	err := c.Run()
	if x, ok := err.(*exec.ExitError); ok {
		return "", fmt.Errorf("goModCache: command 'go env GOMODCACHE' exited with code '%d': %s", x.ExitCode(), errB.Bytes())
	} else if err != nil {
		return "", err
	}

	if errB.Len() > 0 {
		return "", fmt.Errorf("goModCache: %s", errB.Bytes())
	}

	pathGOMODCACHE := strings.TrimSpace(b.String())

	if !filepath.IsAbs(pathGOMODCACHE) {
		return "", fmt.Errorf("goModCache: could not determine GOMODCACHE using 'go env GOMODCACHE' command")
	}

	return pathGOMODCACHE, nil
}
