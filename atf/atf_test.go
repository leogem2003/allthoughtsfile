package main

import (
	"fmt"
	"path/filepath"
	"os"

	atf "github.com/leogem2003/allthoughtsfiles"

	"testing"
)
var root1, root2, settingsPath string

func TestMain(m *testing.M) {
	root1 = atf.GetTmpName([]string{"atf", "test_cli", "src"})
	root2 = atf.GetTmpName([]string{"atf", "test_cli", "dest"})
	var err error
	settingsPath, err = atf.MakeSettings("atf")
	if err != nil {
		fmt.Printf("Error while writing settings: %v", err)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(settingsPath)
	os.Exit(code)
}

func Test(t *testing.T) {
	os.RemoveAll(root1)
	os.RemoveAll(root2)
	atf.MakePlayground(root1, []string{"a/f1.txt", "a/f2.txt", "b/f1.txt", ".venv/python", ".zshrc"})
	atf.MakePlayground(root2, []string{""})
	arg1 := []string{"run", "main.go", "--debug", "--settings", settingsPath, root1}
	arg2 := []string{"run", "main.go", "--debug", "--settings", settingsPath, root2}
	atf.RunPrg(arg1,arg2,t)
	atf.CheckEqual(root1,root2,t)	
	
	os.WriteFile(filepath.Join(root1, "a/f1.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(root2, "b/f1.txt"), []byte("b"), 0644)
	atf.RunPrg(arg1,arg2,t)		
	atf.CheckEqual(root1,root2,t)

	os.Remove(filepath.Join(root1, "a/f1.txt"))
	os.Remove(filepath.Join(root2, "b/f1.txt"))
	atf.RunPrg(arg1,arg2,t)
	atf.CheckEqual(root1,root2,t)
}

func TestExclude(t *testing.T) {
	arg1 := []string{"run", "main.go", "--debug", "--settings", settingsPath, "--filter", "IgnoreDotFolders", root1}
	arg2 := []string{"run", "main.go", "--debug", "--settings", settingsPath, "--filter", "IgnoreDotFolders", root2}
	os.RemoveAll(root1)
	os.RemoveAll(root2)
	atf.MakePlayground(root1, []string{"a/f1.txt", "a/f2.txt", "b/f1.txt", "src/.venv/python", ".zshrc"})
	atf.MakePlayground(root2, []string{""})
	atf.RunPrg(arg1,arg2,t)
	atf.CheckEqual(root1,root2,t, ".venv")	

	os.RemoveAll(root1)
	os.RemoveAll(root2)
	atf.MakePlayground(root1, []string{"a/f1.txt", "a/f2.txt", "b/f1.txt", "src/.venv/python", ".zshrc"})
	atf.MakePlayground(root2, []string{""})
	arg1 = []string{"run", "main.go", "--debug", "--settings", settingsPath, "--filter", "IgnoreDotFiles", root1}
	arg2 = []string{"run", "main.go", "--debug", "--settings", settingsPath, "--filter", "IgnoreDotFiles", root2}
	atf.RunPrg(arg1,arg2,t)
	
	atf.CheckEqual(root1,root2,t, ".zshrc")	

}
