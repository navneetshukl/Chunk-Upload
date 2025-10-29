const uploadFileInChunks=async(file,uploadUrl,onProgress)=>{
    if(!file) throw new Error("No file provided");
    const CHUNK_SIZE=1*1024*1024; // 1 MB per chunk
    const totalChunks=Math.ceil(file.size/CHUNK_SIZE);
    console.log("Total file chunks size is ",totalChunks)
     let uploadedBytes=0;
     for(let i=0;i<totalChunks;i++){
        const start=i*CHUNK_SIZE;
        const end=Math.min(start+CHUNK_SIZE,file.size);
        const chunk=file.slice(start,end);

        // Create form data (or send raw bytes if your backend expects that)

        const formData=new FormData();
        formData.append=("chunk",chunk);
        formData.append("index",i);
        formData.append("totalChunks",totalChunks);
        formData.append("fileName",file.name);

        // upload the chunk
        const res=await fetch(uploadUrl,{
            method:"POST",
            body:formData,
        });

        if(!res.ok){
            throw new Error(`Chunk ${i+1} upload failed`);
        }

        //update progress

        uploadedBytes+=chunk.size;
        if(onProgress){
            const percent=Math.round((uploadedBytes/file.size)*100);
            onProgress(percent);
        }
     }


}

export default uploadFileInChunks;