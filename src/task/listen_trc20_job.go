package task

import (
	"sync"

	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/model"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/service"
	"github.com/assimon/luuu/util/log"
)

type ListenTrc20Job struct {
}

var gListenTrc20JobLock sync.Mutex

func (r ListenTrc20Job) Run() {
	gListenTrc20JobLock.Lock()
	defer gListenTrc20JobLock.Unlock()
	if !config.IsPaymentAssetEnabled(model.ChainNameTRC20, model.AssetUSDT) {
		return
	}
	walletAddress, err := data.GetAvailableWallet(model.ChainNameTRC20)
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
		go service.Trc20ApiScan(address.Token, &wg)
	}
	wg.Wait()
}
