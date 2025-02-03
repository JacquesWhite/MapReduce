from fastapi import FastAPI, File, UploadFile, Form, HTTPException
from fastapi.responses import FileResponse, HTMLResponse
import os
import shutil
import grpc
import master_pb2
import master_pb2_grpc
import zipfile

app = FastAPI()

UPLOAD_DIRECTORY = os.getenv("SHARED_DIR", "./uploaded_files")
MASTER_ADDRESS = os.getenv("MASTER_ADDRESS", "localhost")
MASTER_PORT = os.getenv("MASTER_PORT", "50051")

print(UPLOAD_DIRECTORY)
print(MASTER_ADDRESS)
print(MASTER_PORT)

if not os.path.exists(UPLOAD_DIRECTORY):
    os.makedirs(UPLOAD_DIRECTORY)

@app.get("/", response_class=HTMLResponse)
async def main():
    with open("index.html") as f:
        return f.read()

@app.post("/upload/")
async def upload_files(subdirectory: str = Form(...), files: list[UploadFile] = File(...)):
    subdirectory_path = os.path.join(UPLOAD_DIRECTORY, subdirectory)
    if not os.path.exists(subdirectory_path):
        os.makedirs(subdirectory_path)

    for file in files:
        file_location = os.path.join(subdirectory_path, file.filename)
        with open(file_location, "wb") as f:
            f.write(file.file.read())

    return {"info": f"Files saved in '{subdirectory_path}'"}

@app.get("/download/{subdirectory}")
async def download_subdirectory(subdirectory: str):
    subdirectory_path = os.path.join(UPLOAD_DIRECTORY, subdirectory)
    if not os.path.exists(subdirectory_path):
        return {"error": "Subdirectory not found"}

    zip_path = f"{subdirectory_path}.zip"
    shutil.make_archive(subdirectory_path, 'zip', subdirectory_path)

    return FileResponse(zip_path, filename=f"{subdirectory}.zip")

@app.get("/files")
async def list_files():
    subdirectories = []
    for subdir, dirs, _ in os.walk(UPLOAD_DIRECTORY):
        for dir in dirs:
            subdirectory = os.path.relpath(os.path.join(subdir, dir), UPLOAD_DIRECTORY)
            subdirectories.append(subdirectory)
    return subdirectories

@app.post("/mapreduce/")
async def mapreduce(subdirectory: str = Form(...), num_partitions: int = Form(...)):
    subdirectory_path = os.path.abspath(os.path.join(UPLOAD_DIRECTORY, subdirectory))
    if not os.path.exists(subdirectory_path):
        raise HTTPException(status_code=404, detail="Subdirectory not found")

    working_dir = os.path.abspath(f"{subdirectory_path}-work")

    # Call the Master.MapReduce RPC service
    try:
        with grpc.insecure_channel(f"{MASTER_ADDRESS}:{MASTER_PORT}") as channel:
            stub = master_pb2_grpc.MasterStub(channel)
            request = master_pb2.MapReduceRequest(
                input_dir=subdirectory_path,
                working_dir=working_dir,
                num_partitions=num_partitions
            )
            response = stub.MapReduce(request)
    except grpc.RpcError as e:
        raise HTTPException(status_code=500, detail=f"RPC failed: {e}")

    # Compress the output files to a zip
    output_files = response.output_files
    print(f"Output files: {output_files}")
    output_zip_path = os.path.join(UPLOAD_DIRECTORY, f"{subdirectory}_output.zip")
    with zipfile.ZipFile(output_zip_path, 'w') as zipf:
        for file in output_files:
            zipf.write(file, os.path.basename(file))

    return FileResponse(output_zip_path, filename=f"{subdirectory}_output.zip")