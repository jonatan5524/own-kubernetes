default: build push run

TARGETS_DIR = ./cmd/services

build: $(TARGETS_DIR)/*
	  for folder in $^ ; do \
			echo "building image from $${folder}"; \
      docker build -t jonatan5524/own-kubernetes:$$(basename $${folder}) --build-arg dir=$${folder} --build-arg target=$$(basename $${folder}) .; \
    done
		go build -o ./bin/own-kubectl ./cmd/own-kubectl
		env GOOS=linux GOARCH=amd64 go build -o ./bin/kubelet ./cmd/kubelet

push: $(TARGETS_DIR)/*
		for folder in $^ ; do \
			echo "pushing image to jonatan5524/own-kubernetes:$$(basename $${folder})"; \
			docker push jonatan5524/own-kubernetes:$$(basename $${folder}); \
    done

# run: 
# 	docker network create bridge-kube || true
# 	docker run -p 2379:2379 -p 4001:4001 --network bridge-kube -d --name etcd quay.io/coreos/etcd:v3.5.15 /usr/local/bin/etcd -advertise-client-urls http://0.0.0.0:2397,http://0.0.0.0:4001 -listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 -enable-grpc-gateway -enable-v2 -log-level=debug
# 	docker run --name kube-api --network bridge-kube -p 8080:8080 -e ETCD_ENDPOINT=etcd -d jonatan5524/own-kubernetes:kube-api
	 	
# 	./bin/kubelet &

stop: 
	docker stop etcd kube-api
	pkill kubelet || true

rm:
	docker rm etcd kube-api
	rm -rf ./bin/*

start-node:
	qemu-system-x86_64 -hda ./vms/debian-node.img -m 4000 -accel kvm \
		-smp 2,sockets=1,cores=2,threads=1 -netdev bridge,id=hn0,br=virbr0 -device virtio-net-pci,netdev=hn0,id=nic1,mac=DE:AD:BE:EF:E0:00 &
	sleep 30
	scp ./bin/kubelet user@$$(arp | grep -i DE:AD:BE:EF:E0:00 | awk '{print $$1}'):/home/user/kubelet
	scp ./system-manifest/* user@$$(arp | grep -i DE:AD:BE:EF:E0:00 | awk '{print $$1}'):/home/user/kubernetes/manifests

stop-node:
	kill `ps ax | grep "qemu-system-x86_64" | awk 'NR==1{print $1}'`
	cp ./vms/debian-node.img ./vms/debian-node-$$(date +%Y-%m-%d_%H-%M-%S).img.bak
	cp ./vms/debian-node-clean.img ./vms/debian-node.img 