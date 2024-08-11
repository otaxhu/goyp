package main

import (
	"flag"
	"go/build"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Fatal("not enough arguments")
	}

	command := os.Args[1]

	set := flag.NewFlagSet("goyp", flag.ExitOnError)

	switch command {
	case "run":
		log.Fatal("TODO: 'run' command unimplemented")
	case "build":
		outputPointer := set.String("o", "", "output file path")
		_ = outputPointer

		set.Parse(os.Args[2:])

		var err error
		var pathPackage string
		_ = pathPackage

		if set.NArg() == 0 {
			pathPackage, err = filepath.Abs(".")
			if err != nil {
				log.Fatal(err)
			}
		} else {
			pathPackage, err = filepath.Abs(set.Arg(0))
			if err != nil {
				log.Fatal(err)
			}
		}

		pathGoMod, err := filepath.Abs("go.mod")
		if err != nil {
			log.Fatal(err)
		}

		goModFile, err := os.Open(pathGoMod)
		if err != nil {
			log.Fatal(err)
		}
		defer goModFile.Close()

		goModBytes, err := io.ReadAll(goModFile)
		if err != nil {
			log.Fatal(err)
		}

		_, err = modfile.Parse("go.mod", goModBytes, nil)
		if err != nil {
			log.Fatal(err)
		}

		_, err = build.ImportDir(pathPackage, 0)
		if err != nil {
			log.Fatal(err)
		}

		goFiles, err := filepath.Glob(filepath.Join(pathPackage, "*.go"))
		if err != nil {
			log.Fatal(err)
		}

		toolCompileArgs := []string{
			"tool", "compile", "-o", "TODO", "-I", filepath.Join(build.Default.GOPATH, "pkg", "mod"),
		}

		toolCompileArgs = append(toolCompileArgs, goFiles...)

		cmd := exec.Command("go", toolCompileArgs...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal()
		}

		break

	default:
		log.Fatalf("invalid '%s' command", command)
	}
}
