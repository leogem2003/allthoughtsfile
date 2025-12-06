package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"io"
	"log"
	"slices"

	dc "github.com/leogem2003/directchan"
	atf "github.com/leogem2003/allthoughtsfiles"
)

var errorLog = log.New(os.Stderr, "ERROR: ", 0)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s: %s (send|recv) <file>\n", os.Args[0], os.Args[0])
	flag.PrintDefaults() 
}

func main() {
	var settingsPath string
	var debug bool

	atf.SettingsFlag(&settingsPath)
	atf.DebugFlag(&debug)

	flag.Usage = Usage
	flag.Parse()
	
	op := flag.Arg(0)
	target := flag.Arg(1)
	settings := new(dc.ConnectionSettings) 
	atf.SetDebugMode(debug)

	file, err := os.Open(settingsPath)
	if err != nil {
		errorLog.Fatalf("Failed to read settings: %v", err)
	}
	defer file.Close()

	// Read the file contents
	bytes, err := io.ReadAll(file)
	if err != nil {
		errorLog.Fatalf("Failed to read file: %v", err)
	}

	if err := json.Unmarshal(bytes, settings); err != nil {
		errorLog.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	
	if op == "recv" { // answer
		settings.Operation = 1
	} else if op == "send" { 
		settings.Operation = 0 // offer
	} else {
		errorLog.Fatalf("Expected 'recv' or 'send', got %s", op)
	}

	log.Print("Opening connection")
	conn, err := dc.FromSettings(settings)
	defer conn.CloseAll()

	log.Print("Opened")

	if err != nil {
		errorLog.Fatalf("Error initializing the connection: %v", err)
	}
	
	if settings.Operation == 1 {
		err = Receive(conn, target)
	} else {
		err = Send(conn, target)
	}

	if err != nil {
		errorLog.Fatalf("Error while IO: %v", err)
	}
}

func Receive(c *dc.Connection, basePath string) error {
	info := new(atf.FileInfo)
	err := json.Unmarshal(<-c.Out, info)
	if err != nil {
		return err
	}

	path := atf.PathJoin([]string{basePath, info.Name})
	log.Printf("Writing file to %s", path)

	size := info.Size

	log.Printf("Received %5d bytes", size)

	var file *os.File
	var tarPath string
	if info.IsDir {
		tarPath = atf.GetTmpName([]string{info.Name+".tar"})
		log.Printf("Created tmp tar in %s", tarPath)
		file, err = os.Create(tarPath)
	} else {
		file, err = os.Create(path)
	}

	defer file.Close()

	if err != nil {
		return err
	}
	
	received := 0
	for chunk := range c.Out {
		received += len(chunk)
		log.Printf("Receiving chunk %4d/%4d bytes", received, size)
		_, err := file.Write(chunk)
		if err != nil {
			c.In <- []byte("KO")
			return err
		}

		if int64(received) == size {
			break
		}
	}

	if info.IsDir {
		log.Printf("Extracting tar to %s", path)
		proc := exec.Command("tar", "-xf", tarPath, "-C", basePath)
		if err := proc.Run(); err != nil {
			return err
		}
	}

	c.In <- []byte("ACK")
	log.Printf("Sent ACK")
	return nil
}


func Send(c *dc.Connection, path string) error {
	osInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	
	info := atf.CloneInfo(osInfo)
	
	var file *os.File
	if info.IsDir {
		// directory:
		// keep original information, but info.Size is the size of the
		// tar file
		log.Print("Directory detected")

		tarPath := atf.GetTmpName([]string{info.Name+".tar"})
		proc := exec.Command("tar", "-cf", tarPath, path)
		if err := proc.Run(); err != nil {
			return err
		}

		file, err = os.Open(tarPath)
		if err != nil {
			return err
		}
		finfo, err := os.Stat(tarPath)
		if err != nil {
			return err
		}
		info.Size = finfo.Size()
	} else {
		file, err = os.Open(path)
	}

	defer file.Close()	
	if err != nil {
		return err
	}

	log.Printf("total bytes: %d", info.Size)
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return err
	}

	c.In <- infoBytes
	buf := make([]byte, 1024)	
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		log.Printf("Sent slice of %5d bytes", n)
		c.In <- slices.Clone(buf[:n])
	}

	if res := string(<-c.Out); res != "ACK" {
		return fmt.Errorf("Expected ACK, got %s", res)
	}
	log.Printf("Received ACK")
	return nil
}
