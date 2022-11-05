# Dockerfile for node image
FROM ubuntu

WORKDIR /agent

RUN apt-get update && apt-get install -y wget containerd iproute2 iptables iputils-ping

COPY bin/agent .

EXPOSE 10250

ENTRYPOINT [ "./agent" ]