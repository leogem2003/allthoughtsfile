package atf

import (
	"time"
	"os"
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

//TODO implement
func (f FileInfo) Hash() uint64 {
	panic("Not implemented")
	return 0
}
