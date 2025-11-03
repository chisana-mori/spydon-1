-- 移除外键约束，避免在数据库层面显示外键错误
-- 这样可以让应用层更灵活地处理数据关系

-- 移除 alerts 表的外键约束
ALTER TABLE alerts DROP CONSTRAINT IF EXISTS fk_alerts_cluster;

-- 移除 rca_runs 表的外键约束
ALTER TABLE rca_runs DROP CONSTRAINT IF EXISTS fk_rca_runs_alert;

-- 注意：索引仍然保留，以保证查询性能
-- 数据完整性将由应用层代码保证
