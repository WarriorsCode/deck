package deck

import (
	"os/exec"
	"strings"
)

// Version is injected at build time via ldflags:
//
//	go build -ldflags="-s -w -X github.com/warriorscode/deck.Version=$(git describe --tags)" ./cmd/deck
var Version = ""

func init() {
	if Version == "" {
		if out, _ := exec.Command("git", "describe", "--tags").Output(); len(out) > 0 {
			Version = strings.TrimSpace(string(out))
		} else {
			Version = "dev"
		}
	}
}
