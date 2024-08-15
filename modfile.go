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
