# Chunked File Upload

A high-performance file upload system with chunked transfer support. This project implements a React frontend with a Go backend to handle large file uploads by splitting them into manageable chunks, with real-time progress tracking and built-in safety mechanisms.

## ✨ Features

- 🔄 **Chunked Upload** - Splits large files into 500-byte chunks for reliable transfer
- 📊 **Progress Tracking** - Real-time progress bar showing upload percentage and chunk status
- 🔒 **Thread-Safe** - Per-file mutex locks prevent race conditions during concurrent uploads
- 🌐 **CORS Support** - Configured for cross-origin requests from frontend dev server
- ⚠️ **Error Handling** - Comprehensive error messages for both client and server
- ⚡ **Atomic Operations** - Final chunk triggers atomic file rename for data integrity
- 📡 **JSON API** - RESTful responses with detailed status information

## 📁 Project Structure

```
chunked-upload/
├── server.go                 # Go backend server
├── UploadComponent.jsx        # React upload component
├── uploads/                  # Directory for storing uploaded files
└── README.md
```

## 📋 Prerequisites

- **Node.js** 16+ (for React frontend)
- **Go** 1.16+ (for backend server)
- **npm** or **yarn** (for frontend package management)

## 🚀 Quick Start

### Backend Setup

```bash
# Clone or navigate to your project
cd chunked-upload

# Run the Go server
go run server.go
```

The server will start on `http://localhost:8080`

### Frontend Setup

```bash
# Create or navigate to your React project
npx create-react-app frontend
cd frontend
npm start
```

The frontend dev server will run on `http://localhost:5173`

## ⚙️ Configuration

### Backend (server.go)

Edit these constants to customize behavior:

```go
const (
    UploadDir     = "./uploads"              // Directory for storing files
    MaxMemory     = 32 << 20                 // 32 MB for multipart parsing
    Port          = ":8080"                  // Server port
    AllowedOrigin = "http://localhost:5173"  // CORS allowed origin
)
```

### Frontend (UploadComponent.jsx)

Modify the upload URL if your backend runs on a different address:

```javascript
await uploadFileInChunks(
  file,
  "http://localhost:8080/upload",  // Change this URL as needed
  (current, total) => { ... }
);
```

Adjust chunk size (currently 500 bytes) for production:

```javascript
const chunkSize = 500; // Increase for larger chunks (e.g., 1MB = 1048576)
```

## 💻 Usage

1. **Start the Backend**:
```bash
go run server.go
```

Expected output:
```
Server listening on :8080 | origin=http://localhost:5173
```

2. **Start the Frontend**:
```bash
npm start
```

3. **Upload a File**:
   - Click the file input to select a file
   - The progress bar updates in real-time showing percentage and chunk count
   - Once complete, you'll see a ✅ Upload complete message
   - Files are stored in the `uploads/` directory with original filename

## 📡 API Documentation

### POST `/upload`

Handles chunked file upload requests.

**Request Format**:

| Field | Type | Description |
|-------|------|-------------|
| `chunk` | File | Binary chunk data |
| `index` | number | Current chunk index (0-based) |
| `totalChunks` | number | Total number of chunks |
| `fileName` | string | Original filename |

**Success Response (200 OK) - Intermediate Chunk**:
```json
{
  "status": "ok",
  "received": 5000
}
```

**Success Response (200 OK) - Final Chunk**:
```json
{
  "status": "ok",
  "done": true,
  "path": "./uploads/filename.ext"
}
```

**Error Response (400/500)**:
```json
{
  "error": "Error message describing what went wrong"
}
```

## 🔄 How It Works

### Upload Flow Diagram

```
User selects file
        ↓
Frontend chunks file (500 bytes each)
        ↓
For each chunk:
  - Create FormData with chunk metadata
  - POST to backend
  - Update progress bar
        ↓
Backend receives chunk:
  - Validate index and totalChunks
  - Acquire per-file mutex lock
  - Write to temporary .part file
  - Return progress
        ↓
Final chunk received:
  - Atomic rename from .part to final filename
  - Return success with file path
        ↓
Frontend displays success message
```

### Thread Safety Mechanism

The backend uses a per-file mutex map (`fileLocks`) to ensure only one goroutine writes to a `.part` file at a time, even if multiple uploads target the same filename. This prevents data corruption from concurrent writes.

## 🔧 Customization

### Increase Chunk Size for Production

For better performance with large files, modify the chunk size:

```javascript
// In UploadComponent.jsx
const chunkSize = 1048576; // 1 MB instead of 500 bytes
```

### Change Upload Directory

```go
// In server.go
const UploadDir = "/var/uploads" // Change to your desired path
```

### Allow Multiple Origins

```go
// In server.go - modify CORS handling
origins := map[string]bool{
    "http://localhost:5173": true,
    "https://yourdomain.com": true,
}

// Then check origin in uploadHandler
```

