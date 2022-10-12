default: build

build: build-agent build-node-image
	go build -o bin/main main.go

build-node-image: 
	sudo docker build -t own-kube-node .

build-agent: 
	go build -o bin/agent pkg/agent/agent.go