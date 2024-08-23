package pod

import (
	"fmt"
	"log"
	"sync"

	"github.com/jonatan5524/own-kubernetes/pkg"
	"github.com/jonatan5524/own-kubernetes/pkg/etcd"
	"go.etcd.io/etcd/clientv3"
)

type etcdWatcher struct {
	watchChan     clientv3.WatchChan
	closeChanFunc func()
}

type Pod struct {
	watcher *etcdWatcher
}

func (pod *Pod) Register() error {
	log.Println("register pod handler")

	watchChan, closeChanFunc, err := etcd.GetWatchChannel(pkg.POD_ETCD_KEY)
	if err != nil {
		return err
	}

	pod.watcher = &etcdWatcher{
		watchChan:     watchChan,
		closeChanFunc: closeChanFunc,
	}

	return nil
}

func (pod *Pod) StartWatch(wg *sync.WaitGroup) {
	log.Println("Start watch handle for pods")

	defer pod.watcher.closeChanFunc()
	defer wg.Done()

	for watchResp := range pod.watcher.watchChan {
		if watchResp.Err() != nil {
			log.Fatal("error watcher pod")
		}

		for _, event := range watchResp.Events {
			fmt.Printf("Event received! %s executed on %q with value %q\n", event.Type, event.Kv.Key, event.Kv.Value)
		}
	}
}
