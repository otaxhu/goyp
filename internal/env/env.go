package env

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var envVars map[string]string

// Inits envVars map
func init() {
	// Default values from 'go env' command
	c := exec.Command("go", "env", "-json")

	b := strings.Builder{}
	errB := strings.Builder{}

	c.Stdout = &b
	c.Stderr = &errB

	err := c.Run()
	if x, ok := err.(*exec.ExitError); ok {
		panic(fmt.Sprintf("goyp: could not load env using 'go env' command, program exited with code %d: %s", x.ExitCode(), errB.String()))
	} else if err != nil {
		panic("goyp: could not load env using 'go env' command: " + err.Error())
	}

	err = json.Unmarshal([]byte(b.String()), &envVars)
	if err != nil {
		panic("goyp: invalid json returned by 'go env' command: " + err.Error())
	}

	// Overwrite default values with wanted values
	for _, e := range os.Environ() {
		k, v, _ := strings.Cut(e, "=")
		envVars[k] = v
	}
}

// Retrieves environment variable with specified key
//
// Has some default values pre-written like GOOS and GOARCH if those are not specified
func Getenv(key string) string {
	return envVars[key]
}
