-- ============================================
-- Robusta 数据库表创建 SQL
-- 数据库: MySQL 8.0+
-- 字符集: utf8mb4
-- 生成时间: 2025-12-17
-- ============================================

-- 创建数据库（如果需要）
-- CREATE DATABASE IF NOT EXISTS robusta_hub DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
-- USE robusta_hub;

-- ============================================
-- 1. 集群表 (clusters)
-- ============================================
DROP TABLE IF EXISTS `clusters`;
CREATE TABLE `clusters` (
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at`     DATETIME(3)     NULL DEFAULT NULL COMMENT '软删除时间',
    `cluster_id`     VARCHAR(255)    NOT NULL COMMENT '集群唯一标识符',
    `name`           VARCHAR(255)    NOT NULL COMMENT '集群名称',
    `description`    TEXT            NULL COMMENT '集群描述',
    `status`         VARCHAR(32)     NOT NULL DEFAULT 'active' COMMENT '集群状态: active/inactive/maintenance',
    `last_heartbeat` DATETIME(3)     NULL COMMENT '最后心跳时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_cluster_id` (`cluster_id`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='集群信息表';

-- ============================================
-- 2. 告警表 (alerts)
-- ============================================
DROP TABLE IF EXISTS `alerts`;
CREATE TABLE `alerts` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at`      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at`      DATETIME(3)     NULL DEFAULT NULL COMMENT '软删除时间',
    `fingerprint`     VARCHAR(191)    NOT NULL COMMENT '告警指纹（用于去重）',
    `cluster_id`      VARCHAR(255)    NOT NULL COMMENT '关联集群ID',
    `title`           TEXT            NOT NULL COMMENT '告警标题',
    `description`     TEXT            NULL COMMENT '告警详细描述',
    `severity`        VARCHAR(32)     NOT NULL COMMENT '严重级别: info/warning/error/low/medium/high/critical',
    `status`          VARCHAR(32)     NOT NULL DEFAULT 'firing' COMMENT '告警状态: firing/resolved/silenced',
    `labels`          JSON            NULL COMMENT '告警标签（JSON格式）',
    `annotations`     JSON            NULL COMMENT '告警注解（JSON格式）',
    `starts_at`       DATETIME(3)     NULL COMMENT '告警开始时间',
    `ends_at`         DATETIME(3)     NULL COMMENT '告警结束时间',
    `raw_payload_key` TEXT            NULL COMMENT '原始载荷存储键（MinIO）',
    PRIMARY KEY (`id`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_cluster_id` (`cluster_id`),
    KEY `idx_fingerprint` (`fingerprint`),
    KEY `idx_severity` (`severity`),
    KEY `idx_status` (`status`),
    KEY `idx_starts_at` (`starts_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警信息表';

-- ============================================
-- 3. RCA运行记录表 (rca_runs)
-- ============================================
DROP TABLE IF EXISTS `rca_runs`;
CREATE TABLE `rca_runs` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at`      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at`      DATETIME(3)     NULL DEFAULT NULL COMMENT '软删除时间',
    `alert_id`        BIGINT UNSIGNED NOT NULL COMMENT '关联告警ID',
    `status`          VARCHAR(32)     NOT NULL COMMENT 'RCA状态: pending/running/completed/failed/timeout/queued',
    `summary`         TEXT            NULL COMMENT 'RCA分析摘要',
    `suspects`        JSON            NULL COMMENT '可疑原因列表（JSON格式）',
    `recommendations` JSON            NULL COMMENT '修复建议列表（JSON格式）',
    `attachments`     JSON            NULL COMMENT '附件信息（JSON格式）',
    `started_at`      DATETIME(3)     NOT NULL COMMENT 'RCA开始时间',
    `completed_at`    DATETIME(3)     NULL COMMENT 'RCA完成时间',
    `error_message`   TEXT            NULL COMMENT '错误信息（失败时）',
    `raw_payload_key` TEXT            NULL COMMENT '原始结果存储键（MinIO）',
    PRIMARY KEY (`id`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_alert_id` (`alert_id`),
    KEY `idx_status` (`status`),
    KEY `idx_started_at` (`started_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='RCA根因分析运行记录表';

-- ============================================
-- 4. 审计日志表 (audit_logs)
-- ============================================
DROP TABLE IF EXISTS `audit_logs`;
CREATE TABLE `audit_logs` (
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at`    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at`    DATETIME(3)     NULL DEFAULT NULL COMMENT '软删除时间',
    `user_id`       BIGINT UNSIGNED NULL COMMENT '操作用户ID',
    `action`        VARCHAR(64)     NOT NULL COMMENT '操作类型',
    `resource_type` VARCHAR(64)     NULL COMMENT '资源类型',
    `resource_id`   BIGINT UNSIGNED NULL COMMENT '资源ID',
    `details`       JSON            NULL COMMENT '操作详情（JSON格式）',
    `ip_address`    VARCHAR(45)     NULL COMMENT '客户端IP地址',
    `user_agent`    TEXT            NULL COMMENT '客户端User-Agent',
    PRIMARY KEY (`id`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_action` (`action`),
    KEY `idx_resource_type` (`resource_type`),
    KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='审计日志表';

-- ============================================
-- 5. 用户表 (users)
-- ============================================
DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at`     DATETIME(3)     NULL DEFAULT NULL COMMENT '软删除时间',
    `username`       VARCHAR(191)    NOT NULL COMMENT '用户名（唯一）',
    `email`          VARCHAR(191)    NOT NULL COMMENT '用户邮箱（唯一）',
    `name`           VARCHAR(255)    NULL COMMENT '用户显示名称',
    `picture`        TEXT            NULL COMMENT '用户头像URL',
    `is_admin`       TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '是否管理员: 0=否, 1=是',
    `email_verified` TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '邮箱是否已验证: 0=否, 1=是',
    `provider`       VARCHAR(64)     NOT NULL DEFAULT 'local' COMMENT '登录提供商: local/oidc/cas',
    `provider_id`    VARCHAR(191)    NULL COMMENT '第三方提供商用户ID',
    `last_login_at`  DATETIME(3)     NULL COMMENT '最后登录时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_username` (`username`),
    UNIQUE KEY `uk_email` (`email`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_provider` (`provider`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户信息表';

-- ============================================
-- 6. 刷新令牌表 (refresh_tokens)
-- ============================================
DROP TABLE IF EXISTS `refresh_tokens`;
CREATE TABLE `refresh_tokens` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at` DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at` DATETIME(3)     NULL DEFAULT NULL COMMENT '软删除时间',
    `user_id`    BIGINT UNSIGNED NOT NULL COMMENT '关联用户ID',
    `token`      VARCHAR(255)    NOT NULL COMMENT '令牌哈希值',
    `expires_at` DATETIME(3)     NOT NULL COMMENT '令牌过期时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_token` (`token`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='刷新令牌表';

-- ============================================
-- 7. API密钥表 (api_keys)
-- ============================================
DROP TABLE IF EXISTS `api_keys`;
CREATE TABLE `api_keys` (
    `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at`   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at`   DATETIME(3)     NULL DEFAULT NULL COMMENT '软删除时间',
    `user_id`      BIGINT UNSIGNED NOT NULL COMMENT '关联用户ID',
    `name`         VARCHAR(255)    NOT NULL COMMENT 'API密钥名称/描述',
    `key`          VARCHAR(255)    NOT NULL COMMENT 'API密钥值（加密存储）',
    `key_prefix`   VARCHAR(64)     NOT NULL COMMENT '密钥前缀（用于显示）',
    `last_used_at` DATETIME(3)     NULL COMMENT '最后使用时间',
    `expires_at`   DATETIME(3)     NULL COMMENT '过期时间（可选）',
    `is_active`    TINYINT(1)      NOT NULL DEFAULT 1 COMMENT '是否激活: 0=否, 1=是',
    `permissions`  VARCHAR(32)     NOT NULL DEFAULT 'read' COMMENT '权限范围: read/write/admin',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_key` (`key`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_is_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='API密钥表';

-- ============================================
-- 8. 系统设置表 (system_settings)
-- ============================================
DROP TABLE IF EXISTS `system_settings`;
CREATE TABLE `system_settings` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at`  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at`  DATETIME(3)     NULL DEFAULT NULL COMMENT '软删除时间',
    `setting_key` VARCHAR(191)    NOT NULL COMMENT '设置键名（唯一）',
    `value`       JSON            NULL COMMENT '设置值（JSON格式）',
    `description` TEXT            NULL COMMENT '设置描述',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_setting_key` (`setting_key`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统设置表';

-- ============================================
-- 9. 知识库文章表 (knowledge_articles)
-- ============================================
DROP TABLE IF EXISTS `knowledge_articles`;
CREATE TABLE `knowledge_articles` (
    `id`                        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at`                DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`                DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at`                DATETIME(3)     NULL DEFAULT NULL COMMENT '软删除时间',
    `alert_rule_name`           VARCHAR(255)    NOT NULL COMMENT '关联告警规则名称',
    `alert_rule_name_normalized` VARCHAR(255)   NOT NULL COMMENT '标准化告警规则名称（用于索引）',
    `tags`                      JSON            NULL COMMENT '标签列表（JSON格式）',
    `status`                    VARCHAR(16)     NOT NULL DEFAULT 'draft' COMMENT '文章状态: draft/published/archived',
    `object_key`                VARCHAR(512)    NOT NULL COMMENT 'MinIO对象存储键',
    `version`                   INT             NOT NULL DEFAULT 1 COMMENT '当前版本号',
    `created_by`                VARCHAR(128)    NULL COMMENT '创建人',
    `updated_by`                VARCHAR(128)    NULL COMMENT '最后更新人',
    PRIMARY KEY (`id`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_alert_rule_name_normalized` (`alert_rule_name_normalized`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='知识库文章表';

-- ============================================
-- 10. 知识库文章版本表 (knowledge_article_versions)
-- ============================================
DROP TABLE IF EXISTS `knowledge_article_versions`;
CREATE TABLE `knowledge_article_versions` (
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at`     DATETIME(3)     NULL DEFAULT NULL COMMENT '软删除时间',
    `article_id`     BIGINT UNSIGNED NOT NULL COMMENT '关联文章ID',
    `version`        INT             NOT NULL COMMENT '版本号',
    `object_key`     VARCHAR(512)    NOT NULL COMMENT 'MinIO对象存储键',
    `change_summary` TEXT            NULL COMMENT '变更摘要',
    `created_by`     VARCHAR(128)    NULL COMMENT '创建人',
    PRIMARY KEY (`id`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_article_id` (`article_id`),
    KEY `idx_version` (`article_id`, `version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='知识库文章版本历史表';

-- ============================================
-- 外键约束（可选，根据需要启用）
-- ============================================
-- 注意：GORM 模型中注释了不使用数据库外键约束
-- 如果需要启用外键，取消以下注释：

-- ALTER TABLE `alerts`
--     ADD CONSTRAINT `fk_alerts_cluster` FOREIGN KEY (`cluster_id`) REFERENCES `clusters` (`cluster_id`)
--     ON DELETE SET NULL ON UPDATE CASCADE;

-- ALTER TABLE `rca_runs`
--     ADD CONSTRAINT `fk_rca_runs_alert` FOREIGN KEY (`alert_id`) REFERENCES `alerts` (`id`)
--     ON DELETE CASCADE ON UPDATE CASCADE;

-- ALTER TABLE `refresh_tokens`
--     ADD CONSTRAINT `fk_refresh_tokens_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
--     ON DELETE CASCADE ON UPDATE CASCADE;

-- ALTER TABLE `api_keys`
--     ADD CONSTRAINT `fk_api_keys_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
--     ON DELETE CASCADE ON UPDATE CASCADE;

-- ALTER TABLE `knowledge_article_versions`
--     ADD CONSTRAINT `fk_knowledge_versions_article` FOREIGN KEY (`article_id`) REFERENCES `knowledge_articles` (`id`)
--     ON DELETE CASCADE ON UPDATE CASCADE;

-- ============================================
-- 初始数据（可选）
-- ============================================

-- 插入默认系统设置
INSERT INTO `system_settings` (`setting_key`, `value`, `description`) VALUES
('auto_rca', '{"enabled": false, "rate_limit": 10, "period": 60, "allowed_severities": ["critical", "high"]}', 'Auto-RCA自动分析配置'),
('notification', '{"email_enabled": false, "webhook_enabled": false}', '通知配置')
ON DUPLICATE KEY UPDATE `updated_at` = CURRENT_TIMESTAMP(3);

-- ============================================
-- 索引优化建议
-- ============================================
-- 1. 对于高频查询字段，已创建索引
-- 2. JSON 字段可使用虚拟列 + 索引进行优化（MySQL 5.7+）
-- 3. 大表建议使用分区表优化查询性能
-- 4. 定期清理软删除数据或归档历史数据
