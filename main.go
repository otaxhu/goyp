package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	modzip "golang.org/x/mod/zip"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("goyp: ")

	set := flag.NewFlagSet("goyp", flag.ExitOnError)

	set.Usage = func() {
		os.Stderr.WriteString(`Usage: goyp <command> [arguments...]

Description:
  goyp is a build system for Golang that allows you to compile your code, with support for
  Golang binary dependencies and Go Modules.

  It uses the Go toolchain ('go' command) to perform all of the low level details. So it's required
  to be installed and available in PATH environment variable.

Commands:
  build     Compile the specified Go Module, outputing a zipfile containing all of the binary code
            generated from a non-main .

  dist-lib  Decompress zipfile generated by 'build', to distribute your library through Go Modules.

  help      Prints help about the different commands.

  install   Install main package to $GOPATH/bin or $HOME/go/bin if $GOPATH is not specified

  run       Build and run the specified Go main package.

  version   Display the current version of goyp.

See 'goyp help' for more information.
`)
	}

	if len(os.Args) < 2 {
		set.Usage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "run", "version", "help", "dist-lib", "install":
		log.Fatalf("TODO: '%s' command unimplemented", command)
	case "build":
		outputPointer := set.String("o", "", "output file path")

		set.Parse(os.Args[2:])

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

		objPath, err := compilePackage(tempDir, pathPackage, modInfo.Module.Mod.Path)
		if err != nil {
			log.Fatal(err)
		}

		// TODO: use a better semver, maybe from git tags
		modInfo.Module.Mod.Version = "v0.0.0"

		// TODO: After compiling all of the packages, call this function with a slice
		// with all of the files
		err = modzip.Create(outputFile, modInfo.Module.Mod, []modzip.File{
			&moduleFile{
				root:     filepath.Dir(objPath),
				filename: filepath.Base(objPath),
			},
			&moduleFile{
				root:     pathPackage,
				filename: "go.mod",
			},
		})
		if err != nil {
			log.Fatal(err)
		}

		break

	default:
		log.Fatalf("invalid '%s' command", command)
	}
}

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
