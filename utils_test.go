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
