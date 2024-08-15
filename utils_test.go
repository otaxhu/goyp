package main

import (
	"go/version"
	"testing"
)

func TestGoToolchainVersion(t *testing.T) {
	v, err := goToolchainVersion()
	if err != nil {
		t.Fatal(err)
	}
	if !version.IsValid(v) {
		t.Fatalf("'%s' in not a valid go version", v)
	}
}
