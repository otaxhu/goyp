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
	"io"
	"os"
	"path/filepath"
)

// Implements [golang.org/x/mod/zip.File] interface using 'os' package functions.
//
// Interesting property:
//
// root + filename = abs path to file in OS
type moduleFile struct {
	// Absolute path to root of Module
	root string

	// Relative from module
	filename string
}

func (m moduleFile) Path() string {
	return m.filename
}

func (m moduleFile) Open() (io.ReadCloser, error) {
	return os.Open(filepath.Join(m.root, m.filename))
}

func (m moduleFile) Lstat() (os.FileInfo, error) {
	return os.Lstat(filepath.Join(m.root, m.filename))
}
