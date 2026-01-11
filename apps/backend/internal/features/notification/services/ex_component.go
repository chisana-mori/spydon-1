package services

import (
	"robusta-web/backend/internal/pkg/nodesync"
	"robusta-web/backend/pkg/errs"
	"strings"
)

const (
	Deployment string = "deployment"
	DaemonSet  string = "daemonSet"

	IngressControllerNS string = "ingress-nginx"
	HigressControllerNS string = "higress-system"
	HaloNS              string = "fps-solar"
	CatNs               string = "itm-cat"
	JmxNs               string = "pdom-dpms"
	HiveNS              string = "isom-bms"
	IstioNS             string = "istio-system"
	AstaNS              string = "fps-asta"
	ArmadaNS            string = "sdos-nova"
)

// ExComponentService 扩展组件检查服务
type ExComponentService struct {
	nodeManager *nodesync.Manager
}

// NewExComponentService 创建扩展组件检查服务
func NewExComponentService(nodeManager *nodesync.Manager) *ExComponentService {
	return &ExComponentService{
		nodeManager: nodeManager,
	}
}

var (
	// exComponentService 单例实例
	exComponentService *ExComponentService
)

// InitExComponentService 初始化扩展组件服务单例
func InitExComponentService(nodeManager *nodesync.Manager) {
	exComponentService = NewExComponentService(nodeManager)
}

// FetchExComponentCheckRst 获取扩展组件检查结果（包级函数，向后兼容）
func FetchExComponentCheckRst(cluster string) ExComponentCheck {
	if exComponentService == nil {
		return ExComponentCheck{}
	}
	return exComponentService.FetchExComponentCheckRst(cluster)
}

// FetchExComponentCheckRst 获取扩展组件检查结果
func (s *ExComponentService) FetchExComponentCheckRst(cluster string) ExComponentCheck {
	var rst ExComponentCheck
	rst.Asta, _ = s.AstaVerification(cluster)
	rst.Armada, _ = s.ArmadaDeployVerification(cluster)
	rst.CatAgent, _ = s.CatAgentVerification(cluster)
	rst.HIDS, _ = s.HIDSVerification(cluster)
	rst.Higress, _ = s.HigressVerification(cluster)
	rst.Ingress, _ = s.IngressVerification(cluster)
	rst.Istio, _ = s.IstioDeployVerification(cluster)
	rst.SolarAgent, _ = s.HaloAgentVerification(cluster)
	rst.JMXDS, _ = s.JmxDSVerification(cluster)
	return rst
}

// IngressVerification Ingress 控制器验证
func (s *ExComponentService) IngressVerification(cluster string) (CheckRst, errs.Error) {
	var rst CheckRst
	rst.ResourceType = Deployment

	deploys, err := s.nodeManager.ListDeployments(cluster, IngressControllerNS)
	if err != nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	var count int
	for _, deploy := range deploys {
		if strings.Contains(deploy.Name, "ingress-controller") {
			rst.Exist = true
			count++
		}
	}
	rst.ResourceCount = count
	return rst, nil
}

// HigressVerification Higress 验证
func (s *ExComponentService) HigressVerification(cluster string) (CheckRst, errs.Error) {
	var rst CheckRst
	rst.ResourceType = Deployment

	deploys, err := s.nodeManager.ListDeployments(cluster, HigressControllerNS)
	if err != nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	if len(deploys) > 0 {
		rst.Exist = true
		rst.ResourceCount = len(deploys)
	}
	return rst, nil
}

// HaloAgentVerification Halo Agent 验证
func (s *ExComponentService) HaloAgentVerification(cluster string) (CheckRst, errs.Error) {
	var rst CheckRst
	rst.ResourceType = DaemonSet

	daemonSets, err := s.nodeManager.ListDaemonSets(cluster, HaloNS)
	if err != nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	if len(daemonSets) > 0 {
		rst.Exist = true
		rst.ResourceCount = len(daemonSets)
	}
	return rst, nil
}

// CatAgentVerification CAT Agent 验证
func (s *ExComponentService) CatAgentVerification(cluster string) (CheckRst, errs.Error) {
	var rst CheckRst
	rst.ResourceType = DaemonSet

	daemonSets, err := s.nodeManager.ListDaemonSets(cluster, CatNs)
	if err != nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	if len(daemonSets) > 0 {
		rst.Exist = true
		rst.ResourceCount = len(daemonSets)
	}
	return rst, nil
}

