package main

import (
	"runtime/debug"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/cmd"
)

var Version = "dev"

func main() {
	if Version == "dev" || Version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				Version = bi.Main.Version
			}
		}
	}

	cmd.Execute(Version)
}
