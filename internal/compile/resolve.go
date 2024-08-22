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

var stdPackages = make(map[string]struct{})

// Initialize stdPackages map
func init() {
	pkgs, err := packages.Load(nil, "std")
	if err != nil {
		panic("goyp: could not load std packages: " + err.Error())
	}
	for _, p := range pkgs {
		stdPackages[p.PkgPath] = struct{}{}
	}
}

type queueElement struct {
	pkg               *build.Package
	importPath        string
	unresolvedImports map[string]struct{}
}

type Resolver struct {
	// Targeted platform
	GOOS, GOARCH string

	ModCachePath       string
	GoToolchainVersion string
}

func (r *Resolver) ResolveDeps(modDir string, modFile *modfile.File) (pkgsToCompile []*build.Package, zipModFound []string, err error) {
	ctx := &build.Context{
		GOARCH:   r.GOARCH,
		GOOS:     r.GOOS,
		Compiler: "gc",
	}

	queue := []queueElement{}

	pkgsResolved := maps.Clone(stdPackages)
	pkgsListed := maps.Clone(stdPackages)

	for _, m := range modFile.Require {
		modRoot := filepath.Join(r.ModCachePath, filepath.FromSlash(m.Mod.Path)+"@"+m.Mod.Version)
		pathToZipMod := filepath.Join(modRoot, ".goyp", r.GOOS+"@"+r.GOARCH+"@"+r.GoToolchainVersion+".zip")
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
				modDir, modBase := pathpkg.Split(m.Mod.Path)
				// Files in zip must be named the following: modulebasename/**/*.a OR modulebasename.a
				if !strings.HasPrefix(zf.Name, modBase) {
					return nil, nil, fmt.Errorf("ResolveDeps(): zip modules: invalid zip file, contains following file '%s' found in module '%s'", zf.Name, m.Mod.Path)
				}
				zipModFound = append(zipModFound, pathToZipMod)
				pkgImportPath := pathpkg.Join(modDir, zf.Name[:len(zf.Name)-len(ext)])
				pkgsListed[pkgImportPath] = struct{}{}
				pkgsResolved[pkgImportPath] = struct{}{}
			}
		} else if err != os.ErrNotExist {
			return nil, nil, err
		}

		// At this point zip packages has been added to pkgsResolved, pkgsListed
		// and zipModFound variables, or zip file has not been found, in which case,
		// it means that module doesn't provide object code packages, keep searching in
		// module source code

		err = filepath.WalkDir(modRoot, func(path string, entry fs.DirEntry, cbErr error) error {
			if cbErr != nil {
				return cbErr
			}
			// We are looking only for directories, skips file
			if !entry.IsDir() {
				return nil
			}

			relFromModRoot, err := filepath.Rel(modRoot, path)
			if err != nil {
				return err
			}
			// Inside of .goyp should not be any source code files,
			// but there can be a .goyp dir not rooted in modRoot
			if relFromModRoot == ".goyp" {
				return filepath.SkipDir
			}

			pkg, err := ctx.ImportDir(path, 0)
			if err != nil {
				return err
			}

			unresolved := map[string]struct{}{}
			for _, imp := range pkg.Imports {
				if _, ok := pkgsResolved[imp]; !ok {
					unresolved[imp] = struct{}{}
				}
			}

			pkgImportPath := pathpkg.Join(m.Mod.Path, filepath.ToSlash(relFromModRoot))

			if len(unresolved) > 0 {
				queue = append(queue, queueElement{
					importPath:        pkgImportPath,
					pkg:               pkg,
					unresolvedImports: unresolved,
				})
			} else {
				pkgsToCompile = append(pkgsToCompile, pkg)
				pkgsResolved[pkgImportPath] = struct{}{}
			}

			pkgsListed[pkgImportPath] = struct{}{}

			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	// After resolving all of the required modules, resolve the targeted module
	err = filepath.WalkDir(modDir, func(path string, entry fs.DirEntry, cbErr error) error {
		if cbErr != nil {
			return cbErr
		}

		if !entry.IsDir() {
			return nil
		}

		pkg, err := ctx.ImportDir(path, 0)
		if err != nil {
			return err
		}

		unresolved := map[string]struct{}{}

		for _, imp := range pkg.Imports {
			if _, ok := pkgsResolved[imp]; !ok {
				unresolved[imp] = struct{}{}
			}
		}

		relFromModRoot, err := filepath.Rel(modDir, path)
		if err != nil {
			return err
		}

		pkgImportPath := pathpkg.Join(modFile.Module.Mod.Path, filepath.ToSlash(relFromModRoot))

		if len(unresolved) > 0 {
			queue = append(queue, queueElement{
				importPath:        pkgImportPath,
				pkg:               pkg,
				unresolvedImports: unresolved,
			})
		} else {
			pkgsToCompile = append(pkgsToCompile, pkg)
			pkgsResolved[pkgImportPath] = struct{}{}
		}

		pkgsListed[pkgImportPath] = struct{}{}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	for len(queue) > 0 {
		resolvedOne := false
		for i := 0; i < len(queue); {
			el := queue[i]

			for k := range el.unresolvedImports {
				if _, ok := pkgsListed[k]; !ok {
					return nil, nil, fmt.Errorf("ResolveDeps(): no module provides '%s' package, try to run: go get %s", k, k)
				}
				if _, ok := pkgsResolved[k]; ok {
					delete(el.unresolvedImports, k)
				}
			}
			if len(el.unresolvedImports) == 0 {
				pkgsToCompile = append(pkgsToCompile, el.pkg)
				pkgsResolved[el.importPath] = struct{}{}
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

// Returns a list of packages that p depends on in order to compile, all of the packages listed in
// slice must be compiled in the order the list is returned
func ResolveImports(p *build.Package, modFile *modfile.File, modCachePath string) ([]string, error) {
	// Steps:
	//
	// 1. Read go.mod file
	//
	// 2. Walk all of the modules required inside of modCachePath
	//
	// 3. If it finds a directory named '.goyp' then look inside of it for the following file:
	//
	//   <GOOS>_<GOARCH>_go<go-version>.zip
	//
	// Where <go-version> is current Go toolchain version, <GOOS> and <GOARCH> is the
	// targeted OS and platform architecture
	//
	// Any file inside of the zip will be treated as if it was relative to the module root,
	// also the zip must not contain any files other than '.a' ended files.
	//
	// 4. No matter if it finds '.goyp' directory or not, keep walking file by file, ignoring
	// files that starts with '.' or '_', parsing the directories walked using [build.ImportDir]
	// function
	//
	// 5. Packages in order to be compiled, they need all of their dependencies available as
	// object code format, so we need to put in a Queue the packages that their dependencies are,
	// for now, not available, but they will be available.
	//
	// The Queue algorithm will be something like this:
	//
	//   Queue := make([]struct{ Package *build.Package; UnresolvedImports map[string]struct{}; })
	//
	//   pkgsResolved := make(map[string]struct{})
	//
	//   pkgsListed := make(map[string]struct{})
	//
	//   for _, m := range requiredModules {
	//     modRoot := filepath.Join(goModCache, m.Path + "@" + m.Version)
	//     filepath.WalkDir(modRoot, func (path string, entry fs.DirEntry) error {
	//       if entry.IsDir() {
	//         if entry.Name() == ".goyp" {
	//           // Read .goyp contents
	//           // Add packages found to pkgsResolved and pkgsListed
	//           // Then...
	//           return filepath.SkipDir
	//         }
	//         pkg := build.ImportDir(path, 0)
	//
	//         unresolved := make(map[string]struct{})
	//
	//         for _, imp := range pkg.Imports {
	//           if _, ok := pkgsResolved[imp]; !ok {
	//             unresolved[imp] = struct{}
	//           }
	//         }
	//
	//         if len(unresolved) > 0 {
	//           Queue = append(Queue, { Package: pkg, UnresolvedImports: unresolved })
	//         } else {
	//           // Compile package, all of its dependencies are available in object code format
	//         }
	//
	//         pkgsListed[filepath.Rel(modRoot, path)] = struct{}
	//       }
	//       return nil
	//     })
	//   }
	//   for len(Queue) > 0 {
	//     for i := 0; i < len(Queue); {
	//       el := Queue[i]
	//       for k := range el.UnresolvedImports {
	//         if _, ok := pkgsListed[k]; !ok {
	//           log.Fatalf("no module provides '%s' package, try to run: go get %s", k, k)
	//         }
	//         if _, ok := pkgsResolved[k]; ok {
	//           delete(el.UnresolvedImports, k)
	//         }
	//       }
	//       if len(el.UnresolvedImports) == 0 {
	//         // Compile package, then...
	//         Queue = append(Queue[:i], Queue[i+1:])
	//       } else {
	//         i++
	//       }
	//     }
	//   }

	return nil, nil
}
