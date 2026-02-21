package main

import (
	"io"
	"encoding/json"
	"fmt"
	"flag"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	atf "github.com/leogem2003/allthoughtsfiles"
	dc "github.com/leogem2003/directchan"
)

const DBNAME = ".allthoughtsfile"
const CHUNK_SIZE = 1024
var ACK = []byte(":ACK")
var Usage = func() {
	fmt.Printf("Usage: %s [OPTIONS] <dir>\nSynchronizes a directory across devices\n", os.Args[0]) 
	flag.PrintDefaults()
}	

var errorLog = log.New(os.Stderr, "ERROR: ", 0)
var SendLock = make(chan bool, 1)

func main() {
	var settingsPath string
	var debug bool
	var create bool
	
	atf.SettingsFlag(&settingsPath)
	atf.DebugFlag(&debug)
	flag.BoolVar(&create, "create", false, "create a new db")

	flag.Usage = Usage
	flag.Parse()

	dir := flag.Arg(0)
	log.Printf("Creating stats...")
	excludeDB := atf.MakeIgnoreSuffix(DBNAME)

	policy := func(p string) bool {
		return excludeDB(p) && p != dir
	}

	file, err := os.Open(settingsPath)
	if err != nil {
		errorLog.Fatalf("Failed to read settings: %v", err)
	}

	bytes, err := io.ReadAll(file)
	if err != nil {
		errorLog.Fatalf("Failed to read file: %v", err)
	}
	file.Close()
	
	var settings dc.ConnectionSettings
	if err := json.Unmarshal(bytes, &settings); err != nil {
		errorLog.Fatalf("Invalid settings: %v", err)
	}
	log.Printf("Opening connection...")
	conn, err := dc.FromSettings(&settings)
	if err != nil {
		errorLog.Fatalf("Failed to open connection: %v", err)
	}

	defer conn.CloseAll()

	go StateLog(conn)

	statsFileName := GetStatsDB(dir)
	oldStats, err := LoadStats(dir)
	if err != nil {
		if os.IsNotExist(err) { // new DB: empty stats
			log.Println("Creating new folder database")
			if err := os.WriteFile(statsFileName, []byte{}, 0644); err != nil {
				errorLog.Fatalf("Failed creating stats file: %v", err)
			}
		} else {
			errorLog.Fatalf("Cannot load new stats: %v", err)
		}
	}

	newStats, err := atf.CreateStats(dir, policy)
	if err != nil {
		errorLog.Fatalf("Cannot create new stats: %v", err)
	}
	
	added := atf.RemovePrefix(dir, atf.StatsKeyDiff(newStats, oldStats))
	deleted := atf.RemovePrefix(dir, atf.StatsKeyDiff(oldStats, newStats))
	modified := atf.RemovePrefix(dir, atf.StatsValueDiff(newStats, oldStats))
	
	updater := make(chan []string, 3)
	go SendUpdates(conn, added, deleted, modified)
	go RecvUpdates(conn, updater)
	remoteAdded := <- updater
	remoteDeleted := <- updater
	remoteModified := <- updater
	<- SendLock
	
	log.Printf("sent: %#v %#v %#v\n", added, deleted, modified)
	log.Printf("received: %#v %#v %#v\n", remoteAdded, remoteDeleted, remoteModified)
	
	changed := append(added, modified...)
	remoteChanged := append(remoteAdded, remoteModified...)
	toRequest, err := LatestModSolver(conn, newStats, changed, remoteChanged)
	if err != nil {
		errorLog.Fatalf("Error while resolving + conflicts: %v", err)
	}

	toDelete, err := LatestModSolver(conn, newStats, modified, remoteDeleted) // only conflict possible: modified locally deleted remotely
	log.Printf("To download: %#v", toRequest)
	log.Printf("To delete: %#v", toDelete)
	
	if err := deleteFiles(dir, toDelete); err != nil {
		log.Fatalf("Error while deleting files: %v", err)
	}
	
	closeChannel := make(chan bool, 1)
	proxy1, proxy2 := dc.DualDispatch(conn, closeChannel)	
	errChannel := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	if conn.Offer {
		go SendFiles(proxy1, newStats, dir, &wg, errChannel)
		go DownloadFiles(proxy2, newStats, dir, toRequest, &wg, errChannel)
	} else {
		go DownloadFiles(proxy1, newStats, dir, toRequest, &wg, errChannel)
		go SendFiles(proxy2, newStats, dir, &wg, errChannel)
	}
	go func() {
		err := <- errChannel
		errorLog.Fatalf("Error: %v", err)
	}()

	wg.Wait()
	closeChannel <- true
	
	statsBytes, err := json.Marshal(newStats)
	if err != nil {
		log.Fatalf("Error: cannot serialize new stats: %v", err)
	}

	if err := os.WriteFile(statsFileName, statsBytes, 0644); err != nil {
		errorLog.Fatalf("Failed writing stats file: %v", err)
	}
	
	log.Printf("Closing")
}

func GetStatsDB(dir string) string {
	return atf.PathJoin([]string{dir, DBNAME})
}

func LoadStats(dir string) (atf.Stats, error) {
	statsPath := GetStatsDB(dir)
	file, err := os.Open(statsPath)
	defer file.Close()
	if err != nil {
		return make(atf.Stats), err
	}

	bytes, err := io.ReadAll(file)
	if err != nil {
		return make(atf.Stats), err
	}

	return atf.StatsFromJSON(bytes)
}

