// server.go
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

const (
	UploadDir     = "./uploads"
	MaxMemory     = 32 << 20 // 32 MB for multipart parsing
	Port          = ":8080"
	AllowedOrigin = "http://localhost:5173"
)

// ---------------------------------------------------------------------
// Per-file mutex map (prevents race conditions on the same file name)
// ---------------------------------------------------------------------
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

// ---------------------------------------------------------------------
// Directory helper
// ---------------------------------------------------------------------
func ensureUploadDir() error {
	err := os.MkdirAll(UploadDir, 0o755)
	if err != nil {
		log.Printf("ERROR: cannot create upload directory: %v", err)
	}
	return err
}

// ---------------------------------------------------------------------
// JSON response structs
// ---------------------------------------------------------------------
type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Status   string `json:"status"`
	Received int64  `json:"received,omitempty"`
	Done     bool   `json:"done,omitempty"`
	Path     string `json:"path,omitempty"`
	Note     string `json:"note,omitempty"`
}

// ---------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------
func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("ERROR: JSON encode failed: %v", err)
	}
}

func respondError(w http.ResponseWriter, code int, msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	log.Printf("HTTP %d | ERROR: %s", code, msg)
	respondJSON(w, code, ErrorResponse{Error: msg})
}

func respondSuccess(w http.ResponseWriter, data SuccessResponse) {
	log.Printf("HTTP 200 | SUCCESS: received=%d bytes | done=%v", data.Received, data.Done)
	respondJSON(w, http.StatusOK, data)
}

// ---------------------------------------------------------------------
// Main handler
// ---------------------------------------------------------------------
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// ----- CORS -----
	w.Header().Set("Access-Control-Allow-Origin", AllowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "only POST allowed")
		return
	}

	// ----- Init upload dir -----
	if err := ensureUploadDir(); err != nil {
		respondError(w, http.StatusInternalServerError, "cannot initialise upload directory")
		return
	}

	// ----- Parse multipart -----
	if err := r.ParseMultipartForm(MaxMemory); err != nil {
		respondError(w, http.StatusBadRequest, "multipart parse error: %v", err)
		return
	}

	// ----- Form fields -----
	indexStr := r.FormValue("index")
	totalStr := r.FormValue("totalChunks")
	fileName := r.FormValue("fileName")

	fmt.Println("IndexStr ",indexStr)
	fmt.Println("TotalStr ",totalStr)
	fmt.Println("Filename ",fileName)

	if indexStr == "" || totalStr == "" || fileName == "" {
		respondError(w, http.StatusBadRequest, "missing index, totalChunks or fileName")
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 {
		respondError(w, http.StatusBadRequest, "invalid index")
		return
	}
	totalChunks, err := strconv.Atoi(totalStr)
	if err != nil || totalChunks <= 0 {
		respondError(w, http.StatusBadRequest, "invalid totalChunks")
		return
	}
	if index >= totalChunks {
		respondError(w, http.StatusBadRequest, "index >= totalChunks")
		return
	}

	// ----- Chunk file -----
	chunkFile, header, err := r.FormFile("chunk")
	if err != nil {
		respondError(w, http.StatusBadRequest, "missing chunk: %v", err)
		return
	}
	defer chunkFile.Close()

	chunkSize := header.Size
	log.Printf("Chunk received | idx=%d/%d | size=%d | name=%s", index+1, totalChunks, chunkSize, fileName)

	// ----- Per-file lock -----
	lock := getLock(fileName)
	lock.Lock()
	defer lock.Unlock()

	partPath := filepath.Join(UploadDir, fileName+".part")
	finalPath := filepath.Join(UploadDir, fileName)

	// ----- Open part file (truncate on first chunk) -----
	var f *os.File
	if index == 0 {
		f, err = os.OpenFile(partPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	} else {
		f, err = os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "cannot open part file: %v", err)
		return
	}
	defer f.Close()

	// ----- **FIXED** copy: destination = file, source = chunkFile -----
	written, err := io.Copy(f, chunkFile) // <-- correct signature
	if err != nil {
		respondError(w, http.StatusInternalServerError, "write error: %v", err)
		return
	}
	if written != chunkSize {
		respondError(w, http.StatusInternalServerError,
			"incomplete write: expected %d, wrote %d", chunkSize, written)
		return
	}
	log.Printf("Wrote chunk %d (%d bytes) -> %s", index, written, partPath)

	// ----- Final chunk? -----
	if index == totalChunks-1 {
		if err := os.Rename(partPath, finalPath); err != nil {
			log.Printf("WARN: rename failed %s -> %s: %v", partPath, finalPath, err)
			respondSuccess(w, SuccessResponse{
				Status: "ok",
				Done:   true,
				Path:   finalPath,
				Note:   fmt.Sprintf("rename failed: %v", err),
			})
			return
		}
		log.Printf("Upload finished: %s (%d chunks)", finalPath, totalChunks)
		respondSuccess(w, SuccessResponse{
			Status: "ok",
			Done:   true,
			Path:   finalPath,
		})
		return
	}

	// ----- Intermediate progress -----
	fi, err := os.Stat(partPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "stat error after write: %v", err)
		return
	}
	respondSuccess(w, SuccessResponse{
		Status:   "ok",
		Received: fi.Size(),
	})
}

// ---------------------------------------------------------------------
// Server entry point
// ---------------------------------------------------------------------
func main() {
	if err := ensureUploadDir(); err != nil {
		log.Fatalf("FATAL: upload dir: %v", err)
	}
	http.HandleFunc("/upload", uploadHandler)
	log.Printf("Server listening on %s | origin=%s", Port, AllowedOrigin)
	log.Fatal(http.ListenAndServe(Port, nil))
}