// JmxDSVerification JMX DaemonSet 验证
func (s *ExComponentService) JmxDSVerification(cluster string) (CheckRst, errs.Error) {
	var rst CheckRst
	rst.ResourceType = DaemonSet

	daemonSets, err := s.nodeManager.ListDaemonSets(cluster, JmxNs)
	if err != nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	if len(daemonSets) > 0 {
		rst.Exist = true
		rst.ResourceCount = len(daemonSets)
	}
	return rst, nil
}

// HIDSVerification HIDS 验证
func (s *ExComponentService) HIDSVerification(cluster string) (CheckRst, errs.Error) {
	var rst CheckRst
	rst.ResourceType = DaemonSet

	daemonSets, err := s.nodeManager.ListDaemonSets(cluster, HiveNS)
	if err != nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	if len(daemonSets) > 0 {
		rst.Exist = true
		rst.ResourceCount = len(daemonSets)
	}
	return rst, nil
}

// AstaVerification Asta 验证
func (s *ExComponentService) AstaVerification(cluster string) (CheckRst, errs.Error) {
	var rst CheckRst
	rst.ResourceType = DaemonSet

	daemonSets, err := s.nodeManager.ListDaemonSets(cluster, AstaNS)
	if err != nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	if len(daemonSets) > 0 {
		rst.Exist = true
		rst.ResourceCount = len(daemonSets)
	}
	return rst, nil
}

// IstioDeployVerification Istio Deployment 验证
func (s *ExComponentService) IstioDeployVerification(cluster string) (CheckRst, errs.Error) {
	var rst CheckRst
	rst.ResourceType = Deployment

	deploys, err := s.nodeManager.ListDeployments(cluster, IstioNS)
	if err != nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	if len(deploys) > 0 {
		rst.Exist = true
		rst.ResourceCount = len(deploys)
	}
	return rst, nil
}

// ArmadaDeployVerification Armada Deployment 验证
func (s *ExComponentService) ArmadaDeployVerification(cluster string) (CheckRst, errs.Error) {
	var rst CheckRst
	rst.ResourceType = Deployment

	deploys, err := s.nodeManager.ListDeployments(cluster, ArmadaNS)
	if err != nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	var count int
	for _, deploy := range deploys {
		if strings.Contains(deploy.Name, "10690") {
			rst.Exist = true
			count++
		}
	}
	rst.ResourceCount = count
	return rst, nil
}

// 向后兼容的包级函数
func IngressVerification(cluster string) (rst CheckRst, err errs.Error) {
	if exComponentService == nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, "ExComponentService 未初始化")
	}
	return exComponentService.IngressVerification(cluster)
}

func HigressVerification(cluster string) (rst CheckRst, err errs.Error) {
	if exComponentService == nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, "ExComponentService 未初始化")
	}
	return exComponentService.HigressVerification(cluster)
}

func HaloAgentVerification(cluster string) (rst CheckRst, err errs.Error) {
	if exComponentService == nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, "ExComponentService 未初始化")
	}
	return exComponentService.HaloAgentVerification(cluster)
}

func CatAgentVerification(cluster string) (rst CheckRst, err errs.Error) {
	if exComponentService == nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, "ExComponentService 未初始化")
	}
	return exComponentService.CatAgentVerification(cluster)
}

func JmxDSVerification(cluster string) (rst CheckRst, err errs.Error) {
	if exComponentService == nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, "ExComponentService 未初始化")
	}
	return exComponentService.JmxDSVerification(cluster)
}

func HIDSVerification(cluster string) (rst CheckRst, err errs.Error) {
	if exComponentService == nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, "ExComponentService 未初始化")
	}
	return exComponentService.HIDSVerification(cluster)
}

func AstaVerification(cluster string) (rst CheckRst, err errs.Error) {
	if exComponentService == nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, "ExComponentService 未初始化")
	}
	return exComponentService.AstaVerification(cluster)
}

func IstioDeployVerification(cluster string) (rst CheckRst, err errs.Error) {
	if exComponentService == nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, "ExComponentService 未初始化")
	}
	return exComponentService.IstioDeployVerification(cluster)
}

func ArmadaDeployVerification(cluster string) (rst CheckRst, err errs.Error) {
	if exComponentService == nil {
		return rst, errs.NewErrorWithMsg(errs.InternalErr, "ExComponentService 未初始化")
	}
	return exComponentService.ArmadaDeployVerification(cluster)
}
