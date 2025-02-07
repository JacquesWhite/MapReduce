from fastapi import FastAPI, File, UploadFile, Form, HTTPException
from fastapi.responses import FileResponse, HTMLResponse
import os
import shutil
import grpc
import master_pb2
import master_pb2_grpc
import zipfile

app = FastAPI()

# Directories and master configuration from environment variables
UPLOAD_DIRECTORY = os.getenv("SHARED_DIR", "./uploaded_files")
MASTER_ADDRESS = os.getenv("MASTER_ADDRESS", "localhost")
MASTER_PORT = os.getenv("MASTER_PORT", "50051")
PLUGIN_DIRECTORY = os.getenv("PLUGIN_DIR", "./plugins")

DATASETS_DIRECTORY = os.path.join(UPLOAD_DIRECTORY, "datasets")
RESULTS_DIRECTORY = os.path.join(UPLOAD_DIRECTORY, "results")

if not os.path.exists(DATASETS_DIRECTORY):
    os.makedirs(DATASETS_DIRECTORY)
if not os.path.exists(RESULTS_DIRECTORY):
    os.makedirs(RESULTS_DIRECTORY)
if not os.path.exists(PLUGIN_DIRECTORY):
    os.makedirs(PLUGIN_DIRECTORY)

@app.get("/", response_class=HTMLResponse)
async def main():
    with open("index.html") as f:
        return f.read()

@app.post("/upload/")
async def upload_files(subdirectory: str = Form(...), files: list[UploadFile] = File(...)):
    subdirectory_path = os.path.join(DATASETS_DIRECTORY, subdirectory)
    if not os.path.exists(subdirectory_path):
        os.makedirs(subdirectory_path)

    for file in files:
        file_location = os.path.join(subdirectory_path, file.filename)
        with open(file_location, "wb") as f:
            f.write(file.file.read())

    return {"status": "success", "message": f"Dataset uploaded successfully"}

@app.post("/upload_plugin/")
async def upload_plugin(file: UploadFile = File(...)):
    if not file.filename.endswith(".so"):
        raise HTTPException(status_code=400, detail="Only .so files are allowed")
    file_location = os.path.join(PLUGIN_DIRECTORY, file.filename)
    with open(file_location, "wb") as f:
        f.write(file.file.read())
    return {"status": "success", "message": f"Plugin '{file.filename}' successfully uploaded"}

@app.get("/plugins")
async def list_plugins():
    plugins = ['Default']
    for file in os.listdir(PLUGIN_DIRECTORY):
        if file.endswith(".so"):
            plugins.append(file)
    return {"plugins": plugins}

# A unified download endpoint that returns a file if the identifier is a file
# or zips a directory if needed.
@app.get("/download/{filetype}/{identifier}")
async def download_identifier(filetype: str, identifier: str):
    if filetype not in ["datasets", "results"]:
        raise HTTPException(status_code=404, detail="Invalid file type")
    if filetype == "datasets":
        file_path = os.path.join(DATASETS_DIRECTORY, identifier)
    else:
        file_path = os.path.join(RESULTS_DIRECTORY, identifier)
    zip_path = f"{file_path}.zip"
    # Try checking if a zip file exists with that name
    if os.path.exists(f"{file_path}.zip"):
        return FileResponse(zip_path, filename=identifier)
    elif os.path.exists(file_path):
        if os.path.isfile(file_path):
            return FileResponse(file_path, filename=identifier)
        elif os.path.isdir(file_path):
            shutil.make_archive(file_path, 'zip', file_path)
            return FileResponse(zip_path, filename=f"{identifier}.zip")
    else:
        raise HTTPException(status_code=404, detail="File or directory not found")

@app.get("/files/{filetype}")
async def list_files(filetype: str):
    if filetype == "datasets":
        return next(os.walk(DATASETS_DIRECTORY))[1]
    elif filetype == "results":
        return next(os.walk(RESULTS_DIRECTORY))[1]
    raise HTTPException(status_code=404, detail="Invalid file type")


@app.post("/mapreduce/")
async def mapreduce(
    subdirectory: str = Form(...),
    num_partitions: int = Form(...),
    plugin: str = Form(...)
):
    dataset_path = os.path.abspath(os.path.join(DATASETS_DIRECTORY, subdirectory))
    if not os.path.exists(dataset_path):
        raise HTTPException(status_code=404, detail="Subdirectory not found")

    working_dir_path = os.path.abspath(os.path.join(RESULTS_DIRECTORY, f"{subdirectory}-output"))

    working_dir = os.path.abspath(f"{working_dir_path}-{num_partitions}-partitions")

    # Call the Master.MapReduce RPC service
    try:
        with grpc.insecure_channel(f"{MASTER_ADDRESS}:{MASTER_PORT}") as channel:
            stub = master_pb2_grpc.MasterStub(channel)
            if plugin == "Default":
                request = master_pb2.MapReduceRequest(
                    input_dir=dataset_path,
                    working_dir=working_dir,
                    num_partitions=num_partitions,
                )
            else:
                request = master_pb2.MapReduceRequest(
                    input_dir=dataset_path,
                    working_dir=working_dir,
                    num_partitions=num_partitions,
                    plugin_path=os.path.join(PLUGIN_DIRECTORY, plugin),
                )
            response = stub.MapReduce(request)
    except grpc.RpcError as e:
        raise HTTPException(status_code=500, detail=f"RPC failed: {e}")

    # Compress the output files into a zip file.
    output_zip_filename = f"{subdirectory}-{num_partitions}-partitions-output.zip"
    output_zip_path = os.path.join(RESULTS_DIRECTORY, output_zip_filename)
    with zipfile.ZipFile(output_zip_path, 'w') as zipf:
        for file in response.output_files:
            zipf.write(file, os.path.basename(file))

    return {
        "status": "success",
        "message": "MapReduce job completed successfully",
        "download_url": f"/download/results/{output_zip_filename}"
    }
