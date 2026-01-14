package tool

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

type FSRequest struct {
	Op   string `json:"op"`
	Path string `json:"path"`
	Data string `json:"data,omitempty"`
	Dest string `json:"dest,omitempty"`
}

type FSResult struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type entry struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	Mode    string    `json:"mode"`
	ModTime time.Time `json:"mod_time"`
}

type statInfo struct {
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	Mode    string    `json:"mode"`
	ModTime time.Time `json:"mod_time"`
}

func FS(req FSRequest) FSResult {
	switch req.Op {
	case "read":
		return readFile(req.Path)
	case "write":
		return writeFile(req.Path, req.Data)
	case "append":
		return appendFile(req.Path, req.Data)
	case "delete":
		return removeAll(req.Path)
	case "mkdir":
		return makeDir(req.Path)
	case "rmdir":
		return removeDir(req.Path)
	case "list":
		return listDir(req.Path)
	case "stat":
		return statPath(req.Path)
	case "move":
		return movePath(req.Path, req.Dest)
	case "copy":
		return copyPath(req.Path, req.Dest)
	default:
		return FSResult{OK: false, Error: "unknown op"}
	}
}

func readFile(path string) FSResult {
	if path == "" {
		return FSResult{OK: false, Error: "path is required"}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	return FSResult{OK: true, Data: string(b)}
}

func writeFile(path, data string) FSResult {
	if path == "" {
		return FSResult{OK: false, Error: "path is required"}
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	return FSResult{OK: true}
}

func appendFile(path, data string) FSResult {
	if path == "" {
		return FSResult{OK: false, Error: "path is required"}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	defer f.Close()
	if _, err := f.WriteString(data); err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	return FSResult{OK: true}
}

func removeAll(path string) FSResult {
	if path == "" {
		return FSResult{OK: false, Error: "path is required"}
	}
	if err := os.RemoveAll(path); err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	return FSResult{OK: true}
}

func makeDir(path string) FSResult {
	if path == "" {
		return FSResult{OK: false, Error: "path is required"}
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	return FSResult{OK: true}
}

func removeDir(path string) FSResult {
	if path == "" {
		return FSResult{OK: false, Error: "path is required"}
	}
	if err := os.Remove(path); err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	return FSResult{OK: true}
}

func listDir(path string) FSResult {
	if path == "" {
		path = "."
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	out := make([]entry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return FSResult{OK: false, Error: err.Error()}
		}
		out = append(out, entry{
			Name:    e.Name(),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime(),
		})
	}
	return FSResult{OK: true, Data: out}
}

func statPath(path string) FSResult {
	if path == "" {
		return FSResult{OK: false, Error: "path is required"}
	}
	info, err := os.Stat(path)
	if err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	return FSResult{OK: true, Data: statInfo{
		Path:    path,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		Mode:    info.Mode().String(),
		ModTime: info.ModTime(),
	}}
}

func movePath(src, dst string) FSResult {
	if src == "" || dst == "" {
		return FSResult{OK: false, Error: "path and dest are required"}
	}
	if err := os.Rename(src, dst); err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	return FSResult{OK: true}
}

func copyPath(src, dst string) FSResult {
	if src == "" || dst == "" {
		return FSResult{OK: false, Error: "path and dest are required"}
	}
	info, err := os.Stat(src)
	if err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	if info.IsDir() {
		return FSResult{OK: false, Error: "copy supports files only"}
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	in, err := os.Open(src)
	if err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return FSResult{OK: false, Error: err.Error()}
	}
	if err := out.Sync(); err != nil && !errors.Is(err, os.ErrInvalid) {
		return FSResult{OK: false, Error: err.Error()}
	}
	return FSResult{OK: true}
}
