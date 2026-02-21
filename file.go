package atf

import (
	"encoding/binary"
	"hash/fnv"
	"time"
	"os"
)

type FileInfo struct {
  Name     string      `json:"name"`
	Size     int64       `json:"size"`
	Mode     os.FileMode `json:"mode"`
	ModTime  time.Time   `json:"mod_time"`
	IsDir    bool        `json:"is_dir"`
}

func CloneInfo(info os.FileInfo) FileInfo {
	return FileInfo {
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime().UTC(),
		IsDir:   info.IsDir(),
	}
}

func (i FileInfo) Hash() uint64 {
    h := fnv.New64a()

    h.Write([]byte(i.Name))

    binary.Write(h, binary.LittleEndian, i.Size)
    binary.Write(h, binary.LittleEndian, int64(i.Mode))

    binary.Write(h, binary.LittleEndian, i.ModTime.UnixNano())

    if i.IsDir {
        h.Write([]byte{1})
    } else {
        h.Write([]byte{0})
    }

    return h.Sum64()
}
