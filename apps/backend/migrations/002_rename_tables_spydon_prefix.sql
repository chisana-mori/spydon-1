-- Active: 1767102429236@@127.0.0.1@3306@robusta_hub
-- =============================================================================
-- Spydon-AWX 数据库迁移脚本
-- 目的: 将现有表重命名为 spydon_ 前缀，clusters 表与 Kite 共用
-- 执行前请备份数据库！
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 第一部分: 重命名现有表（添加 spydon_ 前缀）
-- -----------------------------------------------------------------------------

-- 重命名用户相关表
RENAME TABLE `users` TO `spydon_users`;
RENAME TABLE `refresh_tokens` TO `spydon_refresh_tokens`;
RENAME TABLE `api_keys` TO `spydon_api_keys`;

-- 重命名告警相关表
RENAME TABLE `alerts` TO `spydon_alerts`;
RENAME TABLE `rca_runs` TO `spydon_rca_runs`;

-- 重命名审计日志表
RENAME TABLE `audit_logs` TO `spydon_audit_logs`;

-- 重命名知识库表
RENAME TABLE `knowledge_articles` TO `spydon_knowledge_articles`;
RENAME TABLE `knowledge_article_versions` TO `spydon_knowledge_article_versions`;

-- 重命名系统设置表
RENAME TABLE `system_settings` TO `spydon_system_settings`;

-- 重命名流水线相关表
RENAME TABLE `pipeline_templates` TO `spydon_pipeline_templates`;
RENAME TABLE `pipeline_executions` TO `spydon_pipeline_executions`;
RENAME TABLE `stage_runs` TO `spydon_stage_runs`;

-- -----------------------------------------------------------------------------
-- 第二部分: 修改 clusters 表结构以兼容 Kite
-- -----------------------------------------------------------------------------

-- 使用存储过程安全添加列（如果不存在）
DELIMITER //

DROP PROCEDURE IF EXISTS add_column_if_not_exists//

CREATE PROCEDURE add_column_if_not_exists()
BEGIN
    -- 添加 in_cluster 列
    IF NOT EXISTS (
        SELECT * FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'clusters'
        AND COLUMN_NAME = 'in_cluster'
    ) THEN
        ALTER TABLE `clusters` ADD COLUMN `in_cluster` BOOLEAN DEFAULT FALSE;
    END IF;

    -- 添加 is_default 列
    IF NOT EXISTS (
        SELECT * FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'clusters'
        AND COLUMN_NAME = 'is_default'
    ) THEN
        ALTER TABLE `clusters` ADD COLUMN `is_default` BOOLEAN DEFAULT FALSE;
    END IF;

    -- 添加 enable 列
    IF NOT EXISTS (
        SELECT * FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'clusters'
        AND COLUMN_NAME = 'enable'
    ) THEN
        ALTER TABLE `clusters` ADD COLUMN `enable` BOOLEAN DEFAULT TRUE;
    END IF;

    -- 重命名 kube_config 为 config（如果存在）
    IF EXISTS (
        SELECT * FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'clusters'
        AND COLUMN_NAME = 'kube_config'
    ) THEN
        ALTER TABLE `clusters` CHANGE COLUMN `kube_config` `config` TEXT;
    END IF;
END//

DELIMITER ;

-- 执行存储过程
CALL add_column_if_not_exists();

-- 清理存储过程
DROP PROCEDURE IF EXISTS add_column_if_not_exists;

-- -----------------------------------------------------------------------------
-- 第三部分: 更新 spydon_alerts 表的外键引用（可选）
-- -----------------------------------------------------------------------------

-- 如果有外键约束，需要更新
-- ALTER TABLE `spydon_alerts` DROP FOREIGN KEY IF EXISTS `fk_alerts_cluster`;
-- ALTER TABLE `spydon_rca_runs` DROP FOREIGN KEY IF EXISTS `fk_rca_runs_alert`;

-- -----------------------------------------------------------------------------
-- 验证迁移结果
-- -----------------------------------------------------------------------------

-- 检查表是否正确重命名
SELECT
    TABLE_NAME,
    TABLE_ROWS
FROM
    information_schema.TABLES
WHERE
    TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME LIKE 'spydon_%'
ORDER BY TABLE_NAME;

-- 检查 clusters 表结构
DESCRIBE `clusters`;
