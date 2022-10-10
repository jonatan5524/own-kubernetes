# Dockerfile for node image
FROM ubuntu

WORKDIR /agent

RUN apt-get update \
	&& apt-get install -y wget git gcc \
	&& wget -P /tmp https://go.dev/dl/go1.19.2.linux-amd64.tar.gz \
	&& tar -C /usr/local -xzf "/tmp/go1.19.2.linux-amd64.tar.gz"

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

RUN apt-get install -y containerd

COPY go.mod .
COPY go.sum .
RUN go mod download  

COPY . .

RUN go build -o main pkg/agent/agent.go

EXPOSE 10250

ENTRYPOINT [ "./main" ]

# build:  sudo docker build -t containerd_test .
# run: sudo docker run -it --memory="2900MB" --cpus="2" -p 10250:10250 --privileged containerd_test