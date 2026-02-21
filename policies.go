package atf 
import (
	"os"
	"path/filepath"
	"strings"
)
var slashdot = string(os.PathSeparator)+"."

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

