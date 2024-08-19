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

// 'goyp build' command
package build

import (
	"flag"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/otaxhu/goyp/internal/compile"
	"github.com/otaxhu/goyp/internal/mod"
	"golang.org/x/mod/modfile"
	modzip "golang.org/x/mod/zip"
)

// args after command
func Build(args []string) {
	set := flag.NewFlagSet("build", flag.ExitOnError)
	outputPointer := set.String("o", "", "output file path")

	targets := map[string]struct{}{}
	envTargets := os.Getenv("GOYP_TARGETS")
	for _, t := range strings.Split(envTargets, ",") {
		b, a, _ := strings.Cut(t, "_")
		if b == "" || a == "" {
			log.Fatalf("build: invalid target '%s' found in GOYP_TARGETS environment variable", t)
		}
		targets[t] = struct{}{}
	}

	if len(targets) == 0 {
		GOOS := os.Getenv("GOOS")
		if GOOS == "" {
			GOOS = runtime.GOOS
		}
		GOARCH := os.Getenv("GOARCH")
		if GOARCH == "" {
			GOARCH = runtime.GOARCH
		}
		targets[GOOS+"_"+GOARCH] = struct{}{}
	}

	set.Parse(args)

	// zip file or executable
	var outputPath string
	var err error
	var pathPackage string

	if set.NArg() == 0 {
		pathPackage, err = filepath.Abs(".")
		if err != nil {
			log.Fatal(err)
		}
	} else if set.NArg() == 1 {
		pathPackage, err = filepath.Abs(set.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatalf("invalid number of packages, expected 0 or 1, got %d", set.NArg())
	}

	goModPath := filepath.Join(pathPackage, "go.mod")

	goModBytes, err := os.ReadFile(goModPath)
	if err != nil {
		log.Fatal(err)
	}

	modInfo, err := modfile.Parse("go.mod", goModBytes, nil)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: pkgInfo provides import information, use it to compile dependency packages
	// before compiling it.
	pkgInfo, err := build.ImportDir(pathPackage, 0)
	if err != nil {
		log.Fatal(err)
	}

	if pkgInfo.Name == "main" {
		log.Fatal("TODO: compile 'main' package unimplemented")
	}

	if *outputPointer == "" {
		outputPath, err = filepath.Abs(filepath.Base(modInfo.Module.Mod.Path) + ".zip")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		outputPath, err = filepath.Abs(*outputPointer)
		if err != nil {
			log.Fatal(err)
		}
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()

	tempDir, err := os.MkdirTemp("", "goyp-build-*")
	if err != nil {
		log.Fatal(err)
	}

	// TODO: search in GOMODCACHE for resolving non-std modules

	// TODO: std packages list available in 'go list std' command

	objPath, err := compile.CompilePackage(tempDir, pathPackage, modInfo.Module.Mod.Path, "TODO", "TODO")
	if err != nil {
		log.Fatal(err)
	}

	// TODO: use a better semver, maybe from git tags
	modInfo.Module.Mod.Version = "v0.0.0"

	// TODO: After compiling all of the packages, call this function with a slice
	// with all of the files
	err = modzip.Create(outputFile, modInfo.Module.Mod, []modzip.File{
		&mod.ModuleFile{
			Root:     filepath.Dir(objPath),
			Filename: filepath.Base(objPath),
		},
		&mod.ModuleFile{
			Root:     pathPackage,
			Filename: "go.mod",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

}
