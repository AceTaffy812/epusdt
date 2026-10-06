package data

import (
	"context"
	"fmt"
	"time"

	"github.com/assimon/luuu/model/dao"
	"github.com/assimon/luuu/model/mdb"
	"github.com/assimon/luuu/model/request"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var (
	CacheWalletAddressWithAmountToTradeIdKey = "wallet:%s:%s:%s_%v" // 链:资产:钱包_待支付金额 : 交易号
)

// GetOrderInfoByOrderId 通过客户订单号查询订单
func GetOrderInfoByOrderId(orderId string) (*mdb.Orders, error) {
	order := new(mdb.Orders)
	err := dao.Mdb.Model(order).Limit(1).Find(order, "order_id = ?", orderId).Error
	return order, err
}

// GetOrderInfoByTradeId 通过交易号查询订单
func GetOrderInfoByTradeId(tradeId string) (*mdb.Orders, error) {
	order := new(mdb.Orders)
	err := dao.Mdb.Model(order).Limit(1).Find(order, "trade_id = ?", tradeId).Error
	return order, err
}

// CreateOrderWithTransaction 事务创建订单
func CreateOrderWithTransaction(tx *gorm.DB, order *mdb.Orders) error {
	err := tx.Model(order).Create(order).Error
	return err
}

// GetOrderByBlockIdWithTransaction 通过区块获取订单
func GetOrderByBlockIdWithTransaction(tx *gorm.DB, blockId string) (*mdb.Orders, error) {
	order := new(mdb.Orders)
	err := tx.Model(order).Limit(1).Find(order, "block_transaction_id = ?", blockId).Error
	return order, err
}

// OrderSuccessWithTransaction 事务支付成功
func OrderSuccessWithTransaction(tx *gorm.DB, req *request.OrderProcessingRequest) error {
	err := tx.Model(&mdb.Orders{}).Where("trade_id = ?", req.TradeId).Updates(map[string]interface{}{
		"block_transaction_id": req.BlockTransactionId,
		"status":               mdb.StatusPaySuccess,
		"callback_confirm":     mdb.CallBackConfirmNo,
	}).Error
	return err
}

// GetPendingCallbackOrders 查询出等待回调的订单
func GetPendingCallbackOrders() ([]mdb.Orders, error) {
	var orders []mdb.Orders
	err := dao.Mdb.Model(orders).
		Where("callback_num < ?", 5).
		Where("callback_confirm = ?", mdb.CallBackConfirmNo).
		Where("status = ?", mdb.StatusPaySuccess).
		Find(&orders).Error
	return orders, err
}

// SaveCallBackOrdersResp 保存订单回调结果
func SaveCallBackOrdersResp(order *mdb.Orders) error {
	err := dao.Mdb.Model(order).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"callback_num":     gorm.Expr("callback_num + ?", 1),
		"callback_confirm": order.CallBackConfirm,
	}).Error
	return err
}

// UpdateOrderIsExpirationById 通过id设置订单过期
func UpdateOrderIsExpirationById(id uint64) error {
	err := dao.Mdb.Model(mdb.Orders{}).Where("id = ?", id).Update("status", mdb.StatusExpired).Error
	return err
}

// GetTradeIdByWalletAddressAndAmount 通过钱包地址，支付金额获取交易号
func GetTradeIdByWalletAddressAndAmount(chain, asset, address string, amount float64) (string, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf(CacheWalletAddressWithAmountToTradeIdKey, chain, asset, address, amount)
	result, err := dao.Rdb.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return result, nil
}

// LockTransaction 锁定交易
func LockTransaction(chain, asset, address, tradeId string, amount float64, expirationTime time.Duration) error {
	ctx := context.Background()
	cacheKey := fmt.Sprintf(CacheWalletAddressWithAmountToTradeIdKey, chain, asset, address, amount)
	err := dao.Rdb.Set(ctx, cacheKey, tradeId, expirationTime).Err()
	return err
}

// UnLockTransaction 解锁交易
func UnLockTransaction(chain, asset, address string, amount float64) error {
	ctx := context.Background()
	cacheKey := fmt.Sprintf(CacheWalletAddressWithAmountToTradeIdKey, chain, asset, address, amount)
	err := dao.Rdb.Del(ctx, cacheKey).Err()
	return err
}

// IsWalletLocked 查询钱包是否已被锁定（有任意金额的订单）
// 结果可能不太准确，倾向于已被锁定
func IsWalletLocked(chain, asset, address string) bool {
	ctx := context.Background()
	cacheKey := fmt.Sprintf(CacheWalletAddressWithAmountToTradeIdKey, chain, asset, address, "*")

	var cursor uint64
	// var count uint64

	for {
		keys, nextCursor, err := dao.Rdb.Scan(ctx, cursor, cacheKey, 1000).Result()
		if err != nil {
			fmt.Println("[redis scan]", err)
			return true // ？
		}

		for _, key := range keys {
			// 只统计未过期的键（过期键可能仍然在 SCAN 中被扫到，但已实际失效）
			ttl, err := dao.Rdb.TTL(ctx, key).Result()
			if err != nil {
				return true // ？
			}
			// ttl > 0 表示有效
			if ttl > 0 {
				// count++
				// 确定至少有一个锁定的交易
				return true
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return false
}
