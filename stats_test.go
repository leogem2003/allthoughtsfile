package atf

import (
	"os"
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestStatCreation(t *testing.T) {
	SetDebugMode(true)
	in_dir := GetTmpName([]string{"stat_test"})
	f1 := PathJoin([]string{in_dir, "f1.txt"})
	d1 := PathJoin([]string{in_dir, "d1"})
	f2 := PathJoin([]string{in_dir, "d1", "f2.txt"})

	if err := os.MkdirAll(d1, 0755); err != nil {
		t.Errorf("Error while creating direcory %v, %v", in_dir, err)
		return
	}

	if err := os.WriteFile(f1, []byte(""), 0755); err != nil {
		t.Errorf("Error while creating file %v, %v", f1, err)
		return
	}

	if err := os.WriteFile(f2, []byte(""), 0700); err != nil {
		t.Errorf("Error while creating file %v, %v", f2, err)
		return
	}

	stats, err := CreateStats(in_dir, func(s string) bool {return true})
	if err != nil {
		t.Errorf("Unexpected error while creating stats %v", err)
		return
	}
	if _, ok := stats[in_dir]; !ok {
		t.Errorf("%s not found in stats", in_dir)
	} 
	
	if _, ok := stats[f1]; !ok {
		t.Errorf("%s not found in stats", f1)
	}
	
	if _, ok := stats[d1]; !ok {
		t.Errorf("%s not found in stats", d1)
	}
	
	if _, ok := stats[f2]; !ok {
		t.Errorf("%s not found in stats", f2)
	}
}

func TestDiff(t *testing.T) {
	s1 := make(Stats)
	s2 := make(Stats)
	
	s1["a"] = FileInfo{}
	s1["a/b"] = FileInfo{}

	s2["a"] = FileInfo{Name:"aaa"}
	s2["a/c"] = FileInfo{}

	sub := StatsKeyDiff(s1, s2)
	val := StatsValueDiff(s1, s2)

	expsub := []string{"a/b"}
	expval := []string{"a"}

	if !slices.Equal(expsub, sub) {
		t.Errorf("Wrong sub: expected %v got %v", expsub, sub)
	}

	if !slices.Equal(val, expval) {
		t.Errorf("Wrong val: expected %v got %v", expval, val)
	}
}

func TestJson(t *testing.T) {
	file := FileInfo{
			Name:    "example.txt",
			Size:    12345,
			Mode:    0644,
			ModTime: time.Date(2025, 1, 2, 3, 4, 5, 6, time.UTC),
			IsDir:   false,
	}
	s1 := make(Stats)
	s1["a"] = file

	// Marshal
	data, err := StatsToJSON(s1) 
	if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal
	s2, err := StatsFromJSON(data)
	if err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Compare
	if !reflect.DeepEqual(s1, s2) {
			t.Fatalf("round-trip mismatch:\noriginal: %#v\ndecoded:  %#v",
					s1, s2)
	}
}
