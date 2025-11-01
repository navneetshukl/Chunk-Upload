import React, { useState } from 'react';

const uploadFileInChunks = async (file, uploadUrl, onProgress) => {
  const chunkSize = 500; // 1MB
  const totalChunks = Math.ceil(file.size / chunkSize);

  for (let i = 0; i < totalChunks; i++) {
    const start = i * chunkSize;
    const end = Math.min(file.size, start + chunkSize);
    const chunkBlob = file.slice(start, end);

    const chunkFile = new File([chunkBlob], file.name, {
      type: file.type || 'application/octet-stream',
      lastModified: file.lastModified,
    });

    const formData = new FormData();
    formData.append('chunk', chunkFile);
    formData.append('index', i);
    formData.append('totalChunks', totalChunks);
    formData.append('fileName', file.name); // ✅ correct field

    const response = await fetch(uploadUrl, {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      throw new Error(await response.text());
    }

    onProgress(i + 1, totalChunks);
  }
};

const UploadComponent = () => {
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState('Choose file');
  const [error, setError] = useState('');

  const handleFile = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setProgress(0);
    setStatus("Uploading...");
    setError("");

    try {
      await uploadFileInChunks(
        file,
        "http://localhost:8080/upload",
        (current, total) => {
          const percent = Math.round((current / total) * 100);
          setProgress(percent);
          setStatus(`Uploading chunk ${current}/${total}`);
        }
      );
      setStatus("✅ Upload complete");
    } catch (err) {
      setError(err.message);
      setStatus("❌ Upload failed");
    }
  };

  return (
    <div style={{ padding: 20 }}>
      <h2>Chunk Upload</h2>
      <input type="file" onChange={handleFile} />

      <div style={{ width: '100%', background: '#ddd', height: 25, marginTop: 10 }}>
        <div style={{
          width: `${progress}%`,
          background: progress === 100 ? "green" : "blue",
          height: "100%",
          color: "white",
          textAlign: "center"
        }}>
          {progress}%
        </div>
      </div>

      <p>{status}</p>
      {error && <p style={{ color: "red" }}>{error}</p>}
    </div>
  );
};

export default UploadComponent;
