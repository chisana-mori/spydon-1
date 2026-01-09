
--
-- Table structure for table `clusters`
-- 集群表
CREATE TABLE `clusters` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `name` varchar(100) NOT NULL COMMENT '集群名称，唯一索引',
  `description` text COMMENT '集群描述',
  `config` text COMMENT '集群配置，敏感信息会加密存储',
  `prometheus_url` varchar(255) DEFAULT NULL COMMENT 'Prometheus的URL',
  `in_cluster` tinyint(1) DEFAULT '0' COMMENT '是否Kite运行在此集群内',
  `is_default` tinyint(1) DEFAULT '0' COMMENT '是否是默认集群',
  `enable` tinyint(1) DEFAULT '1' COMMENT '是否启用',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_clusters_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='集群配置表';

--
-- Table structure for table `oauth_providers`
-- OAuth提供者表
CREATE TABLE `oauth_providers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `name` varchar(100) NOT NULL COMMENT 'OAuth提供者名称，唯一索引，存储为小写',
  `client_id` varchar(255) NOT NULL COMMENT '客户端ID',
  `client_secret` text NOT NULL COMMENT '客户端Secret，敏感信息会加密存储',
  `auth_url` varchar(255) DEFAULT NULL COMMENT '授权URL',
  `token_url` varchar(255) DEFAULT NULL COMMENT 'Token URL',
  `user_info_url` varchar(255) DEFAULT NULL COMMENT '用户信息URL',
  `scopes` varchar(255) DEFAULT 'openid,profile,email' COMMENT 'OAuth作用域',
  `issuer` varchar(255) DEFAULT NULL COMMENT '颁发者',
  `enabled` tinyint(1) DEFAULT '1' COMMENT '是否启用',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_oauth_providers_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='OAuth身份验证提供者配置表';

--
-- Table structure for table `roles`
-- 角色表
CREATE TABLE `roles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `name` varchar(100) NOT NULL COMMENT '角色名称，唯一索引',
  `description` text COMMENT '角色描述',
  `is_system` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为系统预设角色',
  `clusters` text COMMENT '集群列表，逗号分隔，如"*,cluster1"',
  `resources` text COMMENT '资源列表，逗号分隔，如"*,deployment"',
  `namespaces` text COMMENT '命名空间列表，逗号分隔，如"*,default"',
  `verbs` text COMMENT '操作动词列表，逗号分隔，如"*,get,list"',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_roles_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='RBAC角色定义表';

--
-- Table structure for table `role_assignments`
-- 角色分配表
CREATE TABLE `role_assignments` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
  `subject_type` varchar(20) NOT NULL COMMENT '主体类型（user或group）',
  `subject` varchar(255) NOT NULL COMMENT '主体名称（用户名或OIDC组名）',
  PRIMARY KEY (`id`),
  KEY `idx_role_assignments_role_id` (`role_id`),
  CONSTRAINT `fk_roles_assignments` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='RBAC角色与主体分配表';

--
-- Table structure for table `users`
-- 用户表
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `username` varchar(50) NOT NULL COMMENT '用户名，唯一索引',
  `password` varchar(255) DEFAULT NULL COMMENT '用户密码，加密存储',
  `name` varchar(100) DEFAULT NULL COMMENT '用户显示名称',
  `avatar_url` varchar(500) DEFAULT NULL COMMENT '用户头像URL',
  `provider` varchar(50) DEFAULT 'password' COMMENT '认证提供者（password, github等）',
  `oidc_groups` text COMMENT 'OIDC用户组，逗号分隔',
  `last_login_at` datetime(3) DEFAULT NULL COMMENT '上次登录时间',
  `enabled` tinyint(1) DEFAULT '1' COMMENT '是否启用用户',
  `sub` varchar(255) DEFAULT NULL COMMENT 'OIDC Subject ID',
  `api_key` text COMMENT 'API Key，敏感信息会加密存储',
  `sidebar_preference` text COMMENT '侧边栏偏好设置',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_users_username` (`username`),
  KEY `idx_users_sub` (`sub`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='系统用户表';

--
-- Table structure for table `resource_histories`
-- 资源操作历史表
CREATE TABLE `resource_histories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `cluster_name` varchar(100) NOT NULL COMMENT '集群名称',
  `resource_type` varchar(50) NOT NULL COMMENT '资源类型，如Deployment, Pod',
  `resource_name` varchar(255) NOT NULL COMMENT '资源名称',
  `namespace` varchar(100) DEFAULT NULL COMMENT '资源所在命名空间',
  `operation_type` varchar(50) NOT NULL COMMENT '操作类型，如Create, Update, Delete',
  `resource_yaml` text COMMENT '操作后的资源YAML',
  `previous_yaml` text COMMENT '操作前的资源YAML',
  `success` tinyint(1) DEFAULT NULL COMMENT '操作是否成功',
  `error_message` text COMMENT '错误信息',
  `operator_id` bigint unsigned NOT NULL COMMENT '操作者用户ID',
  PRIMARY KEY (`id`),
  KEY `idx_resource_histories_lookup_with_time` (`cluster_name`,`resource_type`,`resource_name`,`namespace`,`created_at` DESC),
  KEY `idx_resource_histories_operation_type` (`operation_type`),
  KEY `fk_users_resource_histories` (`operator_id`),
  CONSTRAINT `fk_users_resource_histories` FOREIGN KEY (`operator_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Kubernetes资源操作历史记录表';


--
-- Table structure for table `resource_templates`
-- 资源模板表
CREATE TABLE `resource_templates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `name` varchar(255) NOT NULL COMMENT '模板名称，唯一索引',
  `description` text COMMENT '模板描述',
  `yaml` text COMMENT '模板的YAML内容',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_resource_templates_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Kubernetes资源模板存储表';

--
-- Default data for `roles`
--
INSERT IGNORE INTO `roles` (`id`, `created_at`, `updated_at`, `name`, `description`, `is_system`, `clusters`, `resources`, `namespaces`, `verbs`) VALUES
(1, NOW(), NOW(), 'admin', 'Administrator role with full access', 1, '*', '*', '*', '*'),
(2, NOW(), NOW(), 'viewer', 'Viewer role with read-only access', 1, '*', '*', '*', 'get,log');
