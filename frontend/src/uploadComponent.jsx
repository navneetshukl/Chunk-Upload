import React, { useState } from 'react'
import uploadFileInChunks from './chunkUploader';


const UploadComponent = () => {
    const[progress,setProgress]=useState(0);
    const[status,setStatus]=useState("");

    const handleFile=async(event)=>{
        const file=event.target.files[0];
        if(!file) return;
        setStatus("Uploading...");
        try{
            await uploadFileInChunks(file,"http://localhost:8080/upload",(p)=>setProgress(p));
            setStatus("Upload complete ✅");

        } catch(err){
            console.log("Error in uploading ",err)
            setStatus("Upload failed ❌");

        }
    };
  return (
    <div>
        <div style={{padding: 20}}>
            <input type='file' onChange={handleFile}/>
            <div>Progress : {progress}%</div>
            <div>{status}</div>
        </div>
      
    </div>
  )
}

export default UploadComponent