## ⚠️ Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| `CORS error` | Frontend origin not allowed | Update `AllowedOrigin` in server.go to match frontend URL |
| `missing index, totalChunks or fileName` | Invalid form data from frontend | Verify all required fields are sent in FormData |
| `multipart parse error` | File exceeds MaxMemory buffer | Increase `MaxMemory` constant in server.go |
| `rename failed` | Filesystem permissions issue | Check that `uploads/` directory is writable by the process |
| `cannot open part file` | Write permission denied | Ensure proper permissions: `chmod 755 uploads/` |
| `incomplete write` | Disk space or I/O error | Verify available disk space and check file descriptor limits |

## 🚀 Performance Tips

- **Chunk Size**: For production, use 1-5 MB chunks instead of 500 bytes for significantly faster uploads
- **Concurrent Uploads**: Backend supports multiple simultaneous uploads with individual per-file locks
- **Memory Management**: Larger chunks = fewer requests but higher memory usage per upload
- **Network Optimization**: Monitor connection quality; adjust chunk size accordingly
- **Server Resources**: Monitor disk space usage and set appropriate `MaxMemory` limits

## 🔐 Security Considerations

### Important Security Measures

1. **Filename Sanitization**: Add validation to prevent directory traversal attacks
   ```go
   if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
       respondError(w, http.StatusBadRequest, "invalid filename")
       return
   }
   ```

2. **File Size Limits**: Implement maximum file size validation
   ```go
   const MaxFileSize = 1 << 30 // 1 GB limit
   ```

3. **Authentication**: Add authentication middleware before `uploadHandler`

4. **Virus Scanning**: Integrate antivirus scanning before finalizing uploads

5. **CORS Restriction**: Never use wildcard (`*`) for `AllowedOrigin` in production

6. **Rate Limiting**: Implement per-IP rate limiting to prevent abuse

## 🐛 Troubleshooting

### Frontend Can't Connect to Backend

```bash
# Test if backend is running
curl -X POST http://localhost:8080/upload
# Should return 405 (Method Not Allowed) if running correctly
```

- Verify backend is running: `go run server.go`
- Check CORS configuration matches your frontend URL exactly
- Ensure no firewall is blocking port 8080
- Check browser console for CORS errors

### Uploads Fail with "Incomplete Write" Error

- Verify chunk size is consistent between frontend and backend
- Check available disk space: `df -h`
- Verify file permissions: `ls -la uploads/`
- Try uploading a smaller test file first

### Progress Bar Stuck

- Open browser DevTools → Network tab
- Verify POST requests are being sent and completing
- Check browser console for JavaScript errors
- Try uploading a smaller file to test
- Restart both frontend and backend servers

### Server Returns 500 Error

```bash
# Check server logs for detailed error messages
# Look for patterns like "cannot open part file" or "stat error"
```

- Ensure `uploads/` directory exists and is writable
- Check disk space available
- Verify process has permissions to create/modify files
- Try creating directory manually: `mkdir -p uploads`

## 📊 Testing

### Manual Testing Steps

1. Upload a small test file (< 5 MB)
2. Verify file appears in `uploads/` directory
3. Check file integrity (file size should match original)
4. Test concurrent uploads with multiple files
5. Test interrupted uploads (refresh page during upload)

### Load Testing

```bash
# Generate a test file
dd if=/dev/urandom of=testfile.bin bs=1M count=100

# Monitor upload with network throttling enabled
```

## 📚 File Structure Explanation

### server.go

- **Constants**: Defines upload directory, memory limits, port, and CORS origin
- **File Locks**: Per-file mutex map for thread-safe concurrent uploads
- **Handlers**: Response helpers and the main upload handler
- **Validation**: Checks for required form fields and valid indices
- **File Operations**: Writes chunks to temporary file, renames on completion

### UploadComponent.jsx

- **uploadFileInChunks()**: Splits file and sends each chunk
- **UploadComponent**: React component managing UI and upload state
- **Progress Tracking**: Updates progress bar and status text
- **Error Handling**: Catches and displays upload errors

## 🤝 Contributing

Contributions are welcome! Please feel free to:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 💡 Future Enhancements

- [ ] Resume interrupted uploads
- [ ] Drag-and-drop file upload
- [ ] Multiple file uploads simultaneously
- [ ] File encryption support
- [ ] Download uploaded files
- [ ] Storage management dashboard
- [ ] S3/Cloud storage backend support

## 📧 Support

For issues, questions, or suggestions:

1. Check the **Troubleshooting** section above
2. Open a GitHub issue with detailed description
3. Include error messages and steps to reproduce
4. Share your configuration details (chunk size, file size, etc.)

## 🙏 Acknowledgments

- Built with React and Go
- Inspired by modern file upload best practices
- Community feedback and contributions

---

**Made with ❤️ for developers**