func PathsToByte(paths []string) []byte {
	return []byte(strings.Join(paths, ";"))
}

func PathsFromByte(b []byte) []string {
	if len(b)==0 {
		return []string{}
	}
	return strings.Split(string(b), ";")
}

func SendUpdates(conn *dc.Connection, added, deleted, modified []string) {
	addedBin := PathsToByte(added)	
	deletedBin := PathsToByte(deleted)	
	modifiedBin := PathsToByte(modified)	
  conn.In <- addedBin
	conn.In <- deletedBin 
	conn.In <- modifiedBin
	SendLock <- true
}

func RecvUpdates(conn *dc.Connection, updater chan []string) {
	addedBin := <- conn.Out
	deletedBin := <- conn.Out
	modifiedBin := <- conn.Out
	updater <- PathsFromByte(addedBin)
	updater <- PathsFromByte(deletedBin)
	updater <- PathsFromByte(modifiedBin)
}

func LatestModSolver(
	conn *dc.Connection,
	db atf.Stats,
	local, remote []string,
) ([]string, error) {
	toPull := make([]string, 0)
	toRequest := make([]string, 0)
	for _,r := range remote {
		if slices.Contains(local, r) {
			toRequest = append(toRequest, r)
		} else {
			toPull = append(toPull, r)
		}
	}

	conn.In <- PathsToByte(toRequest)
	toSend := PathsFromByte(<-conn.Out)

	lock := make(chan bool, 1)

	go func() {
		subDB := make(atf.Stats)
		for _,k := range toSend {
			subDB[k] = db[k]
		}
		payload, _ := atf.StatsToJSON(subDB)
		conn.In <- payload
		lock <- true
	}()

	remoteDB, err := atf.StatsFromJSON(<- conn.Out)
	if err != nil {
		return toPull, err
	}
	for k,v := range remoteDB {
		if v.ModTime.UnixNano() > db[k].ModTime.UnixNano() {
			toPull = append(toPull, k)
		}
	}
	<- lock
	return toPull, nil
}

func deleteFiles(dir string, files []string) error {
	for _, f := range files {
		if err := os.Remove(filepath.Join(dir,f)); err != nil {
			return err
		}
	}
	return nil
}

func StateLog(conn *dc.Connection) {
	for {
		log.Printf("conn state changed: %v", <-conn.State)
	} 
}

func DownloadFiles(
	conn dc.IOChannel, 
	db atf.Stats, 
	dir string,
	toRequest []string,
	wg *sync.WaitGroup,
	errChannel chan error,
) {
	defer wg.Done()

	files := toRequest
	// sort paths using length so that children are always
	// downloaded after parents
	sort.Slice(files,
		func(a, b int) bool {
			return len(files[a])<len(files[b])
		},
	)
	log.Printf("DOWNLOAD: Requesting %d files \n", len(files))
	for _, filename :=  range files {
		path := filepath.Join(dir, filename)
		log.Printf("DOWNLOAD: Requesting %s\n", filename)
		conn.Send([]byte(filename))

		info := new(atf.FileInfo)
		if err := json.Unmarshal(conn.Recv(), info); err != nil {
			errChannel <- err
			return
		}
		received := 0
		if info.IsDir {
			if err := os.MkdirAll(path, info.Mode); err != nil {
				errChannel <- err
				return
			}
		} else {	
			file, err := os.Create(path)
			if err != nil {
				errChannel <- err
				file.Close()
				return
			}

			if err := os.Chmod(path, info.Mode); err != nil {
				errChannel <- err
				file.Close()
				return
			}

			for  {
				chunk := conn.Recv()
				received += len(chunk)	
				_, err := file.Write(chunk)
				if err != nil {
					file.Close()
					errChannel <- err
					return
				}
				log.Printf("DOWNLOAD:\treceived %5d/%5d",received,info.Size)
				if int64(received) == info.Size {
					break
				}
			}
			file.Close()
		}

		// update DB with local info
		FSInfo, err := os.Stat(path)
		if err != nil {
			errChannel <- err
			return
		}
		newInfo := atf.CloneInfo(FSInfo)	
		db[filename] = newInfo
	}

	conn.Send([]byte(":OK"))
	log.Printf("DOWNLOAD: finished requests")
}


func SendFiles(
	conn dc.IOChannel,
	db atf.Stats,
	dir string,
	wg *sync.WaitGroup,
	errChannel chan error,
) {
	defer wg.Done()	
	buf := make([]byte, CHUNK_SIZE)

	for {
		requested := string(conn.Recv())
		if requested == ":OK" {
			break
		}
		log.Printf("SEND: got request %s", requested)
		info, ok := db[requested]
		if !ok {
			errChannel <- fmt.Errorf("SEND: Cannot find file %s in database", requested)
			return
		}

		path := filepath.Join(dir, requested)
		infoBytes, err := json.Marshal(info)
		if err != nil {
			errChannel <- err
			return 
		}

		conn.Send(infoBytes)
		if info.IsDir {
			continue
		}

		file, err := os.Open(path)
		defer file.Close()
		if err != nil {
			errChannel <- err
			return
		}

		for {
			n, err := file.Read(buf)
			if err != nil && err != io.EOF {
				errChannel <- err
				return
			}
			chunk := slices.Clone(buf[:n])
			conn.Send(chunk)
			if err != nil { // EOF
				break
			}
		}	
	}

	log.Printf("SEND: finished requests")
}
