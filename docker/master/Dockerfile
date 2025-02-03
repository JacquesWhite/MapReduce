FROM --platform=linux/amd64 golang:1.23.5

WORKDIR /app

COPY ./src/ ./src/

# Install protoc command
RUN apt-get update && \
    apt-get install -y protobuf-compiler

RUN /app/src/scripts/build.sh

ENV PORT=50000

CMD ["sh", "-c", "./src/bin/master --port=$PORT"]
