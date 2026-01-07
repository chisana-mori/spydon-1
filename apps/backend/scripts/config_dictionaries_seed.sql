-- 配置管理字典初始化脚本
-- 运行此脚本以插入枚举类型字典

-- 1. 标签来源 (label_source)
INSERT IGNORE INTO spydon_dictionaries (code, name, module, description, is_enabled, key_same_as_value, sort_order)
VALUES ('label_source', '标签来源', 'configuration', '标签管理的来源类型',  true, false, 10);

SET @dict_label_source_id = (SELECT id FROM spydon_dictionaries WHERE code = 'label_source' LIMIT 1);

INSERT IGNORE INTO spydon_dictionary_items (dictionary_id, `key`, value, description, is_default, is_enabled, sort_order)
VALUES
(@dict_label_source_id, '0', '内部', '内部定义的标签', true, true, 1),
(@dict_label_source_id, '1', '外部', '外部导入的标签', false, true, 2),
(@dict_label_source_id, '2', '其他', '其他来源的标签', false, true, 3);

-- 2. 标签状态 (label_status)
INSERT IGNORE INTO spydon_dictionaries (code, name, module, description, is_enabled, key_same_as_value, sort_order)
VALUES ('label_status', '标签状态', 'configuration', '标签的启用状态', true, false, 11);

SET @dict_label_status_id = (SELECT id FROM spydon_dictionaries WHERE code = 'label_status' LIMIT 1);

INSERT IGNORE INTO spydon_dictionary_items (dictionary_id, `key`, value, description, is_default, is_enabled, sort_order)
VALUES
(@dict_label_status_id, '0', '正常', '标签正常启用', true, true, 1),
(@dict_label_status_id, '1', '停用', '标签已停用', false, true, 2);

-- 3. 污点状态 (taint_status)
INSERT IGNORE INTO spydon_dictionaries (code, name, module, description, is_enabled, key_same_as_value, sort_order)
VALUES ('taint_status', '污点状态', 'configuration', '污点的启用状态', true, false, 12);

SET @dict_taint_status_id = (SELECT id FROM spydon_dictionaries WHERE code = 'taint_status' LIMIT 1);

INSERT IGNORE INTO spydon_dictionary_items (dictionary_id, `key`, value, description, is_default, is_enabled, sort_order)
VALUES
(@dict_taint_status_id, '0', '正常', '污点正常启用', true, true, 1),
(@dict_taint_status_id, '1', '停用', '污点已停用', false, true, 2);

-- 4. 设备应用类型 (device_type)
INSERT IGNORE INTO spydon_dictionaries (code, name, module, description, is_enabled, key_same_as_value, sort_order)
VALUES ('device_type', '设备应用类型', 'configuration', '设备应用的分类', true, false, 13);

SET @dict_device_type_id = (SELECT id FROM spydon_dictionaries WHERE code = 'device_type' LIMIT 1);

INSERT IGNORE INTO spydon_dictionary_items (dictionary_id, `key`, value, description, is_default, is_enabled, sort_order)
VALUES
(@dict_device_type_id, '0', '设备', '设备类型应用', true, true, 1),
(@dict_device_type_id, '1', '组件', '组件类型应用', false, true, 2);

-- 5. 设备应用状态 (device_status)
INSERT IGNORE INTO spydon_dictionaries (code, name, module, description, is_enabled, key_same_as_value, sort_order)
VALUES ('device_status', '设备应用状态', 'configuration', '设备应用的采集状态', true, false, 14);

SET @dict_device_status_id = (SELECT id FROM spydon_dictionaries WHERE code = 'device_status' LIMIT 1);

INSERT IGNORE INTO spydon_dictionary_items (dictionary_id, `key`, value, description, is_default, is_enabled, sort_order)
VALUES
(@dict_device_status_id, '0', '收集', '正在收集数据', true, true, 1),
(@dict_device_status_id, '1', '停止收集', '已停止收集数据', false, true, 2);
