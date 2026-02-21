package main_test 
import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"testing"

	dc "github.com/leogem2003/directchan"
	// server "github.com/leogem2003/directchan/server"
	atf "github.com/leogem2003/allthoughtsfiles"
	dccp "github.com/leogem2003/allthoughtsfiles/dccp"
)

//actually requires server listening on 8080

func TestFile(t *testing.T) {
	const port = "8080"
	settings := dc.ConnectionSettings{
		Signaling:"ws://0.0.0.0:"+port,
		STUN:[]string{"stun:stun.l.google.com:19302"},
		Key:"ab",
		BufferSize:1024,
	}

	atf.SetDebugMode(true)
	sync := make(chan bool, 1)

	// create file in random tmp	
	in_dir := atf.GetTmpName([]string{"dccp_test"})
	input_path := filepath.Join(in_dir, "input.txt")	
	out_dir := atf.GetTmpName([]string{"dccp_test_out"})
	output_path := filepath.Join(out_dir, "input.txt")

	if err := os.MkdirAll(in_dir, 0755); err != nil {
		t.Errorf("Error while creating direcory %v, %v", in_dir, err)
		return
	}
	
	if err := os.MkdirAll(out_dir, 0755); err != nil {
		t.Errorf("Error while creating direcory %v, %v", in_dir, err)
		return
	}
	payload := make([]byte, 4096)
	rand.Read(payload)
	err := os.WriteFile(input_path, payload, 0644)
	if err != nil {
		t.Errorf("Error while writing file: %v", err)
	}

	go func() {
		sync<-true
		log.Println("creating sender channel")
		conn1, err := dc.FromSettings(&settings)
		defer conn1.CloseAll()

		if err != nil {
			t.Errorf("Error while opening sender channel: %v", err)
			return
		}
		sync <- true
		err = dccp.Send(conn1, input_path)
		if err != nil {
			t.Errorf("Error while sending: %v", err)
		}
		sync <- true
	}()

	<-sync

	conn2, err := dc.FromSettings(&settings)
	defer conn2.CloseAll()
	if err != nil {
		t.Errorf("Error while opening the recv channel: %v", err)
		return
	}

	<-sync
	err = dccp.Receive(conn2, out_dir)
	if err != nil {
		t.Errorf("Error while receiving: %v", err)
	}

	<-sync
	read_buf, err := os.ReadFile(output_path)
	if err != nil {
		t.Errorf("Error while reading output file: %v", err)
	}

	// test file equality
	if !slices.Equal(read_buf, payload) {
		t.Errorf("File content differ")
	}

	os.RemoveAll(in_dir)
	os.RemoveAll(out_dir)
}

func TestDir(t *testing.T) {
	const port = "8080"
	settings := dc.ConnectionSettings{
		Signaling:"ws://0.0.0.0:"+port,
		STUN:[]string{"stun:stun.l.google.com:19302"},
		Key:"cd",
		BufferSize:1024,
	}
	atf.SetDebugMode(true)
	sync := make(chan bool, 1)

	// create file in random tmp	
	in_dir := atf.GetTmpName([]string{"dccp_test"})
	input_path := atf.PathJoin([]string{in_dir, "input.txt"})	
	out_dir := atf.GetTmpName([]string{"dccp_test_out"})

	if err := os.MkdirAll(in_dir, 0755); err != nil {
		t.Errorf("Error while creating direcory %v, %v", in_dir, err)
		return
	}
	
	if err := os.MkdirAll(out_dir, 0755); err != nil {
		t.Errorf("Error while creating direcory %v, %v", in_dir, err)
		return
	}
	payload := make([]byte, 4096)
	rand.Read(payload)
	err := os.WriteFile(input_path, payload, 0644)
	if err != nil {
		t.Errorf("Error while writing file: %v", err)
	}

	go func() {
		sync<-true
		log.Println("creating sender channel")
		conn1, err := dc.FromSettings(&settings)
		defer conn1.CloseAll()

		if err != nil {
			t.Errorf("Error while opening sender channel: %v", err)
			return
		}
		sync <- true
		err = dccp.Send(conn1, in_dir)
		if err != nil {
			t.Errorf("Error while sending: %v", err)
		}
		sync <- true
	}()

	<-sync

	conn2, err := dc.FromSettings(&settings)
	defer conn2.CloseAll()
	if err != nil {
		t.Errorf("Error while opening the recv channel: %v", err)
		return
	}

	<-sync
	err = dccp.Receive(conn2, out_dir)
	if err != nil {
		t.Errorf("Error while receiving: %v", err)
	}

	<-sync
	file_path := atf.PathJoin([]string{out_dir, input_path})
	read_buf, err := os.ReadFile(file_path)
	if err != nil {
		t.Errorf("Error while reading output file: %v", err)
		return
	}

	// test file equality
	if !slices.Equal(read_buf, payload) {
		t.Errorf("File content differ")
	}

	os.RemoveAll(in_dir)
	os.RemoveAll(out_dir)
}


