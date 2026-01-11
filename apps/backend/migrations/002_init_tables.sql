-- =============================================================================
-- Spydon-AWX 数据库初始化脚本 (带有中文注释)
-- 基于 Internal Models 定义
-- =============================================================================
SET NAMES utf8mb4;

-- -----------------------------------------------------------------------------
-- 表: clusters
-- 说明: 集群信息表，与 Kite 共享
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `clusters` (
    `id`             INT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '集群ID (主键)',
    `created_at`     DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`     DATETIME(3) NULL COMMENT '更新时间',
    `name`           VARCHAR(100) NOT NULL COMMENT '集群名称 (唯一索引)',
    `description`    TEXT COMMENT '集群描述',
    `config`         TEXT COMMENT 'KubeConfigs配置 (自动加密存储)',
    `prometheus_url` VARCHAR(255) COMMENT 'Prometheus API 地址',
    `in_cluster`     BOOLEAN DEFAULT FALSE COMMENT '是否为当前集群',
    `is_default`     BOOLEAN DEFAULT FALSE COMMENT '是否为默认集群',
    `enable`         BOOLEAN DEFAULT TRUE COMMENT '是否启用',
    `cluster_id`     VARCHAR(255) COMMENT 'Spydon 特有集群标识',
    `status`         VARCHAR(32) DEFAULT 'active' COMMENT 'Spydon 集群状态 (active/inactive)',
    `last_heartbeat` DATETIME(3) NULL COMMENT 'Spydon 最后一次心跳时间',
    UNIQUE INDEX `idx_clusters_name` (`name`),
    INDEX `idx_clusters_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Kubernetes集群信息表';


-- -----------------------------------------------------------------------------
-- 表: spydon_users
-- 说明: 用户信息表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_users` (
    `id`             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '用户ID (主键)',
    `created_at`     DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`     DATETIME(3) NULL COMMENT '更新时间',
    `username`       VARCHAR(191) NOT NULL COMMENT '用户名',
    `email`          VARCHAR(191) NOT NULL COMMENT '邮箱',
    `name`           VARCHAR(255) COMMENT '显示名称',
    `picture`        TEXT COMMENT '头像URL',
    `is_admin`       BOOLEAN DEFAULT FALSE COMMENT '是否为管理员',
    `email_verified` BOOLEAN DEFAULT FALSE COMMENT '邮箱是否已验证',
    `provider`       VARCHAR(64) DEFAULT 'local' COMMENT '认证提供商 (local/github/google)',
    `provider_id`    VARCHAR(191) COMMENT '第三方认证提供商的用户ID',
    `last_login_at`  DATETIME(3) NULL COMMENT '最后登录时间',
    UNIQUE INDEX `idx_spydon_users_username` (`username`),
    UNIQUE INDEX `idx_spydon_users_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户信息表';


-- -----------------------------------------------------------------------------
-- 表: spydon_refresh_tokens
-- 说明: 用户刷新令牌表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_refresh_tokens` (
    `id`         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at` DATETIME(3) NULL COMMENT '创建时间',
    `updated_at` DATETIME(3) NULL COMMENT '更新时间',
    `user_id`    BIGINT UNSIGNED NOT NULL COMMENT '关联的用户ID',
    `token`      VARCHAR(255) NOT NULL COMMENT 'Refresh Token 哈希值',
    `expires_at` DATETIME(3) NOT NULL COMMENT '过期时间',
    UNIQUE INDEX `idx_spydon_refresh_tokens_token` (`token`),
    INDEX `idx_spydon_refresh_tokens_user_id` (`user_id`),
    INDEX `idx_refresh_tokens_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户刷新令牌表';


-- -----------------------------------------------------------------------------
-- 表: spydon_api_keys
-- 说明: API 密钥表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_api_keys` (
    `id`           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at`   DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`   DATETIME(3) NULL COMMENT '更新时间',
    `user_id`      BIGINT UNSIGNED NOT NULL COMMENT '关联的用户ID',
    `name`         VARCHAR(255) NOT NULL COMMENT '密钥名称/描述',
    `key`          VARCHAR(255) NOT NULL COMMENT '密钥值 (加密存储)',
    `key_prefix`   VARCHAR(64) NOT NULL COMMENT '密钥前缀 (用于展示)',
    `last_used_at` DATETIME(3) NULL COMMENT '最后使用时间',
    `expires_at`   DATETIME(3) NULL COMMENT '过期时间',
    `is_active`    BOOLEAN DEFAULT TRUE COMMENT '是否激活',
    `permissions`  VARCHAR(32) DEFAULT 'read' COMMENT '权限范围 (read/write/admin)',
    UNIQUE INDEX `idx_spydon_api_keys_key` (`key`),
    INDEX `idx_spydon_api_keys_user_id` (`user_id`),
    INDEX `idx_api_keys_key_prefix` (`key_prefix`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API密钥表';


-- -----------------------------------------------------------------------------
-- 表: spydon_alerts
-- 说明: 告警信息表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_alerts` (
    `id`              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at`      DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`      DATETIME(3) NULL COMMENT '更新时间',
    `fingerprint`     VARCHAR(191) NOT NULL COMMENT '告警指纹 (唯一标识)',
    `cluster_name`    VARCHAR(255) NOT NULL COMMENT '集群名称',
    `cluster_id`      VARCHAR(255) COMMENT '集群ID',
    `title`           TEXT NOT NULL COMMENT '告警标题/摘要',
    `description`     TEXT COMMENT '告警详细描述',
    `severity`        VARCHAR(32) NOT NULL COMMENT '严重级别 (critical/high/...)',
    `status`          VARCHAR(32) DEFAULT 'firing' COMMENT '状态 (firing/resolved)',
    `labels`          JSON COMMENT 'Prometheus 标签集',
    `annotations`     JSON COMMENT 'Prometheus 注解集',
    `starts_at`       DATETIME(3) NULL COMMENT '告警开始时间',
    `ends_at`         DATETIME(3) NULL COMMENT '告警结束时间',
    `raw_payload_key` TEXT COMMENT 'MinIO中存储原始Payload的Key',
    INDEX `idx_spydon_alerts_fingerprint` (`fingerprint`),
    INDEX `idx_spydon_alerts_cluster_name` (`cluster_name`),
    INDEX `idx_spydon_alerts_severity` (`severity`),
    INDEX `idx_spydon_alerts_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警信息表';


-- -----------------------------------------------------------------------------
-- 表: spydon_rca_runs
-- 说明: 根因分析(RCA)运行记录表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_rca_runs` (
    `id`              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at`      DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`      DATETIME(3) NULL COMMENT '更新时间',
    `alert_id`        BIGINT UNSIGNED NOT NULL COMMENT '关联的告警ID',
    `status`          VARCHAR(32) NOT NULL COMMENT '运行状态 (pending/running/completed/failed)',
    `summary`         TEXT COMMENT '分析总结',
    `suspects`        JSON COMMENT '疑似根因列表',
    `recommendations` JSON COMMENT '建议操作列表',
    `attachments`     JSON COMMENT '附件/证据列表',
    `started_at`      DATETIME(3) NULL COMMENT '开始时间',
    `completed_at`    DATETIME(3) NULL COMMENT '完成时间',
    `error_message`   TEXT COMMENT '错误信息',
    `raw_payload_key` TEXT COMMENT 'MinIO中存储相关Payload的Key',
    INDEX `idx_spydon_rca_runs_alert_id` (`alert_id`),
    INDEX `idx_spydon_rca_runs_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='RCA根因分析运行记录表';


-- -----------------------------------------------------------------------------
-- 表: spydon_audit_logs
-- 说明: 审计日志表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_audit_logs` (
    `id`            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at`    DATETIME(3) NULL COMMENT '操作时间',
    `updated_at`    DATETIME(3) NULL COMMENT '更新时间',
    `user_id`       BIGINT UNSIGNED COMMENT '操作用户ID (系统操作为空)',
    `action`        VARCHAR(64) NOT NULL COMMENT '操作动作 (如 create_cluster)',
    `resource_type` VARCHAR(64) COMMENT '资源类型 (如 cluster, alert)',
    `resource_id`   BIGINT UNSIGNED COMMENT '资源ID',
    `details`       JSON COMMENT '操作详情/参数',
    `ip_address`    VARCHAR(45) COMMENT '客户端IP地址',
    `user_agent`    TEXT COMMENT '客户端User Agent',
    INDEX `idx_spydon_audit_logs_user_id` (`user_id`),
    INDEX `idx_spydon_audit_logs_action` (`action`),
    INDEX `idx_spydon_audit_logs_resource` (`resource_type`, `resource_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统审计日志表';


-- -----------------------------------------------------------------------------
-- 表: spydon_knowledge_articles
-- 说明: 知识库文章表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_knowledge_articles` (
    `id`                         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at`                 DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`                 DATETIME(3) NULL COMMENT '更新时间',
    `alert_rule_name`            VARCHAR(255) NOT NULL COMMENT '关联的告警规则名称',
    `alert_rule_name_normalized` VARCHAR(255) NOT NULL COMMENT '归一化的规则名称 (索引)',
    `tags`                       JSON COMMENT '标签',
    `status`                     VARCHAR(16) DEFAULT 'draft' COMMENT '状态 (draft/published)',
    `object_key`                 VARCHAR(512) NOT NULL COMMENT 'MinIO对象Key',
    `version`                    INT NOT NULL DEFAULT 1 COMMENT '当前版本号',
    `created_by`                 VARCHAR(128) COMMENT '创建者',
    `updated_by`                 VARCHAR(128) COMMENT '更新者',
    INDEX `idx_spydon_ka_normalized_name` (`alert_rule_name_normalized`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识库文章表';


-- -----------------------------------------------------------------------------
-- 表: spydon_knowledge_article_versions
-- 说明: 知识库文章历史版本表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_knowledge_article_versions` (
    `id`             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at`     DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`     DATETIME(3) NULL COMMENT '更新时间',
    `article_id`     BIGINT UNSIGNED NOT NULL COMMENT '关联的文章ID',
    `version`        INT NOT NULL COMMENT '版本号',
    `object_key`     VARCHAR(512) NOT NULL COMMENT 'MinIO对象Key',
    `change_summary` TEXT COMMENT '变更摘要',
    `created_by`     VARCHAR(128) COMMENT '版本创建者',
    INDEX `idx_spydon_kav_article_id` (`article_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识库文章历史版本表';


-- -----------------------------------------------------------------------------
-- 表: spydon_system_settings
-- 说明: 系统全局设置表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_system_settings` (
    `id`          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at`  DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`  DATETIME(3) NULL COMMENT '更新时间',
    `setting_key` VARCHAR(191) NOT NULL COMMENT '配置键 (唯一索引)',
    `value`       JSON COMMENT '配置值 (JSON格式)',
    `description` TEXT COMMENT '配置描述',
    UNIQUE INDEX `idx_spydon_system_settings_key` (`setting_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统全局设置表';


-- -----------------------------------------------------------------------------
-- 表: spydon_pipeline_templates
-- 说明: 流水线模板表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_pipeline_templates` (
    `id`          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at`  DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`  DATETIME(3) NULL COMMENT '更新时间',
    `name`        VARCHAR(255) NOT NULL COMMENT '模板名称 (唯一)',
    `description` TEXT COMMENT '模板描述',
    `stages`      JSON NOT NULL COMMENT '阶段定义 (JSON)',
    `created_by`  BIGINT UNSIGNED COMMENT '创建者ID',
    UNIQUE INDEX `idx_spydon_pipeline_templates_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流水线模板表';


-- -----------------------------------------------------------------------------
-- 表: spydon_pipeline_executions
-- 说明: 流水线执行实例表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_pipeline_executions` (
    `id`                   BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at`           DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`           DATETIME(3) NULL COMMENT '更新时间',
    `pipeline_template_id` BIGINT UNSIGNED NOT NULL COMMENT '关联的模板ID',
    `cluster_id`           BIGINT UNSIGNED COMMENT '目标集群ID',
    `cluster_name`         VARCHAR(255) COMMENT '目标集群名称',
    `status`               VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT '执行状态',
    `parameters`           JSON COMMENT '运行时参数',
    `current_stage_id`     VARCHAR(64) COMMENT '当前阶段ID',
    `started_at`           DATETIME(3) NULL COMMENT '开始时间',
    `completed_at`         DATETIME(3) NULL COMMENT '完成时间',
    `triggered_by`         BIGINT UNSIGNED COMMENT '触发者ID',
    `error_message`        TEXT COMMENT '错误信息',
    INDEX `idx_spydon_pe_template_id` (`pipeline_template_id`),
    INDEX `idx_spydon_pe_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流水线执行实例表';


-- -----------------------------------------------------------------------------
-- 表: spydon_stage_runs
-- 说明: 流水线阶段执行记录表
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_stage_runs` (
    `id`             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT 'ID (主键)',
    `created_at`     DATETIME(3) NULL COMMENT '创建时间',
    `updated_at`     DATETIME(3) NULL COMMENT '更新时间',
    `execution_id`   BIGINT UNSIGNED NOT NULL COMMENT '关联的流水线执行ID',
    `stage_id`       VARCHAR(64) NOT NULL COMMENT '阶段ID',
    `stage_name`     VARCHAR(255) COMMENT '阶段名称',
    `stage_type`     VARCHAR(32) COMMENT '阶段类型 (awx_job/manual_gate/...)',
    `status`         VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT '阶段状态',
    `awx_job_id`     BIGINT COMMENT '关联的AWX Job ID',
    `awx_job_status` VARCHAR(32) COMMENT 'AWX Job 状态',
    `output`         JSON COMMENT '阶段输出/结果',
    `error_message`  TEXT COMMENT '错误信息',
    `started_at`     DATETIME(3) NULL COMMENT '开始时间',
    `completed_at`   DATETIME(3) NULL COMMENT '完成时间',
    `approved_by`    BIGINT UNSIGNED COMMENT '审批人ID',
    `approval_notes` TEXT COMMENT '审批备注',
    INDEX `idx_spydon_sr_execution_id` (`execution_id`),
    INDEX `idx_stage_runs_execution_id` (`execution_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流水线阶段执行记录表';


-- -----------------------------------------------------------------------------
-- 表: spydon_dictionaries
-- 说明: 数据字典表，管理系统的数据字典定义
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_dictionaries` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID，唯一标识一个数据字典',
    `created_at`      DATETIME(3) DEFAULT NULL COMMENT '创建时间，精确到毫秒',
    `updated_at`      DATETIME(3) DEFAULT NULL COMMENT '最后更新时间，精确到毫秒',
    `code`            VARCHAR(100) NOT NULL COMMENT '字典编码，唯一标识字典，如alarm_status',
    `name`            VARCHAR(255) NOT NULL COMMENT '字典名称，如告警状态',
    `module`          VARCHAR(100) DEFAULT NULL COMMENT '所属模块，如alarm、system等',
    `description`     TEXT COMMENT '字典描述，说明字典用途和取值范围',
    `is_enabled`      TINYINT(1) DEFAULT '1' COMMENT '是否启用，1表示启用，0表示禁用',
    `key_same_as_value` TINYINT(1) DEFAULT NULL COMMENT '键值是否相同，1表示相同，0表示不同',
    `sort_order`      BIGINT DEFAULT '0' COMMENT '排序序号，用于字典列表排序',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_spydon_dictionaries_code` (`code`) COMMENT '字典编码唯一索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据字典表';

-- -----------------------------------------------------------------------------
-- 表: spydon_dictionary_items
-- 说明: 数据字典项表，管理数据字典的具体条目
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `spydon_dictionary_items` (
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID，唯一标识一个字典项',
    `created_at`     DATETIME(3) DEFAULT NULL COMMENT '创建时间，精确到毫秒',
    `updated_at`     DATETIME(3) DEFAULT NULL COMMENT '最后更新时间，精确到毫秒',
    `dictionary_id`  BIGINT UNSIGNED NOT NULL COMMENT '所属字典ID，关联spydon_dictionaries表',
    `key`            VARCHAR(255) NOT NULL COMMENT '字典项键，字典项的标识符',
    `value`          VARCHAR(255) NOT NULL COMMENT '字典项值，字典项的显示文本',
    `description`    TEXT COMMENT '字典项描述，说明该项的含义和用途',
    `is_default`     TINYINT(1) DEFAULT '0' COMMENT '是否为默认值，1表示是，0表示否',
    `is_enabled`     TINYINT(1) DEFAULT '1' COMMENT '是否启用，1表示启用，0表示禁用',
    `sort_order`     BIGINT DEFAULT '0' COMMENT '排序序号，用于字典项列表排序',
    `extra`          TEXT COMMENT '扩展字段，存储额外的配置信息',
    PRIMARY KEY (`id`),
    KEY `idx_spydon_dictionary_items_dictionary_id` (`dictionary_id`) COMMENT '字典ID索引，用于查询某个字典的所有项'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据字典项表';


##
#email_address created_at updated_at
## email_template is_enabled