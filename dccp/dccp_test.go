package main_test 
import (
	"log"
	"math/rand"
	"os"
	"testing"
	"slices"

	dc "github.com/leogem2003/directchan"
	// server "github.com/leogem2003/directchan/server"
	atf "github.com/leogem2003/allthoughtsfiles"
	dccp "github.com/leogem2003/allthoughtsfiles/dccp"
)

//actually requires server listening on 8080

func TestFile(t *testing.T) {
	const port = "8080"
	settings1 := dc.ConnectionSettings{
		Signaling:"ws://0.0.0.0:"+port,
		STUN:[]string{"stun:stun.l.google.com:19302"},
		Key:"ab",
		Operation:0,
		BufferSize:1024,
	}
	settings2 := dc.ConnectionSettings{
		Signaling:"ws://0.0.0.0:"+port,
		STUN:[]string{"stun:stun.l.google.com:19302"},
		Key:"ab",
		Operation:1,
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

	go func() {
		sync<-true
		log.Println("creating sender channel")
		conn1, err := dc.FromSettings(&settings1)
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

	conn2, err := dc.FromSettings(&settings2)
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
	settings1 := dc.ConnectionSettings{
		Signaling:"ws://0.0.0.0:"+port,
		STUN:[]string{"stun:stun.l.google.com:19302"},
		Key:"cd",
		Operation:0,
		BufferSize:1024,
	}
	settings2 := dc.ConnectionSettings{
		Signaling:"ws://0.0.0.0:"+port,
		STUN:[]string{"stun:stun.l.google.com:19302"},
		Key:"cd",
		Operation:1,
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
		conn1, err := dc.FromSettings(&settings1)
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

	conn2, err := dc.FromSettings(&settings2)
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
