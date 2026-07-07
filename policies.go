package atf

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type FilterFunc = func(string) bool

var slashdot = string(os.PathSeparator) + "."

func AllowEverything(_ string) bool {
	return true
}

func IgnoreDot(path string) bool {
	return !strings.Contains(path, slashdot)
}

func IgnoreDotFolders(path string) bool {
	info, _ := os.Stat(path)

	if info.IsDir() {
		return !strings.Contains(filepath.Clean(path), slashdot)
	}

	return !strings.Contains(filepath.Dir(path), slashdot)
}

func IgnoreDotFiles(path string) bool {
	info, _ := os.Stat(path)

	if info.IsDir() {
		return true
	}

	return !strings.HasPrefix(filepath.Base(path), ".")
}

func MakeIgnoreSuffix(suffix string) func(string) bool {
	return func(path string) bool {
		info, _ := os.Stat(path)

		if info.IsDir() {
			return true
		}
		return !strings.HasSuffix(path, suffix)
	}
}

var Policies = map[string]FilterFunc{
	"AllowEverything":  AllowEverything,
	"IgnoreDot":        IgnoreDot,
	"IgnoreDotFolders": IgnoreDotFolders,
	"IgnoreDotFiles":   IgnoreDotFiles,
}

var PolicyNames []string = slices.Collect(maps.Keys(Policies))
