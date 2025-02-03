FROM --platform=linux/amd64 golang:1.23.5

WORKDIR /app

COPY ./src/ ./src/

# Install protoc command
RUN apt-get update && \
    apt-get install -y protobuf-compiler

RUN /app/src/scripts/build.sh
RUN cp src/bin/* .

ENV WORKER_ADDRESS=localhost
ENV WORKER_PORT=50000
ENV MASTER_ADDRESS=localhost
ENV MASTER_PORT=50000
ENV WORKER_PLUGIN=wordcount_plugin.so

CMD ["sh", "-c", "./src/bin/worker --w_port=$WORKER_PORT --m_port=$MASTER_PORT --m_ip=$MASTER_ADDRESS --w_ip=$WORKER_ADDRESS --plugin=/app/src/bin/$WORKER_PLUGIN"]
