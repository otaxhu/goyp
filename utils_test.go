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

func TestInitStdPackagesList(t *testing.T) {
	err := initStdPackagesList()
	if err != nil {
		t.Fatal(err)
	}
}

func TestIsStdPackage(t *testing.T) {

	testCases := []struct {
		Name         string
		Package      string
		ExpectedBool bool
	}{
		{
			Name:         "Success_fmt",
			Package:      "fmt",
			ExpectedBool: true,
		},
		{
			Name:         "Fail_non_std",
			Package:      "non_std",
			ExpectedBool: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			v := isStdPackage(tc.Package)
			if v != tc.ExpectedBool {
				t.Fatalf("expected %v got %v", tc.ExpectedBool, v)
			}
		})
	}

}
