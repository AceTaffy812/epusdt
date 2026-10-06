-- Multi-asset support migration. Existing orders are preserved as USDT.
ALTER TABLE `orders`
    CHANGE COLUMN `token` `wallet_address` VARCHAR(100) NOT NULL COMMENT '所属钱包地址（带有链前缀）' AFTER `order_id`,
    ADD COLUMN `asset` VARCHAR(32) NOT NULL DEFAULT 'usdt' COMMENT '支付资产快照' AFTER `wallet_address`,
    MODIFY COLUMN `amount` DECIMAL(19, 4) NOT NULL COMMENT '订单金额，保留4位小数' AFTER `asset`,
    MODIFY COLUMN `actual_amount` DECIMAL(19, 4) NOT NULL COMMENT '订单实际需要支付的金额，保留4位小数' AFTER `amount`,
    MODIFY COLUMN `block_transaction_id` VARCHAR(128) NULL COMMENT '区块唯一编号' AFTER `actual_amount`;
