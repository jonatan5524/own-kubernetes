default: build push run

TARGETS_DIR = ./cmd

build: $(TARGETS_DIR)/*
	  for folder in $^ ; do \
			echo "building image from $${folder}"; \
      docker build -t jonatan5524/own-kubernetes:$$(basename $${folder}) --build-arg dir=$${folder} --build-arg target=$$(basename $${folder}) .; \
    done

push: $(TARGETS_DIR)/*
		for folder in $^ ; do \
			echo "pushing image to jonatan5524/own-kubernetes:$$(basename $${folder})"; \
			docker push jonatan5524/own-kubernetes:$$(basename $${folder}); \
    done

run: 
	docker network create bridge-kube || true
	docker run -p 2379:2379 -p 4001:4001 --network bridge-kube -d --name etcd quay.io/coreos/etcd:v3.5.15 /usr/local/bin/etcd -advertise-client-urls http://0.0.0.0:2397,http://0.0.0.0:4001 -listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 -enable-grpc-gateway -enable-v2 -log-level=debug
	docker run --name kube-api --network bridge-kube -p 8080:8080 -e ETCD_ENDPOINT=etcd -d jonatan5524/own-kubernetes:kube-api

stop:
	docker stop etcd kube-api

rm:
	docker rm etcd kube-api