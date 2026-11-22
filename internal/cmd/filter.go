package cmd

import (
	"flag"
	"strings"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/engine"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func registerFileFilterFlags(fs *flag.FlagSet) (*stringSliceFlag, *stringSliceFlag) {
	var include stringSliceFlag
	var exclude stringSliceFlag
	fs.Var(&include, "file", "Include only managed files matching the glob pattern (repeatable)")
	fs.Var(&exclude, "exclude-file", "Exclude managed files matching the glob pattern (repeatable)")
	return &include, &exclude
}

func buildFileFilter(include, exclude *stringSliceFlag) (engine.FileFilter, error) {
	return engine.NewFileFilter([]string(*include), []string(*exclude))
}
