package atf

import (
	"slices"
	"maps"
	"testing"
	"path/filepath"
	"fmt"
)

func testPolicy(policy FilterFunc, dir string, expected []string,
	info string, t *testing.T) {
	stats, err := CreateStats(dir, policy)
	if err != nil {
		t.Errorf("Error while creating stats: %v", err)
	}
	selected := slices.Collect(maps.Keys(stats))
	slices.Sort(selected)
	slices.Sort(expected)
	if !slices.Equal(selected, expected) {
		fmt.Printf("f:%s, s:%s,\n",selected[0], expected[0])
		t.Errorf("%s: Expected %v, selected %v\n", info, expected, selected)
	}	
}

func TestExcludeDots(t *testing.T) {
	tmp := filepath.Clean(GetTmpName([]string{"atf", "test_policy"}))
	files := []string { "a/.b/file.txt", "a/b/file.txt", "a/b/.file.txt"}
	MakePlayground(tmp, files)

	expectedExcludeDots := []string {"a","a/b","a/b/file.txt"}
	expectedExcludeDirs := []string {"a","a/b","a/b/file.txt", "a/b/.file.txt"}
	expectedExcludeFiles := []string {"a","a/b","a/b/file.txt", "a/.b", "a/.b/file.txt"}
	expectedExcludeSuffix := []string{"a", "a/b", "a/.b"}

	testPolicy(IgnoreDot, tmp, expectedExcludeDots, "IgnoreDot", t)
	testPolicy(IgnoreDotFolders, tmp, expectedExcludeDirs, "IgnoreDotFolders", t)
	testPolicy(IgnoreDotFiles, tmp, expectedExcludeFiles, "IgnoreDotFiles", t)
	testPolicy(MakeIgnoreSuffix("file.txt"), tmp, expectedExcludeSuffix, "ExcludeSuffix", t)
}
