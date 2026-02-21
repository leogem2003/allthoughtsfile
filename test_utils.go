package atf

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	dc "github.com/leogem2003/directchan"
	"testing"
)

func logWithPrefix(prefix string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Fprintf(os.Stderr, "%s %s\n", prefix, scanner.Text())
	}
}


func MakePlayground(root string, files []string) {
	for _,f := range files {
		path := PathJoin([]string{root, f})
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte{}, 0644)
	}	
}


func DirsEqual(dir1, dir2 string) (bool, error) {
	d1, err := os.ReadDir(dir1)
	if err != nil { return false, err }
	d2, err := os.ReadDir(dir2)
	if err != nil { return false, err }

	if len(d1) != len(d2) { return false, nil }

	for i := range d1 {
		info1, _ := d1[i].Info()
		info2, _ := d2[i].Info()

		// Check Name, IsDir, and Permissions (FileMode)
		if d1[i].Name() != d2[i].Name() || 
		   d1[i].IsDir() != d2[i].IsDir() || 
		   info1.Mode() != info2.Mode() {
			return false, nil
		}

		// Recurse if it's a directory
		if d1[i].IsDir() {
			equal, err := DirsEqual(filepath.Join(dir1, d1[i].Name()), filepath.Join(dir2, d2[i].Name()))
			if !equal || err != nil { return equal, err }
		}
	}
	return true, nil
}
 
func CheckEqual(root1, root2 string, t *testing.T) {
	equal, err := DirsEqual(root1, root2)
	if err != nil {
		t.Fatalf("Error while comparing dirs: %v", err)
	} else if !equal {
		t.Fatal("Result dirs differ")
	}
}


func RunPrg(arg1, arg2 []string, t *testing.T) {
	var wg sync.WaitGroup
	cmd1 := exec.Command("go", arg1...)
	cmd2 := exec.Command("go", arg2...)

	// Get pipes for Stderr
	stderr1, _ := cmd1.StderrPipe()
	stderr2, _ := cmd2.StderrPipe()

	wg.Add(2)

	// Start Program 1
	if err := cmd1.Start(); err != nil {
		t.Fatal(err)
	}
	go func() {
		defer wg.Done()
		logWithPrefix("\033[32m[PROG-A]\033[0m", stderr1) // Green prefix
		cmd1.Wait()
	}()

	// Start Program 2
	if err := cmd2.Start(); err != nil {
		t.Fatal(err)
	}
	go func() {
		defer wg.Done()
		logWithPrefix("\033[34m[PROG-B]\033[0m", stderr2) // Blue prefix
		cmd2.Wait()
	}()

	wg.Wait()
}

func MakeSettings(key string) (string, error) {
	settings := dc.ConnectionSettings{
		Signaling:"ws://0.0.0.0:8080",
		STUN:[]string{"stun:stun.l.google.com:19302"},
		Key:key,
		BufferSize:1,
	}

	settingsDir := GetTmpName([]string{key, "test_cli", "settings"})
	os.MkdirAll(settingsDir, 0755)
	settingsPath := filepath.Join(settingsDir, "settings.json")	
	b, _ := json.Marshal(settings)
	return settingsPath, os.WriteFile(settingsPath, b, 0644)
}
