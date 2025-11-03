-- 回滚：重新添加外键约束

-- 重新添加 alerts 表的外键约束
ALTER TABLE alerts 
ADD CONSTRAINT fk_alerts_cluster 
FOREIGN KEY (cluster_id) REFERENCES clusters(cluster_id);

-- 重新添加 rca_runs 表的外键约束
ALTER TABLE rca_runs 
ADD CONSTRAINT fk_rca_runs_alert 
FOREIGN KEY (alert_id) REFERENCES alerts(id);
