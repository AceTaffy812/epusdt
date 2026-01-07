package task

import (
	"sync"

	"github.com/assimon/luuu/model"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/service"
	"github.com/assimon/luuu/util/log"
)

type ListenEvmJob struct {
}

var gListenEvmJobLock sync.Mutex

func (r ListenEvmJob) Run() {
	gListenEvmJobLock.Lock()
	defer gListenEvmJobLock.Unlock()

	var wg sync.WaitGroup

	listerner := func(chainName string) {
		walletAddress, err := data.GetAvailableWallet(chainName)
		if err != nil {
			log.Sugar.Error(err)
			return
		}
		if len(walletAddress) <= 0 {
			return
		}
		for _, address := range walletAddress {
			wg.Add(1)
			go service.EtherscanApiScan(chainName, address.Token, &wg)
		}
	}

	listerner(model.ChainNamePolygonPOS)
	listerner(model.ChainNameAVAXC)
	listerner(model.ChainNameBSC)
	listerner(model.ChainNameETH)
	listerner(model.ChainNameArbitrum)

	wg.Wait()
}
