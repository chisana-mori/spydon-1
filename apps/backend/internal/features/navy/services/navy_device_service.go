package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models/navy"
	"robusta-web/backend/internal/pkg/nodesync"
	"strings"
	"time"

	"gorm.io/gorm"
)

// NavyDeviceService Navy 设备管理服务
type NavyDeviceService struct {
	db              *db.Database
	nodesyncManager *nodesync.Manager
}

// NewNavyDeviceService 创建 Navy 设备管理服务
func NewNavyDeviceService(db *db.Database, nodesyncManager *nodesync.Manager) *NavyDeviceService {
	return &NavyDeviceService{
		db:              db,
		nodesyncManager: nodesyncManager,
	}
}

// ManagedLabelExists checks for managed labels
const ManagedLabelExists = "EXISTS (SELECT 1 FROM k8s_node_label knl JOIN label_feature lf ON knl.`key` = lf.`key` WHERE knl.node_id = kn.id)"

// ManagedTaintExists checks for managed taints
const ManagedTaintExists = "EXISTS (SELECT 1 FROM k8s_node_taint knt JOIN taint_feature tf ON knt.`key` = tf.`key` WHERE knt.node_id = kn.id)"

// SpecialDeviceCondition 特殊设备判断条件 (移植自 auto-navy)
// 满足以下任一条件即为特殊设备:
// 1. device.group != ”
// 2. 拥有受管理的 label 或 taint
// (不再仅仅因为关联了 k8s_node 就视为特殊设备)
const SpecialDeviceCondition = "device.`group` != '' OR " + ManagedLabelExists + " OR " + ManagedTaintExists

// buildDeviceBaseQuery 构建基础查询 (不含计数字段)
func (s *NavyDeviceService) buildDeviceBaseQuery(ctx context.Context) *gorm.DB {
	// 关联 k8s_node 以获取更多信息
	return s.db.WithContext(ctx).Table("device").
		Select("device.*, kn.role as k8s_role, kn.nodename as k8s_nodename").
		Joins("LEFT JOIN k8s_node kn ON LOWER(device.ip) = LOWER(kn.hostip)")
}

// buildDeviceQueryWithFeatures 构建带特性计数的查询
func (s *NavyDeviceService) buildDeviceQueryWithFeatures(ctx context.Context) *gorm.DB {
	// 计算特性数量: 关联的 label 和 taint 数量
	subqueryLabel := "(SELECT COUNT(*) FROM k8s_node_label knl2 JOIN k8s_node kn2 ON knl2.node_id = kn2.id WHERE LOWER(kn2.hostip) = LOWER(device.ip))"
	subqueryTaint := "(SELECT COUNT(*) FROM k8s_node_taint knt2 JOIN k8s_node kn3 ON knt2.node_id = kn3.id WHERE LOWER(kn3.hostip) = LOWER(device.ip))"

	isSpecialSQL := fmt.Sprintf("CASE WHEN device.`group` != '' OR %s OR %s THEN 1 ELSE 0 END as is_special", ManagedLabelExists, ManagedTaintExists)

	selectFields := fmt.Sprintf(`device.*,
		kn.role as k8s_role,
		kn.nodename as k8s_nodename,
		%s,
		(%s + %s) as feature_count`, isSpecialSQL, subqueryLabel, subqueryTaint)

	return s.db.WithContext(ctx).Table("device").
		Select(selectFields).
		Joins("LEFT JOIN k8s_node kn ON LOWER(device.ip) = LOWER(kn.hostip)")
}

