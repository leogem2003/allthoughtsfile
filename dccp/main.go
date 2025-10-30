package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"io"
	"log"
	"slices"
	"strconv"

	dc "github.com/leogem2003/directchan"
)

func main() {
	flag.Parse()
	op := flag.Arg(0)
	target := flag.Arg(1)
	settings := new(dc.ConnectionSettings) 

	file, err := os.Open("settings.json")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Read the file contents
	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	if err := json.Unmarshal(bytes, settings); err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	
	if op == "recv" { // answer
		settings.Operation = 1
	} else if op == "send" { 
		settings.Operation = 0 // offer
	} else {
		log.Fatalf("Expected 'recv' or 'send', got %s", op)
	}

	log.Print("Opening connection")
	conn, err := dc.FromSettings(settings)
	// defer conn.CloseAll()
	log.Print("Opened")

	if err != nil {
		log.Fatalf("Error initializing the connection: %v", err)
	}
	
	if settings.Operation == 1 {
		err = Receive(conn, target)
	} else {
		err = Send(conn, target)
	}

	if err != nil {
		log.Fatalf("Error while IO: %v", err)
	}
}

func Receive(c *dc.Connection, path string) error {
	size, err := strconv.Atoi(string(<-c.Out))
	if err != nil {
		return err
	}

	log.Printf("receiving %d bytes", size)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	received := 0
	for chunk := range c.Out {
		received += len(chunk)
		log.Printf("Receiving chunk %4d/%4d bytes", received, size)
		_, err := file.Write(chunk)
		if err != nil {
			c.In <- []byte("KO")
			return err
		}

		if received == size {
			break
		}
	}
	
	c.In <- []byte("ACK")
	log.Printf("Sent ACK")
	return nil
}


func Send(c *dc.Connection, path string) error {
	log.Print("sending bytes... ")	
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	
	log.Printf("total bytes: %d", info.Size())
	c.In <- []byte(strconv.Itoa(int(info.Size())))
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
