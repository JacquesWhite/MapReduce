FROM --platform=linux/amd64 python:3.10-slim

WORKDIR /app

COPY requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt

COPY . .

RUN ./build_proto.sh

EXPOSE 8000

CMD ["uvicorn", "upload:app", "--host", "0.0.0.0", "--port", "8000"]