// ListDevices 获取设备列表
func (s *NavyDeviceService) ListDevices(ctx context.Context, query *DeviceQuery) (*DeviceListResponse, error) {
	var models []navy.Device
	var total int64

	db := s.buildDeviceQueryWithFeatures(ctx)

	// 关键字搜索 - 支持多行输入
	if query.Keyword != "" {
		db = s.applyKeywordSearch(db, query.Keyword)
	}

	// 仅显示特殊设备
	if query.OnlySpecial {
		db = db.Where(SpecialDeviceCondition)
	}

	// 计数
	countDB := s.db.WithContext(ctx).Table("device").
		Joins("LEFT JOIN k8s_node kn ON LOWER(device.ip) = LOWER(kn.hostip)")
	if query.Keyword != "" {
		countDB = s.applyKeywordSearch(countDB, query.Keyword)
	}
	if query.OnlySpecial {
		countDB = countDB.Where(SpecialDeviceCondition)
	}
	if err := countDB.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := query.Page
	if page <= 0 {
		page = 1
	}
	size := query.Size
	if size <= 0 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	offset := (page - 1) * size

	if err := db.Offset(offset).Limit(size).Find(&models).Error; err != nil {
		return nil, err
	}

	return &DeviceListResponse{
		List:  s.convertToResponses(models),
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// applyKeywordSearch 应用关键字搜索（支持多行）
func (s *NavyDeviceService) applyKeywordSearch(db *gorm.DB, keyword string) *gorm.DB {
	// 检测是否是多行输入（包含换行符、逗号或OR）
	lines := strings.FieldsFunc(keyword, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	})

	if len(lines) > 1 {
		// 多行模式：每行都进行模糊匹配，结果取并集
		// 手动构建 SQL 字符串以避免 GORM 的 Where(func) 嵌套问题
		var conditions []string
		var args []interface{}

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// 单行匹配逻辑: IP LIKE %kw% OR CI LIKE %kw%
			conditions = append(conditions, "(device.ip LIKE ? OR device.ci_code LIKE ?)")
			kw := "%" + line + "%"
			args = append(args, kw, kw)
		}

		if len(conditions) > 0 {
			// 将所有单行条件用 OR 连接
			finalSQL := strings.Join(conditions, " OR ")
			return db.Where(finalSQL, args...)
		}
	}

	// 单行模式：模糊匹配 (包含集群字段)
	kw := "%" + strings.TrimSpace(keyword) + "%"
	return db.Where("device.ip LIKE ? OR device.ci_code LIKE ? OR device.cluster LIKE ?", kw, kw, kw)
}