func TestCrypt(t *testing.T) {
	const port = "8080"
	settings := dc.ConnectionSettings{
		Signaling:"ws://0.0.0.0:"+port,
		STUN:[]string{"stun:stun.l.google.com:19302"},
		Key:"ab",
		BufferSize:1024,
	}

	atf.SetDebugMode(true)
	sync := make(chan bool, 1)

	// create file in random tmp	
	in_dir := atf.GetTmpName([]string{"dccp_test"})
	input_path := atf.PathJoin([]string{in_dir, "input.txt"})	
	out_dir := atf.GetTmpName([]string{"dccp_test_out"})
	output_path := atf.PathJoin([]string{out_dir, "input.txt"})

	if err := os.MkdirAll(in_dir, 0755); err != nil {
		t.Errorf("Error while creating direcory %v, %v", in_dir, err)
		return
	}
	
	if err := os.MkdirAll(out_dir, 0755); err != nil {
		t.Errorf("Error while creating direcory %v, %v", in_dir, err)
		return
	}
	payload := make([]byte, 4096)
	rand.Read(payload)
	err := os.WriteFile(input_path, payload, 0644)
	if err != nil {
		t.Errorf("Error while writing file: %v", err)
	}

	key := dc.CreateKey(32)
	cypher, err := dc.NewAESGCM(key)
	if err != nil {
		t.Errorf("%v", err)
	}
	
	go func() {
		sync<-true
		log.Println("creating sender channel")
		conn1, err := dc.FromSettings(&settings)
		defer conn1.CloseAll()

		if err != nil {
			t.Errorf("Error while opening sender channel: %v", err)
			return
		}
		chann := dc.NewAESConnection(conn1, cypher)
		sync <- true
		err = dccp.Send(chann, input_path)
		if err != nil {
			t.Errorf("Error while sending: %v", err)
		}
		sync <- true
	}()

	<-sync

	conn2, err := dc.FromSettings(&settings)
	defer conn2.CloseAll()
	if err != nil {
		t.Errorf("Error while opening the recv channel: %v", err)
		return
	}
	chann := dc.NewAESConnection(conn2, cypher)
	<-sync
	err = dccp.Receive(chann, out_dir)
	if err != nil {
		t.Errorf("Error while receiving: %v", err)
	}

	<-sync
	read_buf, err := os.ReadFile(output_path)
	if err != nil {
		t.Errorf("Error while reading output file: %v", err)
	}

	// test file equality
	if !slices.Equal(read_buf, payload) {
		t.Errorf("File content differ")
	}

	os.RemoveAll(in_dir)
	os.RemoveAll(out_dir)
}

func TestCLI(t *testing.T) {
	root1 := atf.GetTmpName([]string{"dccp", "test_cli", "src"})
	atf.MakePlayground(root1, []string{"f1.txt"})
	root2 := atf.GetTmpName([]string{"dccp", "test_cli", "dest"})
	atf.MakePlayground(root2, []string{""})

	filePath := filepath.Join(root1, "f1.txt")
	os.WriteFile(filePath, []byte("hello"), 0644)

	settingsPath, err := atf.MakeSettings("dccp")
	if err != nil {
		t.Fatalf("Error while writing settings: %v", err)
	}

	arg1 := []string{"run", "main.go",
		"--debug", "--settings", settingsPath, "send", filePath}
	arg2 := []string{"run", "main.go",
		"--debug", "--settings", settingsPath, "recv", root2}
	atf.RunPrg(arg1,arg2,t)
	atf.CheckEqual(root1,root2,t)	
	
	os.RemoveAll(root1)
	os.RemoveAll(root2)
	os.RemoveAll(settingsPath)
}
