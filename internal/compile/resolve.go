package compile

import (
	"go/build"

	"golang.org/x/mod/modfile"
)

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