// QueryDevices 复杂查询 - 支持完整的12种条件类型
func (s *NavyDeviceService) QueryDevices(ctx context.Context, req *NavyDeviceQueryRequest) (*DeviceListResponse, error) {
	var models []navy.Device
	var total int64

	db := s.buildDeviceQueryWithFeatures(ctx)

	// 应用筛选组
	db = s.applyFilterGroups(db, req.Groups)

	// 计数查询 (需要重新构建以避免 select 字段问题)
	countDB := s.db.WithContext(ctx).Table("device").
		Joins("LEFT JOIN k8s_node kn ON LOWER(device.ip) = LOWER(kn.hostip)")
	countDB = s.applyFilterGroups(countDB, req.Groups)
	if err := countDB.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := req.Page
	if page <= 0 {
		page = 1
	}
	size := req.Size
	if size <= 0 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	offset := (page - 1) * size

	if err := db.Offset(offset).Limit(size).Find(&models).Error; err != nil {
		return nil, err
	}

	return &DeviceListResponse{
		List:  s.convertToResponses(models),
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// applyFilterGroups 应用筛选组
func (s *NavyDeviceService) applyFilterGroups(db *gorm.DB, groups []FilterGroup) *gorm.DB {
	for i, group := range groups {
		// 过滤未激活的块
		activeBlocks := make([]FilterBlock, 0)
		for _, block := range group.Blocks {
			if block.IsActive == nil || *block.IsActive {
				activeBlocks = append(activeBlocks, block)
			}
		}
		if len(activeBlocks) == 0 {
			continue
		}

		// 构建组条件
		groupCondition := s.buildGroupCondition(activeBlocks)
		if groupCondition == nil {
			continue
		}

		// 应用组间逻辑
		if i == 0 || strings.ToLower(group.Operator) == "and" {
			db = db.Where(groupCondition)
		} else {
			db = db.Or(groupCondition)
		}
	}
	return db
}

// buildGroupCondition 构建单个组的条件
func (s *NavyDeviceService) buildGroupCondition(blocks []FilterBlock) *gorm.DB {
	if len(blocks) == 0 {
		return nil
	}

	// 使用内存数据库实例构建条件
	condition := s.db.DB

	for i, block := range blocks {
		blockCondition := s.buildBlockCondition(block)
		if blockCondition == nil {
			continue
		}

		if i == 0 {
			condition = condition.Where(blockCondition)
		} else if strings.ToLower(block.Operator) == "or" {
			condition = condition.Or(blockCondition)
		} else {
			condition = condition.Where(blockCondition)
		}
	}

	return condition
}

// buildBlockCondition 构建单个筛选块的条件
func (s *NavyDeviceService) buildBlockCondition(block FilterBlock) interface{} {
	switch block.Type {
	case FilterTypeDevice:
		return s.buildDeviceCondition(block)
	case FilterTypeNodeLabel:
		return s.buildLabelCondition(block)
	case FilterTypeTaint:
		return s.buildTaintCondition(block)
	default:
		return nil
	}
}

// buildDeviceCondition 构建设备字段条件
func (s *NavyDeviceService) buildDeviceCondition(block FilterBlock) interface{} {
	column := GetDeviceFieldColumn(block.Key)
	if column == "" {
		return nil
	}

	value := block.Value
	condType := ConditionType(block.ConditionType)

	switch condType {
	case ConditionEqual:
		return gorm.Expr(column+" = ?", value)
	case ConditionNotEqual:
		return gorm.Expr(column+" != ?", value)
	case ConditionContains:
		return gorm.Expr(column+" LIKE ?", fmt.Sprintf("%%%v%%", value))
	case ConditionNotContain:
		return gorm.Expr(column+" NOT LIKE ?", fmt.Sprintf("%%%v%%", value))
	case ConditionExists, ConditionIsNotEmpty:
		return gorm.Expr(column + " IS NOT NULL AND " + column + " != ''")
	case ConditionNotExists, ConditionIsEmpty:
		return gorm.Expr(column + " IS NULL OR " + column + " = ''")
	case ConditionIn:
		values := s.parseArrayValue(value)
		if len(values) == 0 {
			return nil
		}
		return gorm.Expr(column+" IN ?", values)
	case ConditionNotIn:
		values := s.parseArrayValue(value)
		if len(values) == 0 {
			return nil
		}
		return gorm.Expr(column+" NOT IN ?", values)
	case ConditionGT:
		return gorm.Expr(column+" > ?", value)
	case ConditionLT:
		return gorm.Expr(column+" < ?", value)
	default:
		return gorm.Expr(column+" = ?", value)
	}
}

// buildLabelCondition 构建标签条件
func (s *NavyDeviceService) buildLabelCondition(block FilterBlock) interface{} {
	key := block.Key
	value := block.Value
	condType := ConditionType(block.ConditionType)

	// 使用子查询匹配有相应标签的节点
	baseSubquery := "device.ip IN (SELECT LOWER(kn.hostip) FROM k8s_node kn JOIN k8s_node_label knl ON kn.id = knl.node_id WHERE "

	switch condType {
	case ConditionEqual:
		return gorm.Expr(baseSubquery+"knl.`key` = ? AND knl.value = ?)", key, value)
	case ConditionNotEqual:
		return gorm.Expr(baseSubquery+"knl.`key` = ? AND knl.value != ?)", key, value)
	case ConditionContains:
		return gorm.Expr(baseSubquery+"knl.`key` = ? AND knl.value LIKE ?)", key, fmt.Sprintf("%%%v%%", value))
	case ConditionExists:
		return gorm.Expr(baseSubquery + "knl.`key` = ?)" + fmt.Sprintf(" AND knl.`key` = '%s'", key))
	case ConditionNotExists:
		return gorm.Expr("device.ip NOT IN (SELECT LOWER(kn.hostip) FROM k8s_node kn JOIN k8s_node_label knl ON kn.id = knl.node_id WHERE knl.`key` = ?)", key)
	case ConditionIn:
		values := s.parseArrayValue(value)
		if len(values) == 0 {
			return nil
		}
		return gorm.Expr(baseSubquery+"knl.`key` = ? AND knl.value IN ?)", key, values)
	case ConditionNotIn:
		values := s.parseArrayValue(value)
		if len(values) == 0 {
			return nil
		}
		return gorm.Expr(baseSubquery+"knl.`key` = ? AND knl.value NOT IN ?)", key, values)
	case ConditionNotContain:
		return gorm.Expr(baseSubquery+"knl.`key` = ? AND knl.value NOT LIKE ?)", key, fmt.Sprintf("%%%v%%", value))
	case ConditionGT:
		return gorm.Expr(baseSubquery+"knl.`key` = ? AND knl.value > ?)", key, value)
	case ConditionLT:
		return gorm.Expr(baseSubquery+"knl.`key` = ? AND knl.value < ?)", key, value)
	case ConditionIsNotEmpty:
		return gorm.Expr(baseSubquery+"knl.`key` = ? AND knl.value IS NOT NULL AND knl.value != '')", key)
	case ConditionIsEmpty:
		return gorm.Expr("device.ip NOT IN (SELECT LOWER(kn.hostip) FROM k8s_node kn JOIN k8s_node_label knl ON kn.id = knl.node_id WHERE knl.`key` = ?)", key)
	default:
		return gorm.Expr(baseSubquery+"knl.`key` = ? AND knl.value = ?)", key, value)
	}
}

// buildTaintCondition 构建污点条件
func (s *NavyDeviceService) buildTaintCondition(block FilterBlock) interface{} {
	key := block.Key
	value := block.Value
	condType := ConditionType(block.ConditionType)

	baseSubquery := "device.ip IN (SELECT LOWER(kn.hostip) FROM k8s_node kn JOIN k8s_node_taint knt ON kn.id = knt.node_id WHERE "

	switch condType {
	case ConditionEqual:
		return gorm.Expr(baseSubquery+"knt.`key` = ? AND knt.value = ?)", key, value)
	case ConditionNotEqual:
		return gorm.Expr(baseSubquery+"knt.`key` = ? AND knt.value != ?)", key, value)
	case ConditionContains:
		return gorm.Expr(baseSubquery+"knt.`key` = ? AND knt.value LIKE ?)", key, fmt.Sprintf("%%%v%%", value))
	case ConditionExists:
		return gorm.Expr(baseSubquery+"knt.`key` = ?)", key)
	case ConditionNotExists:
		return gorm.Expr("device.ip NOT IN (SELECT LOWER(kn.hostip) FROM k8s_node kn JOIN k8s_node_taint knt ON kn.id = knt.node_id WHERE knt.`key` = ?)", key)
	case ConditionIn:
		values := s.parseArrayValue(value)
		if len(values) == 0 {
			return nil
		}
		return gorm.Expr(baseSubquery+"knt.`key` = ? AND knt.value IN ?)", key, values)
	case ConditionNotIn:
		values := s.parseArrayValue(value)
		if len(values) == 0 {
			return nil
		}
		return gorm.Expr(baseSubquery+"knt.`key` = ? AND knt.value NOT IN ?)", key, values)
	case ConditionNotContain:
		return gorm.Expr(baseSubquery+"knt.`key` = ? AND knt.value NOT LIKE ?)", key, fmt.Sprintf("%%%v%%", value))
	case ConditionGT:
		return gorm.Expr(baseSubquery+"knt.`key` = ? AND knt.value > ?)", key, value)
	case ConditionLT:
		return gorm.Expr(baseSubquery+"knt.`key` = ? AND knt.value < ?)", key, value)
	case ConditionIsNotEmpty:
		return gorm.Expr(baseSubquery+"knt.`key` = ? AND knt.value IS NOT NULL AND knt.value != '')", key)
	case ConditionIsEmpty:
		return gorm.Expr("device.ip NOT IN (SELECT LOWER(kn.hostip) FROM k8s_node kn JOIN k8s_node_taint knt ON kn.id = knt.node_id WHERE knt.`key` = ?)", key)
	default:
		return gorm.Expr(baseSubquery+"knt.`key` = ?)", key)
	}
}

// parseArrayValue 解析数组值
func (s *NavyDeviceService) parseArrayValue(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case string:
		// 尝试解析为JSON数组
		var arr []string
		if err := json.Unmarshal([]byte(v), &arr); err == nil {
			return arr
		}
		// 否则按分隔符拆分
		parts := strings.FieldsFunc(v, func(r rune) bool {
			return r == ',' || r == ';' || r == '\n'
		})
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		return result
	default:
		return nil
	}
}

// GetFilterOptions 获取筛选选项
func (s *NavyDeviceService) GetFilterOptions(ctx context.Context) (*FilterOptionsResponse, error) {
	response := &FilterOptionsResponse{
		DeviceFields:      GetAllDeviceFields(),
		DeviceFieldValues: make([]DeviceFieldValues, 0),
		LabelKeys:         make([]string, 0),
		TaintKeys:         make([]string, 0),
	}

	// 获取常用字段的预选值
	fieldsToPreload := []string{"idc", "status", "cluster", "infraType", "archType", "netZone"}
	for _, field := range fieldsToPreload {
		column := GetDeviceFieldColumn(field)
		if column == "" {
			continue
		}
		// 提取并转义列名（去掉表前缀）
		parts := strings.Split(column, ".")
		colName := parts[len(parts)-1]
		if !strings.HasPrefix(colName, "`") {
			colName = "`" + colName + "`"
		}

		var values []string
		if err := s.db.WithContext(ctx).Table("device").
			Distinct(colName).
			Where(colName+" IS NOT NULL AND "+colName+" != ''").
			Order(colName).
			Limit(100).
			Pluck(colName, &values).Error; err == nil {
			opts := make([]FilterOption, len(values))
			for i, v := range values {
				opts[i] = FilterOption{ID: v, Label: v, Value: v}
			}
			response.DeviceFieldValues = append(response.DeviceFieldValues, DeviceFieldValues{Field: field, Values: opts})
		}
	}

	// 获取受管理的标签键列表（从 label_feature 表）
	if err := s.db.WithContext(ctx).Table("label_feature").
		Distinct("`key`").
		Order("`key`").
		Limit(200).
		Pluck("`key`", &response.LabelKeys).Error; err != nil {
		// 忽略错误，返回空列表
	}

	// 获取受管理的污点键列表（从 taint_feature 表）
	if err := s.db.WithContext(ctx).Table("taint_feature").
		Distinct("`key`").
		Order("`key`").
		Limit(100).
		Pluck("`key`", &response.TaintKeys).Error; err != nil {
		// 忽略错误，返回空列表
	}

	return response, nil
}

// GetLabelValues 获取指定标签的所有值
func (s *NavyDeviceService) GetLabelValues(ctx context.Context, labelKey string) ([]string, error) {
	var values []string
	err := s.db.WithContext(ctx).Table("k8s_node_label").
		Distinct("value").
		Where("`key` = ?", labelKey).
		Order("value").
		Limit(500).
		Pluck("value", &values).Error
	return values, err
}

// GetTaintValues 获取指定污点的所有值
func (s *NavyDeviceService) GetTaintValues(ctx context.Context, taintKey string) ([]TaintValue, error) {
	var results []TaintValue
	err := s.db.WithContext(ctx).Table("k8s_node_taint").
		Select("DISTINCT value, effect").
		Where("`key` = ?", taintKey).
		Order("value").
		Limit(100).
		Find(&results).Error
	return results, err
}

// GetDeviceFieldValues 获取设备字段的所有值
func (s *NavyDeviceService) GetDeviceFieldValues(ctx context.Context, field string) ([]string, error) {
	column := GetDeviceFieldColumn(field)
	if column == "" {
		return nil, fmt.Errorf("unknown field: %s", field)
	}

	// 提取并转义列名
	parts := strings.Split(column, ".")
	colName := parts[len(parts)-1]
	if !strings.HasPrefix(colName, "`") {
		colName = "`" + colName + "`"
	}

	var values []string
	err := s.db.WithContext(ctx).Table("device").
		Distinct(colName).
		Where(colName+" IS NOT NULL AND "+colName+" != ''").
		Order(colName).
		Limit(500).
		Pluck(colName, &values).Error
	return values, err
}

// GetDevice 获取单个设备
func (s *NavyDeviceService) GetDevice(ctx context.Context, id int) (*DeviceResponse, error) {
	var m navy.Device
	err := s.buildDeviceQueryWithFeatures(ctx).Where("device.id = ?", id).First(&m).Error
	if err != nil {
		return nil, err
	}

	res := s.convertToResponse(m)
	return &res, nil
}

// UpdateDeviceRole 更新角色
func (s *NavyDeviceService) UpdateDeviceRole(ctx context.Context, id int, role string) error {
	return s.db.WithContext(ctx).Model(&navy.Device{}).Where("id = ?", id).Update("role", role).Error
}

// UpdateDeviceGroup 更新用途/组
func (s *NavyDeviceService) UpdateDeviceGroup(ctx context.Context, id int, group string) error {
	return s.db.WithContext(ctx).Model(&navy.Device{}).Where("id = ?", id).Update("`group`", group).Error
}

// ExportDevices 导出为 CSV
func (s *NavyDeviceService) ExportDevices(ctx context.Context) ([]byte, error) {
	var devices []navy.Device
	if err := s.buildDeviceBaseQuery(ctx).Find(&devices).Error; err != nil {
		return nil, err
	}

	buffer := &bytes.Buffer{}
	// 添加 UTF-8 BOM 以便 Excel 正确识别中文
	buffer.Write([]byte{0xEF, 0xBB, 0xBF})

	headers := []string{"设备编码", "IP地址", "CPU架构", "IDC", "机房", "机柜", "状态", "角色", "集群", "分组", "厂商", "型号", "CPU", "内存"}
	buffer.WriteString(strings.Join(headers, ",") + "\n")

	for _, d := range devices {
		row := []string{
			d.CICode, d.IP, d.ArchType, d.IDC, d.Room, d.Cabinet, d.Status, d.Role, d.Cluster,
			d.Group, d.Company, d.Model, fmt.Sprintf("%.0f", d.CPU), fmt.Sprintf("%.0f", d.Memory),
		}
		buffer.WriteString(strings.Join(row, ",") + "\n")
	}

	return buffer.Bytes(), nil
}

// 内部转换辅助函数
func (s *NavyDeviceService) convertToResponses(models []navy.Device) []DeviceResponse {
	res := make([]DeviceResponse, len(models))
	for i, m := range models {
		res[i] = s.convertToResponse(m)
	}
	return res
}

func (s *NavyDeviceService) convertToResponse(m navy.Device) DeviceResponse {
	return DeviceResponse{
		ID:             m.ID,
		CICode:         m.CICode,
		IP:             m.IP,
		ArchType:       m.ArchType,
		IDC:            m.IDC,
		Room:           m.Room,
		Cabinet:        m.Cabinet,
		CabinetNO:      m.CabinetNO,
		InfraType:      m.InfraType,
		IsLocalization: m.IsLocalization,
		NetZone:        m.NetZone,
		Group:          m.Group,
		AppID:          m.AppID,
		AppName:        m.AppName,
		OsCreateTime:   m.OsCreateTime,
		CPU:            m.CPU,
		Memory:         m.Memory,
		Model:          m.Model,
		KvmIP:          m.KvmIP,
		OS:             m.OS,
		Company:        m.Company,
		OSName:         m.OSName,
		OSIssue:        m.OSIssue,
		OSKernel:       m.OSKernel,
		Status:         m.Status,
		Role:           m.Role,
		Cluster:        m.Cluster,
		ClusterID:      m.ClusterID,
		K8sStatus:      m.K8sStatus,
		AcceptanceTime: m.AcceptanceTime,
		DiskCount:      m.DiskCount,
		DiskDetail:     m.DiskDetail,
		NetworkSpeed:   m.NetworkSpeed,
		IsSpecial:      m.IsSpecial,
		FeatureCount:   m.FeatureCount,
		CreatedAt:      time.Time(m.CreatedAt),
		UpdatedAt:      time.Time(m.UpdatedAt),
	}
}

// DeviceFeaturesRequest 批量获取设备特性请求
type DeviceFeaturesRequest struct {
	CICodes []string `json:"ci_codes"`
}

// AggregatedLabel 聚合后的 Label
type AggregatedLabel struct {
	Key   string   `json:"key"`
	Value string   `json:"value"`
	Nodes []string `json:"nodes"` // 拥有此 Label 的节点名称列表
}

// AggregatedTaint 聚合后的 Taint
type AggregatedTaint struct {
	Key    string   `json:"key"`
	Value  string   `json:"value"`
	Effect string   `json:"effect"`
	Nodes  []string `json:"nodes"` // 拥有此 Taint 的节点名称列表
}

// GetBatchDeviceFeatures 批量获取设备特性（受管理的 Labels 和 Taints）
// 移植自 auto-navy，只返回在 label_feature 和 taint_feature 表中定义的"受管理特性"
func (s *NavyDeviceService) GetBatchDeviceFeatures(ctx context.Context, ciCodes []string) (map[string]interface{}, error) {
	if len(ciCodes) == 0 {
		return map[string]interface{}{
			"labels": []LabelValue{},
			"taints": []TaintValue{},
		}, nil
	}

	// 对于单个设备，使用优化的 UNION ALL 查询（与 auto-navy 一致）
	if len(ciCodes) == 1 {
		return s.getDeviceFeatureDetails(ctx, ciCodes[0])
	}

	// 对于多个设备，聚合查询
	return s.getBatchDeviceFeaturesAggregated(ctx, ciCodes)
}

// getDeviceFeatureDetails 获取单个设备的特性详情（与 auto-navy 的 GetDeviceFeatureDetails 一致）
func (s *NavyDeviceService) getDeviceFeatureDetails(ctx context.Context, ciCode string) (map[string]interface{}, error) {
	// 定义结果结构体
	type FeatureResult struct {
		Type   string `gorm:"column:type"`
		Key    string `gorm:"column:key"`
		Value  string `gorm:"column:value"`
		Effect string `gorm:"column:effect"`
	}

	var results []FeatureResult

	// 构建 UNION ALL 查询
	// 注意：robusta-web 使用 hostip 关联，而 auto-navy 使用 nodename
	// 这里同时支持两种关联方式
	query := `
		WITH node_id AS (
			SELECT id FROM k8s_node
			WHERE LOWER(hostip) = LOWER(?) OR LOWER(nodename) = LOWER(?)
			LIMIT 1
		)

		SELECT 'label' as type, knl.` + "`key`" + ` as ` + "`key`" + `, knl.value as value, '' as effect
		FROM node_id n
		JOIN k8s_node_label knl ON n.id = knl.node_id
		JOIN label_feature lf ON knl.` + "`key`" + ` = lf.` + "`key`" + `

		UNION ALL

		SELECT 'taint' as type, knt.` + "`key`" + ` as ` + "`key`" + `, knt.value as value, knt.effect as effect
		FROM node_id n
		JOIN k8s_node_taint knt ON n.id = knt.node_id
		JOIN taint_feature tf ON knt.` + "`key`" + ` = tf.` + "`key`" + `
	`

	// 执行查询
	if err := s.db.WithContext(ctx).Raw(query, ciCode, ciCode).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("查询设备特性失败: %w", err)
	}

	// 分类结果
	labels := make([]LabelValue, 0)
	taints := make([]TaintValue, 0)

	for _, r := range results {
		if r.Type == "label" {
			labels = append(labels, LabelValue{
				Key:   r.Key,
				Value: r.Value,
			})
		} else {
			taints = append(taints, TaintValue{
				Key:    r.Key,
				Value:  r.Value,
				Effect: r.Effect,
			})
		}
	}

	return map[string]interface{}{
		"labels": labels,
		"taints": taints,
	}, nil
}

