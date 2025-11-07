package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"io"
	"log"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"time"

	dc "github.com/leogem2003/directchan"
)

const (
	TAR_COMPRESS = "tar -cf"
	TAR_EXTRACT = "tar -xf"
)

type FileInfo struct {
  Name    string      `json:"name"`
	Size    int64       `json:"size"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"mod_time"`
	IsDir   bool        `json:"is_dir"`
}

func CloneInfo(info os.FileInfo) FileInfo {
	return FileInfo {
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}
}

func PathJoin(parts []string) string {
	return strings.Join(parts, string(os.PathSeparator))
}
func GetTmpName(suffix []string) string {
	l := make([]string, len(suffix)+1, len(suffix)+1)
	l[0] = os.TempDir()
	suffix[len(suffix)-1] += strconv.Itoa(rand.Intn(1024))
	l = append(l, suffix...)
	return PathJoin(l)
}

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
	defer conn.CloseAll()

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

func Receive(c *dc.Connection, basePath string) error {
	info := new(FileInfo)
	err := json.Unmarshal(<-c.Out, info)
	if err != nil {
		return err
	}

	path := PathJoin([]string{basePath, info.Name})
	log.Printf("Writing file to %s", path)

	size := info.Size

	log.Printf("Receiving %d bytes", size)

	var file *os.File
	var tarPath string
	if info.IsDir {
		tarPath = GetTmpName([]string{info.Name+".tar"})
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
		cmd := strings.Join([]string{TAR_EXTRACT, tarPath, "-C", path}, " ")
		log.Println(cmd)
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
	
	info := CloneInfo(osInfo)
	
	var file *os.File
	if info.IsDir {
		// directory:
		// keep original information, but info.Size is the size of the
		// tar file
		log.Print("Directory detected")

		// TODO random string in the name to avoid clashes
		tarPath := GetTmpName([]string{info.Name+".tar"})
		cmd := strings.Join([]string{TAR_COMPRESS, tarPath, path}, " ")
		log.Print(cmd)
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
