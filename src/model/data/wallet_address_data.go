package data

import (
	"github.com/assimon/luuu/model/dao"
	"github.com/assimon/luuu/model/mdb"
	"github.com/assimon/luuu/util/constant"
)

// AddWalletAddress 创建钱包
func AddWalletAddress(token string, channel string) (*mdb.WalletAddress, error) {
	exist, err := GetWalletAddressByToken(token)
	if err != nil {
		return nil, err
	}
	if exist.ID > 0 {
		return nil, constant.WalletAddressAlreadyExists
	}
	walletAddress := &mdb.WalletAddress{
		Token:   token,
		Channel: channel,
		Status:  mdb.TokenStatusEnable,
	}
	err = dao.Mdb.Create(walletAddress).Error
	return walletAddress, err
}

// GetWalletAddressByToken 通过钱包地址获取token
func GetWalletAddressByToken(token string) (*mdb.WalletAddress, error) {
	walletAddress := new(mdb.WalletAddress)
	err := dao.Mdb.Model(walletAddress).Limit(1).Find(walletAddress, "token = ?", token).Error
	return walletAddress, err
}

// GetWalletAddressById 通过id获取钱包
func GetWalletAddressById(id uint64) (*mdb.WalletAddress, error) {
	walletAddress := new(mdb.WalletAddress)
	err := dao.Mdb.Model(walletAddress).Limit(1).Find(walletAddress, id).Error
	return walletAddress, err
}

// DeleteWalletAddressById 通过id删除钱包
func DeleteWalletAddressById(id uint64) error {
	err := dao.Mdb.Where("id = ?", id).Delete(&mdb.WalletAddress{}).Error
	return err
}

// GetAvailableTrc20Wallet 获得所有可用的钱包地址
func GetAvailableTrc20Wallet() ([]mdb.WalletAddress, error) {
	var WalletAddressList []mdb.WalletAddress
	err := dao.Mdb.Model(WalletAddressList).Where("channel = 'trc20' and status = ?", mdb.TokenStatusEnable).Find(&WalletAddressList).Error
	return WalletAddressList, err
}

// GetAvailablePolygonWallet 获得所有可用的钱包地址
func GetAvailablePolygonWallet() ([]mdb.WalletAddress, error) {
	var WalletAddressList []mdb.WalletAddress
	err := dao.Mdb.Model(WalletAddressList).Where("channel = 'polygon' and status = ?", mdb.TokenStatusEnable).Find(&WalletAddressList).Error
	return WalletAddressList, err
}

// GetAllWalletAddress 获得所有钱包地址
func GetAllWalletAddress() ([]mdb.WalletAddress, error) {
	var WalletAddressList []mdb.WalletAddress
	err := dao.Mdb.Model(WalletAddressList).Find(&WalletAddressList).Error
	return WalletAddressList, err
}

// ChangeWalletAddressStatus 启用禁用钱包
func ChangeWalletAddressStatus(id uint64, status int) error {
	err := dao.Mdb.Model(&mdb.WalletAddress{}).Where("id = ?", id).Update("status", status).Error
	return err
}