// getBatchDeviceFeaturesAggregated 批量获取设备特性（聚合模式，用于多个设备）
func (s *NavyDeviceService) getBatchDeviceFeaturesAggregated(ctx context.Context, ciCodes []string) (map[string]interface{}, error) {
	// 定义结果结构体
	type FeatureResult struct {
		NodeName string `gorm:"column:nodename"`
		Type     string `gorm:"column:type"`
		Key      string `gorm:"column:key"`
		Value    string `gorm:"column:value"`
		Effect   string `gorm:"column:effect"`
	}

	var results []FeatureResult

	// 构建批量查询
	query := `
		SELECT kn.nodename, 'label' as type, knl.` + "`key`" + ` as ` + "`key`" + `, knl.value as value, '' as effect
		FROM k8s_node kn
		JOIN k8s_node_label knl ON kn.id = knl.node_id
		JOIN label_feature lf ON knl.` + "`key`" + ` = lf.` + "`key`" + `
		WHERE LOWER(kn.hostip) IN (?) OR LOWER(kn.nodename) IN (?)

		UNION ALL

		SELECT kn.nodename, 'taint' as type, knt.` + "`key`" + ` as ` + "`key`" + `, knt.value as value, knt.effect as effect
		FROM k8s_node kn
		JOIN k8s_node_taint knt ON kn.id = knt.node_id
		JOIN taint_feature tf ON knt.` + "`key`" + ` = tf.` + "`key`" + `
		WHERE LOWER(kn.hostip) IN (?) OR LOWER(kn.nodename) IN (?)
	`

	// 转换为小写以进行不区分大小写的匹配
	lowerCodes := make([]string, len(ciCodes))
	for i, c := range ciCodes {
		lowerCodes[i] = strings.ToLower(c)
	}

	if err := s.db.WithContext(ctx).Raw(query, lowerCodes, lowerCodes, lowerCodes, lowerCodes).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("查询设备特性失败: %w", err)
	}

	// 聚合 Labels
	labelMap := make(map[string]*AggregatedLabel)
	taintMap := make(map[string]*AggregatedTaint)

	for _, r := range results {
		if r.Type == "label" {
			mapKey := fmt.Sprintf("%s=%s", r.Key, r.Value)
			if _, exists := labelMap[mapKey]; !exists {
				labelMap[mapKey] = &AggregatedLabel{
					Key:   r.Key,
					Value: r.Value,
					Nodes: []string{},
				}
			}
			labelMap[mapKey].Nodes = append(labelMap[mapKey].Nodes, r.NodeName)
		} else {
			mapKey := fmt.Sprintf("%s=%s:%s", r.Key, r.Value, r.Effect)
			if _, exists := taintMap[mapKey]; !exists {
				taintMap[mapKey] = &AggregatedTaint{
					Key:    r.Key,
					Value:  r.Value,
					Effect: r.Effect,
					Nodes:  []string{},
				}
			}
			taintMap[mapKey].Nodes = append(taintMap[mapKey].Nodes, r.NodeName)
		}
	}

	// 转换为列表
	labels := make([]AggregatedLabel, 0, len(labelMap))
	for _, l := range labelMap {
		labels = append(labels, *l)
	}

	taints := make([]AggregatedTaint, 0, len(taintMap))
	for _, t := range taintMap {
		taints = append(taints, *t)
	}

	return map[string]interface{}{
		"labels": labels,
		"taints": taints,
	}, nil
}

