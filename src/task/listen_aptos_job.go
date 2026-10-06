package task

import (
	"sync"

	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/model"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/service"
	"github.com/assimon/luuu/util/log"
)

type ListenAptosJob struct {
}

var gListenAptosJobLock sync.Mutex

func (r ListenAptosJob) Run() {
	gListenAptosJobLock.Lock()
	defer gListenAptosJobLock.Unlock()

	var wg sync.WaitGroup

	walletAddress, err := data.GetAvailableWallet(model.ChainNameAptos)
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	if len(walletAddress) <= 0 {
		return
	}
	for _, address := range walletAddress {
		for _, asset := range model.EnabledPaymentAssets(model.ChainNameAptos, config.IsPaymentAssetEnabled) {
			wg.Add(1)
			go service.AptosApiScan(asset, address.Token, &wg)
		}
	}

	wg.Wait()
}
