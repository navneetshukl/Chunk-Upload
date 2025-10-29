// server.go
// Simple chunk receiver that matches the frontend that sends FormData:
// - chunk        : file part (binary)
// - index        : chunk index (0-based)
// - totalChunks  : total number of chunks
// - fileName     : original file name
//
// Behavior:
//  - writes chunks sequentially by appending to uploads/<fileName>.part
//  - if index==0 it truncates any existing .part file (start fresh)
//  - when index == totalChunks-1 it renames .part -> final file
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

const UploadDir = "./uploads"
const MaxMemory = 32 << 20 // 32 MB for parsing multipart form headers

// simple per-file lock to avoid concurrent writes
var fileLocks = struct {
	sync.Mutex
	m map[string]*sync.Mutex
}{m: make(map[string]*sync.Mutex)}

func getLock(name string) *sync.Mutex {
	fileLocks.Lock()
	defer fileLocks.Unlock()
	if l, ok := fileLocks.m[name]; ok {
		return l
	}
	l := &sync.Mutex{}
	fileLocks.m[name] = l
	return l
}

func ensureUploadDir() error {
	return os.MkdirAll(UploadDir, 0o755)
}

type jsonResp map[string]interface{}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := ensureUploadDir(); err != nil {
		http.Error(w, "cannot create upload dir", http.StatusInternalServerError)
		return
	}

	// parse multipart form
	if err := r.ParseMultipartForm(MaxMemory); err != nil {
		http.Error(w, "invalid multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// read form values
	indexStr := r.FormValue("index")
	totalStr := r.FormValue("totalChunks")
	fileName := r.FormValue("fileName")

	if indexStr == "" || totalStr == "" || fileName == "" {
		http.Error(w, "missing index/totalChunks/fileName", http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 {
		http.Error(w, "invalid index", http.StatusBadRequest)
		return
	}
	totalChunks, err := strconv.Atoi(totalStr)
	if err != nil || totalChunks <= 0 {
		http.Error(w, "invalid totalChunks", http.StatusBadRequest)
		return
	}

	// get the chunk file
	file, _, err := r.FormFile("chunk")
	if err != nil {
		http.Error(w, "missing chunk file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// per-file locking
	lock := getLock(fileName)
	lock.Lock()
	defer lock.Unlock()

	partPath := filepath.Join(UploadDir, fileName+".part")

	// If it's the first chunk, truncate/create the .part file
	if index == 0 {
		// Truncate or create
		f, err := os.OpenFile(partPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			http.Error(w, "cannot create part file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// write chunk to file
		written, err := io.Copy(f, file)
		_ = f.Close()
		if err != nil {
			http.Error(w, "write error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("wrote chunk index=%d (%d bytes) to %s", index, written, partPath)
	} else {
		// Append to existing .part file
		f, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			http.Error(w, "cannot open part file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		written, err := io.Copy(f, file)
		_ = f.Close()
		if err != nil {
			http.Error(w, "append error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("appended chunk index=%d (%d bytes) to %s", index, written, partPath)
	}

	// If last chunk, rename to final file name
	if index == totalChunks-1 {
		finalPath := filepath.Join(UploadDir, fileName)
		if err := os.Rename(partPath, finalPath); err != nil {
			// if rename fails, still return success but log error
			log.Printf("rename error: %v", err)
			respondJSON(w, http.StatusOK, jsonResp{
				"status": "ok",
				"note":   fmt.Sprintf("received last chunk but rename failed: %v", err),
			})
			return
		}
		log.Printf("upload complete: %s", finalPath)
		respondJSON(w, http.StatusOK, jsonResp{
			"status": "ok",
			"done":   true,
			"path":   finalPath,
		})
		return
	}

	// Not last chunk
	// report current size
	fi, err := os.Stat(partPath)
	received := int64(0)
	if err == nil {
		received = fi.Size()
	}
	respondJSON(w, http.StatusOK, jsonResp{
		"status":   "ok",
		"received": received,
	})
}

func respondJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	if err := ensureUploadDir(); err != nil {
		log.Fatalf("cannot create upload dir: %v", err)
	}
	http.HandleFunc("/upload", uploadHandler)
	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
