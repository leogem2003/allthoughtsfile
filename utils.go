package atf
import (
	"flag"
	"hash/fnv"
	"io"
	"log"
	"math/rand"
	"strings"
	"strconv"
	"os"
)

// Join path parts using OS separator
func PathJoin(parts []string) string {
	return strings.Join(parts, string(os.PathSeparator))
}

// Create a temporary directory containing suffix with a random number added
// Structure: <os_tempdir>/<suffix||random>
func GetTmpName(suffix []string) string {
	l := make([]string, 1, len(suffix)+1)
	l[0] = os.TempDir()
	suffix[len(suffix)-1] += strconv.Itoa(rand.Intn(1024))
	l = append(l, suffix...)
	return PathJoin(l)
}

var DefaultSettings = PathJoin([]string{os.Getenv("HOME"), ".config", "allthoughtsfile", "settings.json"})

func SettingsFlag(target *string) {
	flag.StringVar(target, "settings", DefaultSettings, "path to settings.json")
}


func DebugFlag(target *bool) {
	flag.BoolVar(target, "debug", false, "enable debugging")
}

func AESFlag(target *string) {
	flag.StringVar(target, "aes", "", "AES key file")
}

func SetDebugMode(debug bool) {
	if debug {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(io.Discard)
	}
}

func HashString(s string) uint64 {
    h := fnv.New64a()
    h.Write([]byte(s))
    return h.Sum64()
}
