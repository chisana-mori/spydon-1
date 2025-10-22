-- 删除触发器
DROP TRIGGER IF EXISTS update_clusters_updated_at ON clusters;
DROP TRIGGER IF EXISTS update_alerts_updated_at ON alerts;
DROP TRIGGER IF EXISTS update_rca_runs_updated_at ON rca_runs;
DROP TRIGGER IF EXISTS update_audit_logs_updated_at ON audit_logs;

-- 删除触发器函数
DROP FUNCTION IF EXISTS update_updated_at_column();

-- 删除索引
DROP INDEX IF EXISTS idx_clusters_deleted_at;
DROP INDEX IF EXISTS idx_clusters_status;
DROP INDEX IF EXISTS idx_clusters_cluster_id;

DROP INDEX IF EXISTS idx_audit_logs_deleted_at;
DROP INDEX IF EXISTS idx_audit_logs_user_action;

DROP INDEX IF EXISTS idx_rca_runs_deleted_at;
DROP INDEX IF EXISTS idx_rca_runs_alert_status;

DROP INDEX IF EXISTS idx_alerts_deleted_at;
DROP INDEX IF EXISTS idx_alerts_status_created;
DROP INDEX IF EXISTS idx_alerts_cluster_severity;
DROP INDEX IF EXISTS idx_alerts_fingerprint_cluster;

-- 删除表（注意外键约束的顺序）
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS rca_runs;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS clusters;

-- 删除扩展（可选，因为可能被其他应用使用）
-- DROP EXTENSION IF EXISTS "pgcrypto";
-- DROP EXTENSION IF EXISTS "uuid-ossp";
