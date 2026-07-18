package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	atf "github.com/leogem2003/allthoughtsfiles"
	dc "github.com/leogem2003/directchan"
)

var errorLog = log.New(os.Stderr, "ERROR: ", 0)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s: %s (send|recv) <file>\n", os.Args[0], os.Args[0])
	flag.PrintDefaults()
}

var ABORT = []byte("ERR")
var EOF = []byte("EOF")
var OK = []byte("OK")

func main() {
	var settingsPath string
	var debug bool
	var aes string
	var stream bool
	var extra string

	atf.SettingsFlag(&settingsPath)
	atf.DebugFlag(&debug)
	atf.AESFlag(&aes)
	flag.BoolVar(&stream, "stream", false, "enable tar compression streaming")
	flag.StringVar(&extra, "exclude", "", "exclude patterns: use ; as separator")
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

	if op != "send" && op != "recv" {
		errorLog.Fatalf("Expected 'recv' or 'send', got %s", op)
	}

	log.Print("Opening connection")
	conn, err := dc.FromSettings(settings)
	defer conn.CloseAll()

	var channel dc.IOChannel
	if aes != "" {
		file, err := os.Open(aes)
		if err != nil {
			errorLog.Fatalf("Failed to open file: %v", err)
		}
		key, err := io.ReadAll(file)
		if err != nil {
			errorLog.Fatalf("Failed to read key file: %v", err)
		}
		file.Close()
		cypher, err := dc.NewAESGCM(key)
		if err != nil {
			errorLog.Fatalf("Error while creating AES cypher: %v", err)
		}
		channel = dc.NewAESConnection(conn, cypher)
	} else {
		channel = conn
	}

	go func() {
		for {
			log.Printf("state changed: %v \n", <-conn.State)
		}
	}()

	log.Println("Opened")

	if err != nil {
		errorLog.Fatalf("Error initializing the connection: %v", err)
	}

	if op == "recv" {
		err = Receive(channel, target, stream)
	} else {
		err = Send(channel, target, stream, extra)
	}

	if err != nil {
		errorLog.Fatalf("%v", err)
	}

	log.Printf("Exiting")
}

func Receive(c dc.IOChannel, basePath string, stream bool) error {
	info := new(atf.FileInfo)
	err := json.Unmarshal(c.Recv(), info)
	if err != nil {
		return err
	}

	path := filepath.Join(basePath, info.Name)
	log.Printf("Writing file to %s", path)

	size := info.Size
	log.Printf("Size: %d", size)
	if size < 0 {
		if !stream {
			return fmt.Errorf("Received unexpected stream")
		}
		proc := exec.Command("tar", "-xzf", "-", "-C", basePath)
		stdin, err := proc.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed creating stdin pipe: %v", err)
		}
		if err := proc.Start(); err != nil {
			return fmt.Errorf("Failure while starting tar: %v", err)
		}
		
		tot := 0
		for {
			status := c.Recv()
			if slices.Equal(status, ABORT) {
				return fmt.Errorf("Operation aborted by the other party")
			}

			buf := c.Recv()
			tot += len(buf)
			log.Printf("received %d bytes (total %d)", len(buf), tot)
			_, err := stdin.Write(buf)
			if err != nil {
				return err
			}

			if slices.Equal(status, EOF) {
				stdin.Close()
				break
			} else if slices.Equal(status, ABORT) {
				stdin.Close()
				log.Fatalf("Operation aborted by remote party")
			}
		}
	} else {
		var file *os.File
		var tarPath string
		if info.IsDir {
			tarPath = atf.GetTmpName([]string{info.Name + ".tar"})
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
		for {
			chunk := c.Recv()
			log.Printf("Received chunk of %3d bytes", len(chunk))
			received += len(chunk)

			_, err := file.Write(chunk)
			if err != nil {
				c.Send([]byte("KO"))
				return err
			}

			if int64(received) == size {
				break
			}
		}

		if info.IsDir {
			log.Printf("Extracting tar to %s", path)
			proc := exec.Command("tar", "-xzf", tarPath, "-C", basePath)
			if err := proc.Run(); err != nil {
				return err
			}
		}
	}
	c.Send([]byte("ACK"))
	log.Printf("Sent ACK")
	return nil
}

func Send(c dc.IOChannel, path string, stream bool, exclude string) error {
	osInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	info := atf.CloneInfo(osInfo)
	tarCmd := []string{"-czf"}
	tarPath := atf.GetTmpName([]string{info.Name + ".tar"})

	if stream {
		tarCmd = append(tarCmd, "-")
	} else {
		tarCmd = append(tarCmd, tarPath)
	}

	if exclude != "" {
		for e := range strings.SplitSeq(exclude, ";") {
			tarCmd = append(tarCmd, "--exclude", e)
		}
	}

	tarCmd = append(tarCmd, path)

	var file *os.File
	var proc *exec.Cmd
	if stream {
		log.Printf("streaming, cmd: tar %s", strings.Join(tarCmd, " "))
		info.Size = -1 // signal stream
		proc = exec.Command("tar", tarCmd...)
	} else if info.IsDir {
		// directory:
		// keep original information, but info.Size is the size of the
		// tar file
		log.Print("Directory detected")
		// create tar file
		proc = exec.Command("tar", tarCmd...)
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
		defer file.Close()
	} else {
		file, err = os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
	}

	log.Printf("total bytes: %d", info.Size)
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("cannot serialize stats: %v", err)
	}

	c.Send(infoBytes)

	buf := make([]byte, 1024)
	if stream {
		stdout, err := proc.StdoutPipe()
		if err != nil {
			return fmt.Errorf("Failed creating stdout pipe: %v", err)
		}
		log.Println("Starting tar")
		if err := proc.Start(); err != nil {
			return fmt.Errorf("failure while starting tar: %v", err)
		}

		for {
			n, err := stdout.Read(buf)
			log.Printf("%d bytes read\n", n)
			if err != nil {
				if err == io.EOF {
					c.Send(EOF)
					log.Printf("EOF")
				} else {
					c.Send(ABORT)
					return fmt.Errorf("error while reading buffer: %v", err)
				}
			} else {
				c.Send(OK)
			}

			chunk := slices.Clone(buf[:n])
			c.Send(chunk)
			if err == io.EOF {
				break
			}
		}
		proc.Wait()
	} else {
		for {
			n, err := file.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			log.Printf("Sent slice of %5d bytes", n)
			chunk := slices.Clone(buf[:n])
			c.Send(chunk)
		}
	}

	log.Printf("Waiting for ACK")
	if res := string(c.Recv()); res != "ACK" {
		return fmt.Errorf("Expected ACK, got %s", res)
	}
	log.Printf("Received ACK")
	return nil
}
