package atf

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
)

type Stats = map[string]FileInfo
type HashStats = map[uint64]uint64

type StatPair struct {
	Name string
	Info FileInfo
}

func CreateStats(dir string, policy func(string) bool) (Stats, error) {
	stats := make(Stats)
	dirFunc := func(path string, info fs.DirEntry, err error) error {
		if err != nil {
				return err
		}
		if !policy(path) {
			return nil
		}

		fileInfo, err := info.Info()
		if err != nil {
			return err
		}
		stats[path] = CloneInfo(os.FileInfo(fileInfo))
		return nil
	}

	filepath.WalkDir(dir, dirFunc)
	return stats, nil
}

// Returns keys that are in a but not in b
func StatsKeyDiff(a, b Stats) []string {
	diff := make([]string, 0, 1) 
	for ka, _ := range a {
		if _, ok := b[ka]; !ok {
			diff = append(diff, ka)
		}
	}

	return diff
}


// Returns keys that are associated with different
// values in a and b
func StatsValueDiff(a, b Stats) []string {
	diff := make([]string, 0, 1) 
	for ka, va := range a {
	  vb, ok := b[ka];
		if ok && vb != va {
			diff = append(diff, ka)
		}
	}

	return diff
}

func StatsToHash(s Stats) HashStats {
	hashed := make(HashStats, len(s))
	for k, v := range s {
		hashed[HashString(k)] = v.Hash()
	}
	return hashed
}

func StatsToJSON(s Stats) ([]byte, error) {
	return json.Marshal(s)
}

func StatsFromJSON(data []byte) (Stats, error) {
	s := make(Stats)
	err := json.Unmarshal(data, &s)
	return s, err
}
