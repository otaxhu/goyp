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
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type env map[string]string

func (e env) Get(s string) string {
	if e == nil {
		return ""
	}
	return e[s]
}

func parseGoEnv() (env, error) {
	b := bytes.Buffer{}
	errB := bytes.Buffer{}

	c := exec.Command("go", "env", "-json")
	c.Stderr = &errB
	c.Stdout = &b

	err := c.Run()
	if x, ok := err.(*exec.ExitError); ok {
		return nil, fmt.Errorf("parseGoEnv: command 'go env -json' exited with code %d: %s", x.ExitCode(), errB.Bytes())
	} else if err != nil {
		return nil, fmt.Errorf("parseGoEnv: %w", err)
	}

	if errB.Len() > 0 {
		return nil, fmt.Errorf("parseGoEnv: %s", errB.Bytes())
	}

	var m = env{}

	err = json.Unmarshal(b.Bytes(), &m)
	if err != nil {
		return nil, fmt.Errorf("parseGoEnv: json.Unmarshal: %w", err)
	}

	// TODO: add some aditional variables like GOYP_BUILD_OS and GOYP_BUILD_ARCH that may take a
	// comma separated list of OS and arch platforms when building non-main package. GOOS and
	// GOARCH will still work as usual but if it finds GOYP_BUILD_OS or GOYP_BUILD_ARCH it will
	// prefer those over GOOS or GOARCH

	return m, nil
}
