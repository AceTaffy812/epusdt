package task

import "github.com/robfig/cron/v3"

func Start() {
	c := cron.New()
	// 汇率监听
	c.AddJob("@every 60s", UsdtRateJob{})
	// trc20钱包监听
	c.AddJob("@every 15s", ListenTrc20Job{})
	// polygon钱包监听
	c.AddJob("@every 30s", ListenPolygonJob{})
	c.Start()
}
