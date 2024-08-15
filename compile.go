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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Compiles package to binary, outputting it to outDir root dir, respecting 'importPath' directory
// structure so that 'go tool compile' command knows where to find it.
//
// outDir is the root directory where the file is gonna be written to.
//
// NOTE on outDir:
//
// This function will use outDir as -I param, may be changed in the future.
//
// pkgPath is the directory where to find all of the .go source files from the package.
//
// importPath is the module-aware path to find this package, used to create directory structure
// inside of outDir.
//
// Returns path to object file generated.
func compilePackage(outDir, pkgPath, importPath string) (objPath string, err error) {
	entries, err := os.ReadDir(pkgPath)
	if err != nil {
		return
	}

	var goFiles []string

	for _, e := range entries {
		// TODO: Prototyping goyp, in the future it will compile .s, .S, .c, .h and more files.
		// for now, it only compiles not *_test.go ended go files.
		if e.IsDir() || strings.HasSuffix(e.Name(), "_test.go") || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}

		goFiles = append(goFiles, filepath.Join(pkgPath, e.Name()))
	}

	if len(goFiles) == 0 {
		err = fmt.Errorf("no valid go files inside of '%s' directory", pkgPath)
		return
	}

	err = os.MkdirAll(filepath.Join(outDir, filepath.Dir(importPath)), 0755)
	if err != nil {
		return
	}

	objPath = filepath.Join(outDir, importPath+".a")

	toolCompileArgs := []string{
		"tool", "compile", "-pack",
		"-I", outDir, // Include directory, tells the compiler where to find dependency packages
		"-p", importPath, // Import path, how other packages can call code from this package
		"-o", objPath, // Output path
	}

	toolCompileArgs = append(toolCompileArgs, goFiles...)

	c := exec.Command("go", toolCompileArgs...)

	c.Stderr = os.Stderr
	c.Stdout = os.Stdout

	err = c.Run()
	if err != nil {
		log.Println("go tool compile error!!!")
		return
	}

	return
}
