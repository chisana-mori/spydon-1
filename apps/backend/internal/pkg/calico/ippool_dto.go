package calico

import (
	"robusta-web/backend/internal/models/navy"
	"time"
)

// IPPool represents the Calico IPPool CRD resource
type IPPool struct {
	Name         string         `json:"name"`         // pool名
	CIDR         string         `json:"cidr"`         // cidr
	NodeSelector string         `json:"nodeSelector"` // nodeSelector
	BlockSize    int            `json:"blockSize"`    // blockSize
	IPMode       string         `json:"ipMode"`       // ip模式开启状态
	VXLanMode    string         `json:"vxlanMode"`    // vxlan 模式
	CreatedAt    *navy.NavyTime `json:"createdAt"`    // 创建时间
	IsNat        bool           `json:"isNat"`        // 是否nat
	IPVersion    string         `json:"ipVersion"`    // ip版本 IPv4 or IPv6
	Category     string         `json:"category"`     // category
	CategoryCN   string         `json:"categoryCN"`   // categoryCN
}

// SyncRet represents the response from syncing IP pools
type SyncRet struct {
	Updated int64 `json:"updated"` // 创建个数
	Created int64 `json:"created"` // 更新个数
	Deleted int64 `json:"deleted"` // 删除个数
}

// WayneIpPool represents the internal model for Wayne IP pool
type WayneIpPool struct {
	Id          int64        `orm:"auto" json:"id,omitempty"`                                // 自增ID
	Category    string       `orm:"index;size(128)" json:"category,omitempty"`               // 分类
	Name        string       `orm:"index;size(128)" json:"name,omitempty"`                   // 名称
	DisplayName string       `orm:"size(128)" json:"displayName,omitempty"`                  // 显示名称
	LabelKey    string       `orm:"size(128)" json:"key,omitempty"`                          // 标签键
	LabelValue  string       `orm:"size(128)" json:"value,omitempty"`                        // 标签值
	User        string       `orm:"size(128)" json:"user,omitempty"`                         // 用户
	Env         string       `orm:"size(10)" json:"env,omitempty"`                           // 环境
	Status      IpPoolStatus `orm:"default(0)" json:"status"`                                // 状态
	CreateTime  *time.Time   `orm:"auto_now_add;type(datetime)" json:"createTime,omitempty"` // 创建时间
	UpdateTime  *time.Time   `orm:"auto_now;type(datetime)" json:"updateTime,omitempty"`     // 更新时间
	Cidr        string       `orm:"index;size(128)" json:"cidr,omitempty"`                   // CIDR
	ClusterId   int64        `orm:"-" json:"clusterId,omitempty"`                            // 集群ID
}

// IpPoolStatus represents the status of an IP pool
type IpPoolStatus int32

const (
	IpPoolStatusNormal      IpPoolStatus = 0 // 正常
	IpPoolStatusMaintaining IpPoolStatus = 1 // 维护中
)

// ListIPPoolResp represents the response for listing IP pools
type ListIPPoolResp struct {
	Data []IPPool `json:"data"`
}

// Constants for IP version
const (
	IPv4 string = "IPv4"
	IPv6 string = "IPv6"
)
