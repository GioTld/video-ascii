package decoder

import "time"

type FileInfo struct {
	Path    string    `json:"path"`
	Format  string    `json:"format"`
	AddedAt time.Time `json:"added_at"`
	isDir   bool
}

type File struct {
	Name string   `json:"name"`
	Size int64    `json:"size"`
	Info FileInfo `json:"info"`
}
