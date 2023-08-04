package task

import (
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/service"
	"github.com/assimon/luuu/util/log"
	"sync"
)

type ListenPolygonJob struct {
}

var gListenPolygonJobLock sync.Mutex

func (r ListenPolygonJob) Run() {
	gListenPolygonJobLock.Lock()
	defer gListenPolygonJobLock.Unlock()
	walletAddress, err := data.GetAvailablePolygonWallet()
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	if len(walletAddress) <= 0 {
		return
	}
	var wg sync.WaitGroup
	for _, address := range walletAddress {
		wg.Add(1)
		if err != nil {
			log.Sugar.Error(err)
			return
		}
		go service.PolygonCallBack(address.Token, &wg)
	}
	wg.Wait()
}
