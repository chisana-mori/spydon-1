package services

// =====================================================
// Configuration DTO Definitions
// =====================================================

// LabelManagementDTO 标签管理响应
type LabelManagementDTO struct {
	ID     int      `json:"id"`
	Name   string   `json:"name"`
	Key    string   `json:"key"`
	Source int      `json:"source"`
	Status int      `json:"status"`
	Values []string `json:"values"`
}

// CreateLabelRequest 创建标签请求
type CreateLabelRequest struct {
	Name   string   `json:"name" binding:"required"`
	Key    string   `json:"key" binding:"required"`
	Source int      `json:"source"`
	Status int      `json:"status"`
	Values []string `json:"values"`
}

// UpdateLabelRequest 更新标签请求
type UpdateLabelRequest struct {
	Name   *string   `json:"name"`
	Key    *string   `json:"key"`
	Source *int      `json:"source"`
	Status *int      `json:"status"`
	Values *[]string `json:"values"`
}

// LabelListQuery 标签列表查询
type LabelListQuery struct {
	Keyword string `form:"keyword"`
	Source  *int   `form:"source"`
	Status  *int   `form:"status"`
	Page    int    `form:"page"`
	Size    int    `form:"size"`
}

// TaintManagementDTO 污点管理响应
type TaintManagementDTO struct {
	ID          int    `json:"id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Effect      string `json:"effect"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Status      int    `json:"status"`
}

// CreateTaintRequest 创建污点请求
type CreateTaintRequest struct {
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value"`
	Effect      string `json:"effect"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Status      int    `json:"status"`
}

// UpdateTaintRequest 更新污点请求
type UpdateTaintRequest struct {
	Key         *string `json:"key"`
	Value       *string `json:"value"`
	Effect      *string `json:"effect"`
	Description *string `json:"description"`
	Type        *string `json:"type"`
	Status      *int    `json:"status"`
}

// TaintListQuery 污点列表查询
type TaintListQuery struct {
	Keyword string `form:"keyword"`
	Effect  string `form:"effect"`
	Status  *int   `form:"status"`
	Page    int    `form:"page"`
	Size    int    `form:"size"`
}

// DeviceAppDTO 设备应用响应
type DeviceAppDTO struct {
	ID          int    `json:"id"`
	AppId       string `json:"app_id"`
	Type        int    `json:"type"`
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	Feature     string `json:"feature"`
	Description string `json:"description"`
	Status      int    `json:"status"`
}

// CreateDeviceAppRequest 创建设备应用请求
type CreateDeviceAppRequest struct {
	AppId       string `json:"app_id" binding:"required"`
	Type        int    `json:"type"`
	Name        string `json:"name" binding:"required"`
	Owner       string `json:"owner"`
	Feature     string `json:"feature"`
	Description string `json:"description"`
	Status      int    `json:"status"`
}

// UpdateDeviceAppRequest 更新设备应用请求
type UpdateDeviceAppRequest struct {
	AppId       *string `json:"app_id"`
	Type        *int    `json:"type"`
	Name        *string `json:"name"`
	Owner       *string `json:"owner"`
	Feature     *string `json:"feature"`
	Description *string `json:"description"`
	Status      *int    `json:"status"`
}

// DeviceAppListQuery 设备应用列表查询
type DeviceAppListQuery struct {
	Keyword string `form:"keyword"`
	Type    *int   `form:"type"`
	Status  *int   `form:"status"`
	Page    int    `form:"page"`
	Size    int    `form:"size"`
}

// ListResponse 通用列表响应
type ListResponse struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}
