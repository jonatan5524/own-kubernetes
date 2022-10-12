# Dockerfile for node image
FROM ubuntu

WORKDIR /agent

RUN apt-get update && apt-get install -y wget containerd

COPY bin/agent .

EXPOSE 10250

ENTRYPOINT [ "./agent" ]