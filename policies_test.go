package atf

import (
	"slices"
	"testing"
	"os"
	"path/filepath"
	"fmt"
)

func testPolicy(policy func(string)bool, files []string, expected []string,
	info string, t *testing.T) {
	selected := make([]string, 0)
	for _,f := range files {
		if policy(f){
			selected = append(selected, f)	
		}
	}
	slices.Sort(selected)
	slices.Sort(expected)
	if !slices.Equal(selected, expected) {
		fmt.Printf("f:%s, s:%s,\n",selected[0], expected[0])
		t.Errorf("%s: Expected %v, selected %v\n", info, expected, selected)
	}	
}

func makeTmp(tmp string, p []string) []string {
	tmpList := make([]string,0)
	for _, f := range p {
		tmpList = append(tmpList, PathJoin([]string{tmp, f}))
	}

	return tmpList
}

func TestExcludeDots(t *testing.T) {
	tmp := filepath.Clean(GetTmpName([]string{"atf", "test_policy"}))
	files := []string { "a/.b/file.txt", "a/b/file.txt", "a/b/.file.txt"}
	folders := []string {"a", "a/.b", "a/b" }
	
	tmpFolders := make([]string,0, len(folders))
	for _, f := range folders {
		tmpFolder := PathJoin([]string{tmp, f})	
		tmpFolders = append(tmpFolders, tmpFolder)
		if err := os.MkdirAll(tmpFolder, 0755); err != nil {
			t.Fatalf("Cannot create directory: %v", err)
		}
	}

	
	tmpFiles := make([]string,0, len(files))
	for _, f := range files {
		tmpFile := PathJoin([]string{tmp, f})	
		tmpFiles = append(tmpFiles, tmpFile)
		if err := os.WriteFile(tmpFile, []byte{}, 0644); err != nil {
			t.Fatalf("Cannot write file: %v", err)
		}
	}

	expectedExcludeDots := []string {"a","a/b","a/b/file.txt"}
	tmpExcDots := makeTmp(tmp, expectedExcludeDots)

	tmpTot := append(tmpFiles, tmpFolders...)

	expectedExcludeDirs := []string {"a","a/b","a/b/file.txt", "a/b/.file.txt"}
	tmpExcDirs := makeTmp(tmp, expectedExcludeDirs)

	expectedExcludeFiles := []string {"a","a/b","a/b/file.txt", "a/.b", "a/.b/file.txt"}
	tmpExcFiles := makeTmp(tmp, expectedExcludeFiles)

	testPolicy(IgnoreDot, tmpTot, tmpExcDots, "IgnoreDot", t)
	testPolicy(IgnoreDotFolders, tmpTot, tmpExcDirs, "IgnoreDotFolders", t)
	testPolicy(IgnoreDotFiles, tmpTot, tmpExcFiles, "IgnoreDotFiles", t)
}
