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

package compile

import (
	"archive/zip"
	"fmt"
	"go/build"
	"io/fs"
	"maps"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// Value true means that is a main package
//
// Otherwise is a importable package
var stdPackages = make(map[string]bool)

// Initialize stdPackages map
func init() {
	pkgs, err := packages.Load(nil, "std")
	if err != nil {
		panic("goyp: could not load std packages: " + err.Error())
	}
	for _, p := range pkgs {
		// packages.Load doesn't return main packages
		stdPackages[p.PkgPath] = false
	}
}

type queueElement struct {
	// set pkg.ImportPath to every queueElement before adding to queue
	pkg               *build.Package
	unresolvedImports map[string]struct{}
}

type Resolver struct {
	ModCachePath       string
	GoToolchainVersion string
}

// ctxt is the build.Context struct that is going to be used for resolving
// packages
//
// modDir is the root directory of the module that is being processed,
// should be an absolute path
//
// targetPkg is an optional path parameter that is relative to modDir,
// specifying a main package that is required to be resolved, should be empty
// string if no main package is gonna be build
//
// modFile is the parsed go.mod file from the targeted module modDir
//
// # Returns:
//
// pkgsToCompile is a slice that contains all of the packages that the
// targeted module provides in source code format, including targetPkg if specified. They are
// sorted in a way such that the 'go tool compile' can compile them correctly
// with no dependency errors
//
// zipModFound is a slice that contains paths to zip modules, where you can find libraries in
// object code format
//
// err is non-nil only if:
//   - Dependencies cannot be resolved (missing packages, build constraints not satisfied, etc)
//   - if specified targetPkg, and targetPkg is not a main package or targetPkg is not a valid
//     relative path to modDir
//   - OS I/O error
//   - Circular dependency between packages
//   - Any package imports a main package
func (r *Resolver) ResolveDeps(ctxt *build.Context, modDir, targetPkg string, modFile *modfile.File) (pkgsToCompile []*build.Package, zipModFound []string, err error) {

	queue := []queueElement{}

	pkgsResolved := maps.Clone(stdPackages)
	pkgsListed := maps.Clone(stdPackages)

	if targetPkg != "" {
		pkg, err := ctxt.ImportDir(filepath.Join(modDir, targetPkg), 0)
		if err != nil {
			return nil, nil, err
		}
		if !pkg.IsCommand() {
			return nil, nil, fmt.Errorf("ResolveDeps(): target package '%s' is not a main package", filepath.Join(modDir, targetPkg))
		}
		unresolved := map[string]struct{}{}
		for _, imp := range pkg.Imports {
			if _, ok := pkgsResolved[imp]; !ok {
				unresolved[imp] = struct{}{}
			}
		}
		pkg.ImportPath = pathpkg.Join(modFile.Module.Mod.Path, targetPkg)
		if len(unresolved) > 0 {
			queue = append(queue, queueElement{
				pkg:               pkg,
				unresolvedImports: unresolved,
			})
		} else {
			pkgsToCompile = append(pkgsToCompile, pkg)
			// Do not add to pkgsResolved since this is a main package and cannot
			// be imported by any package
			//
			// pkgsResolved[pkg.ImportPath] = struct{}{}
		}
		// True because is a main package
		pkgsListed[pkg.ImportPath] = true
	}

	walkFn := func(modRoot, modPath string) error {
		return filepath.WalkDir(modRoot, func(path string, entry fs.DirEntry, cbErr error) error {
			if cbErr != nil {
				return cbErr
			}
			// We are looking only for directories, skips file
			if !entry.IsDir() {
				return nil
			}

			// ignore dirs that starts with '.' or '_'
			if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
				return filepath.SkipDir
			}

			pkg, err := ctxt.ImportDir(path, 0)
			if err != nil {
				return err
			}

			relFromModRoot, err := filepath.Rel(modRoot, path)
			if err != nil {
				return err
			}

			pkg.ImportPath = pathpkg.Join(modPath, filepath.ToSlash(relFromModRoot))

			// Early return if it's a main package, the only main package that their dependencies
			// will be resolved is the targetPkg
			if pkg.IsCommand() {
				// value used in queue loop to return an error if another package imports this
				pkgsListed[pkg.ImportPath] = true
				return nil
			}

			unresolved := map[string]struct{}{}
			for _, imp := range pkg.Imports {
				if _, ok := pkgsResolved[imp]; !ok {
					unresolved[imp] = struct{}{}
				}
			}

			if len(unresolved) > 0 {
				queue = append(queue, queueElement{
					pkg:               pkg,
					unresolvedImports: unresolved,
				})
			} else {
				pkgsToCompile = append(pkgsToCompile, pkg)
				pkgsResolved[pkg.ImportPath] = false
			}

			if _, ok := pkgsListed[pkg.ImportPath]; ok {
				return fmt.Errorf("ResolveDeps: package '%s' has been redeclared", pkg.ImportPath)
			}

			pkgsListed[pkg.ImportPath] = false

			return nil
		})
	}

	for _, m := range modFile.Require {
		modRoot := filepath.Join(r.ModCachePath, filepath.FromSlash(m.Mod.Path)+"@"+m.Mod.Version)
		pathToZipMod := filepath.Join(modRoot, ".goyp", ctxt.GOOS+"@"+ctxt.GOARCH+"@"+r.GoToolchainVersion+".zip")
		zipFile, err := os.Open(pathToZipMod)

		// If file is found, then read it and add entries to corresponding list and maps
		if err == nil {
			zipInfo, err := zipFile.Stat()
			if err != nil {
				zipFile.Close()
				return nil, nil, err
			}
			zipR, err := zip.NewReader(zipFile, zipInfo.Size())
			zipFile.Close()
			if err != nil {
				return nil, nil, err
			}
			for _, zf := range zipR.File {
				ext := pathpkg.Ext(zf.Name)
				if ext != ".a" || pathpkg.Base(zf.Name) == ".a" {
					return nil, nil, fmt.Errorf("ResolveDeps(): zip modules: invalid filename '%s' found in module '%s'", zf.Name, m.Mod.Path)
				}
				// Files in zip must be named the following: modulename/**/*.a OR modulename.a
				if !strings.HasPrefix(zf.Name, m.Mod.Path) {
					return nil, nil, fmt.Errorf("ResolveDeps(): zip modules: invalid zip file, contains following file '%s' found in module '%s'", zf.Name, m.Mod.Path)
				}
				zipModFound = append(zipModFound, pathToZipMod)
				pkgImportPath := zf.Name[:len(zf.Name)-len(ext)]
				// There is a conflict of packages, invalid zip module
				if _, ok := pkgsListed[pkgImportPath]; ok {
					return nil, nil, fmt.Errorf("ResolveDeps(): zip modules: invalid zip module '%s', package '%s' has been redeclared", pathToZipMod, pkgImportPath)
				}
				pkgsListed[pkgImportPath] = false
				pkgsResolved[pkgImportPath] = false
			}
		} else if err != os.ErrNotExist {
			return nil, nil, err
		}

		// At this point zip packages has been added to pkgsResolved, pkgsListed
		// and zipModFound variables, or zip file has not been found, in which case,
		// it means that module doesn't provide object code packages, keep searching in
		// module source code

		err = walkFn(modRoot, m.Mod.Path)
		if err != nil {
			return nil, nil, err
		}
	}

	// After resolving all of the required modules, resolve the targeted module
	err = walkFn(modDir, modFile.Module.Mod.Path)
	if err != nil {
		return nil, nil, err
	}

	for len(queue) > 0 {
		resolvedOne := false
		for i := 0; i < len(queue); {
			el := queue[i]

			for k := range el.unresolvedImports {
				if isMain, ok := pkgsListed[k]; !ok {
					return nil, nil, fmt.Errorf("ResolveDeps(): no module provides '%s' package, try to run: go get %s", k, k)
				} else if isMain {
					return nil, nil, fmt.Errorf("ResolveDeps(): invalid import, package '%s' is importing package main '%s'", el.pkg.ImportPath, k)
				}
				if _, ok := pkgsResolved[k]; ok {
					delete(el.unresolvedImports, k)
				}
			}
			if len(el.unresolvedImports) == 0 {
				pkgsToCompile = append(pkgsToCompile, el.pkg)
				pkgsResolved[el.pkg.ImportPath] = false
				queue = append(queue[:i], queue[i+1:]...)
				resolvedOne = true
			} else {
				i++
			}
		}
		if !resolvedOne {
			return nil, nil, fmt.Errorf("ResolveDeps(): cannot resolve dependencies due to circular dependency")
		}
	}

	return pkgsToCompile, zipModFound, nil
}
