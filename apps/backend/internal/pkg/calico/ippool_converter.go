package calico

import (
	"fmt"
	"strings"

	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
)

// IPPoolConverter handles conversion from Calico IPPool CRD to model.WayneIpPool
type IPPoolConverter struct{}

// NewIPPoolConverter creates a new IPPoolConverter instance
func NewIPPoolConverter() *IPPoolConverter {
	return &IPPoolConverter{}
}

// ConvertToModelIPPool converts a Calico IPPool (v3) to WayneIpPool
func (c *IPPoolConverter) ConvertToModelIPPool(calicoIPPool calicov3.IPPool, clusterName string) (*WayneIpPool, error) {
	if clusterName == "" {
		return nil, fmt.Errorf("cluster cannot be nil")
	}

	// Parse status from labels
	status := c.parseStatus(calicoIPPool.Labels)

	// Parse category from labels
	displayName := c.parseCategory(calicoIPPool.Labels)

	// Get display name based on category
	category := c.getCategoryCN(displayName)
	if displayName == "" {
		displayName = calicoIPPool.Name
	}

	// Parse node selector to labelKey and labelValue
	labelKey, labelValue := c.parseNodeSelector(calicoIPPool.Spec.NodeSelector)

	ipPool := &WayneIpPool{
		Name:        strings.TrimSpace(calicoIPPool.Name),
		DisplayName: strings.TrimSpace(displayName),
		Cidr:        strings.TrimSpace(calicoIPPool.Spec.CIDR),
		LabelKey:    strings.TrimSpace(labelKey),
		LabelValue:  strings.TrimSpace(labelValue),
		Status:      status,
		Category:    strings.TrimSpace(category),
		// User field will be populated by caller
	}

	return ipPool, nil
}

// parseStatus parses the status from labels
// kfeature.io/enabled: true = online (IPPoolStatusNormal), false = maintenance (IPPoolStatusMaintaining)
func (c *IPPoolConverter) parseStatus(labels map[string]string) IpPoolStatus {
	if labels == nil {
		return IpPoolStatusNormal
	}

	enabled, exists := labels["kfeature.io/enabled"]
	if !exists {
		return IpPoolStatusNormal
	}

	if strings.ToLower(enabled) == "false" {
		return IpPoolStatusMaintaining
	}

	return IpPoolStatusNormal
}

// parseCategory parses the category from labels
// kfeature.io/subfunction represents category, convert to uppercase
func (c *IPPoolConverter) parseCategory(labels map[string]string) string {
	if labels == nil {
		return "unknown"
	}

	subfunction, exists := labels["kfeature.io/subfunction"]
	if !exists || subfunction == "" {
		return "unknown"
	}

	return strings.ToUpper(subfunction)
}

// getCategoryCN returns the Chinese display name for a given category
func (c *IPPoolConverter) getCategoryCN(category string) string {
	categoryDisplayNames := map[string]string{
		"K8SAPPGENERAL": "K8SAPP通用",
		"K8SDB":         "K8S数据库",
		"K8SREDIS":      "K8SREDIS网段",
		"K8SBASE":       "K8S组件",
		"K8SBAPP":       "k8s业务APP",
		"K8SAPPCORE":    "K8SAPP核心",
		"K8SAPPSPECIAL": "K8SAPP专用",
		"K8SPLUS":       "K8S新核心专用",
	}

	if displayName, exists := categoryDisplayNames[category]; exists {
		return displayName
	}

	// If category not found, return category with "网络" suffix
	return category + "网络"
}

// parseNodeSelector parses the nodeSelector string to extract labelKey and labelValue
// Expects format like "key = 'value'" or "key = \"value\""
func (c *IPPoolConverter) parseNodeSelector(nodeSelector string) (string, string) {
	if nodeSelector == "" {
		return "", ""
	}

	if nodeSelector == "!all()" {
		return "", ""
	}

	// Parse nodeSelector format: "key = 'value'" or "key = \"value\""
	parts := strings.Split(nodeSelector, " = ")
	if len(parts) != 2 {
		return "", ""
	}

	labelKey := strings.TrimSpace(parts[0])
	labelValue := strings.TrimSpace(parts[1])

	// Remove quotes from value if present
	if len(labelValue) >= 2 {
		if (labelValue[0] == '\'' && labelValue[len(labelValue)-1] == '\'') ||
			(labelValue[0] == '"' && labelValue[len(labelValue)-1] == '"') {
			labelValue = labelValue[1 : len(labelValue)-1]
		}
	}

	return labelKey, labelValue
}

// ConvertBatchToModelIPpools converts multiple Calico IPPools to WayneIpPool slice
func (c *IPPoolConverter) ConvertBatchToModelIPpools(calicoIPpools []calicov3.IPPool, cluster string) ([]*WayneIpPool, error) {
	if len(calicoIPpools) == 0 {
		return []*WayneIpPool{}, nil
	}

	if cluster == "" {
		return nil, fmt.Errorf("cluster cannot be nil")
	}

	result := make([]*WayneIpPool, 0, len(calicoIPpools))

	for i, calicoIPPool := range calicoIPpools {
		ipPool, err := c.ConvertToModelIPPool(calicoIPPool, cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to convert IPPool at index %d: %w", i, err)
		}
		result = append(result, ipPool)
	}

	return result, nil
}