// SaveQueryTemplate 保存查询模板
func (s *NavyDeviceService) SaveQueryTemplate(ctx context.Context, template *QueryTemplate) error {
	groupsJSON, err := json.Marshal(template.Groups)
	if err != nil {
		return err
	}

	m := navy.QueryTemplate{
		Name:        template.Name,
		Description: template.Description,
		Groups:      string(groupsJSON),
	}
	m.ID = template.ID

	if m.ID > 0 {
		return s.db.WithContext(ctx).Save(&m).Error
	}
	return s.db.WithContext(ctx).Create(&m).Error
}

// GetQueryTemplates 获取模板列表
func (s *NavyDeviceService) GetQueryTemplates(ctx context.Context, page, size int) (*QueryTemplateListResponse, error) {
	var models []navy.QueryTemplate
	var total int64

	db := s.db.WithContext(ctx).Model(&navy.QueryTemplate{})
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	offset := (page - 1) * size

	if err := db.Order("updated_at DESC").Offset(offset).Limit(size).Find(&models).Error; err != nil {
		return nil, err
	}

	list := make([]QueryTemplate, len(models))
	for i, m := range models {
		var groups []FilterGroup
		_ = json.Unmarshal([]byte(m.Groups), &groups)
		list[i] = QueryTemplate{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Groups:      groups,
			CreatedAt:   time.Time(m.CreatedAt),
			UpdatedAt:   time.Time(m.UpdatedAt),
		}
	}

	return &QueryTemplateListResponse{
		List:  list,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// GetQueryTemplate 获取单个模板
func (s *NavyDeviceService) GetQueryTemplate(ctx context.Context, id int) (*QueryTemplate, error) {
	var m navy.QueryTemplate
	if err := s.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}

	var groups []FilterGroup
	_ = json.Unmarshal([]byte(m.Groups), &groups)

	return &QueryTemplate{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Groups:      groups,
		CreatedAt:   time.Time(m.CreatedAt),
		UpdatedAt:   time.Time(m.UpdatedAt),
	}, nil
}

// DeleteQueryTemplate 删除模板
func (s *NavyDeviceService) DeleteQueryTemplate(ctx context.Context, id int) error {
	return s.db.WithContext(ctx).Delete(&navy.QueryTemplate{}, id).Error
}
