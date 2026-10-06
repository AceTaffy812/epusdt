package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	AppDebug bool
	// AppVersion is set by GoReleaser through ldflags. Local builds use dev.
	AppVersion  = "dev"
	MysqlDns    string
	RuntimePath string
	LogSavePath string
	StaticPath  string
	TgBotToken  string
	TgProxy     string
	TgManage    int64
	UsdtRate    float64
)

func Init() {
	viper.AddConfigPath("./")
	viper.SetConfigFile(".env")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	gwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	AppDebug = viper.GetBool("app_debug")
	StaticPath = viper.GetString("static_path")
	RuntimePath = fmt.Sprintf(
		"%s%s",
		gwd,
		viper.GetString("runtime_root_path"))
	os.Mkdir(RuntimePath, 0755)
	LogSavePath = fmt.Sprintf(
		"%s%s",
		RuntimePath,
		viper.GetString("log_save_path"))
	os.Mkdir(LogSavePath, 0755)
	MysqlDns = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		viper.GetString("mysql_user"),
		viper.GetString("mysql_passwd"),
		fmt.Sprintf(
			"%s:%s",
			viper.GetString("mysql_host"),
			viper.GetString("mysql_port")),
		viper.GetString("mysql_database"))
	TgBotToken = viper.GetString("tg_bot_token")
	TgProxy = viper.GetString("tg_proxy")
	TgManage = viper.GetInt64("tg_manage")
}

func GetAppName() string {
	appName := viper.GetString("app_name")
	if appName == "" {
		return "epusdt"
	}
	return appName
}

func GetAppUri() string {
	return viper.GetString("app_uri")
}

func GetEtherscanApi() string {
	return viper.GetString("etherscan_api")
}

func GetApiAuthToken() string {
	return viper.GetString("api_auth_token")
}

func GetUsdtRate() float64 {
	forcedUsdtRate := viper.GetFloat64("forced_usdt_rate")
	if forcedUsdtRate > 0 {
		return forcedUsdtRate
	}
	if UsdtRate <= 0 {
		return 6.4
	}
	return UsdtRate
}

// IsPaymentAssetEnabled returns true for every built-in asset when the optional
// enabled_channel setting is absent. Explicit settings use chain_asset values.
func IsPaymentAssetEnabled(chain, asset string) bool {
	raw := strings.TrimSpace(viper.GetString("enabled_channel"))
	if raw == "" {
		return true
	}
	wanted := strings.ToLower(chain + "_" + asset)
	for _, value := range strings.Split(raw, ",") {
		value = normalizeEnabledChannel(value)
		if value == wanted || value == strings.ToLower(chain) {
			return true
		}
	}
	return false
}

// GetPaymentChainOrder returns the network order configured in enabled_channel.
// The first occurrence of a network wins, so both chain_asset values and base
// chain values can be used to control the order shown by the checkout.
func GetPaymentChainOrder() []string {
	raw := strings.TrimSpace(viper.GetString("enabled_channel"))
	if raw == "" {
		return nil
	}

	order := make([]string, 0)
	seen := make(map[string]struct{})
	for _, value := range strings.Split(raw, ",") {
		value = normalizeEnabledChannel(value)
		if value == "" {
			continue
		}
		chain := value
		if index := strings.IndexByte(value, '_'); index >= 0 {
			chain = value[:index]
		}
		if _, ok := seen[chain]; ok {
			continue
		}
		seen[chain] = struct{}{}
		order = append(order, chain)
	}
	return order
}

func normalizeEnabledChannel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.ReplaceAll(value, "_-_", "_")
}

func GetOrderExpirationTime() int {
	timer := viper.GetInt("order_expiration_time")
	if timer <= 0 {
		return 10
	}
	return timer
}

func GetOrderExpirationTimeDuration() time.Duration {
	timer := GetOrderExpirationTime()
	return time.Minute * time.Duration(timer)
}
