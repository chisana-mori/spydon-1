package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"net"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/pkg/nodesync"
	"robusta-web/backend/pkg/errs"
	"robusta-web/backend/pkg/logger"
	"robusta-web/backend/pkg/mailer"
	"robusta-web/backend/pkg/narwhal"
	"robusta-web/backend/pkg/orchid"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/tidwall/gjson"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
)

const (
	NodeValues            = "$nodes"
	PodValues             = "$pods"
	HistoryPodValues      = "$historyPods"
	ComponentValues       = "$components"
	DeployValues          = "$deploys"
	CIDRValues            = "$nodeCidrs"
	OfflineDeployValues   = "$offlineDeploys"
	OnlineDeployValues    = "$onlineDeploys"
	MigrationDeployValues = "$migrationDeploys"

	Pod    = "pod"
	Deploy = "deploy"
)

type NoticeEmailFe struct {
	db          *gorm.DB
	mailer      *mailer.Mailer
	nodeSyncMgr *nodesync.Manager
}

func NewNoticeEmailFe(db *db.Database, mailer *mailer.Mailer, nodeSyncMgr *nodesync.Manager) *NoticeEmailFe {
	return &NoticeEmailFe{
		db:          db.DB,
		mailer:      mailer,
		nodeSyncMgr: nodeSyncMgr,
	}
}

func (fm *NoticeEmailFe) PostEmail(req SendEmailReq) errs.Error {
	attachments := buildAttachments(req.AttachFiles)
	if len(attachments) > 0 {
		if err := fm.mailer.SendWithAttachment(req.Addresses, req.Subject, req.Content, attachments); err != nil {
			return errs.NewErrorWithMsg(errs.InternalErr, fmt.Sprintf("发送带附件邮件失败: %v", err))
		}
		return nil
	}

	if err := fm.mailer.SendHTML(req.Addresses, req.Subject, req.Content); err != nil {
		return errs.NewErrorWithMsg(errs.InternalErr, fmt.Sprintf("发送邮件失败: %v", err))
	}
	return nil
}

func (fm *NoticeEmailFe) BuildEmail(p MailGenReq) (rst SendEmailReq, err errs.Error) {
	clusterObj, err := fetchCluster(fm.db, p.Additional)
	if err != nil {
		return SendEmailReq{}, err
	}

	templateObj, err := fetchTemplate(fm.db, uint64(p.TemplateId))
	if err != nil {
		return SendEmailReq{}, err
	}

	addressObj, err := fetchAddress(fm.db, uint64(p.AddressId))
	if err != nil {
		return SendEmailReq{}, err
	}

	nodes := extractNodes(p.Additional)
	mailProperty := MailContentProperty{
		Cluster:    clusterObj,
		Additional: p.Additional,
	}

	files, err := processTableParams(fm, p, templateObj, clusterObj, nodes, &mailProperty)
	if err != nil {
		return SendEmailReq{}, err
	}

	addresses, err := buildEmailAddresses(p, addressObj)
	if err != nil {
		return SendEmailReq{}, err
	}

	rst.Addresses = addresses
	rst.Subject, rst.Content = GenEmail(templateObj.Body, templateObj.Title, mailProperty)
	rst.AttachFiles = files
	rst.TemplateId = p.TemplateId

	return rst, nil
}

func getRequiredValue(key string, store map[string]interface{}) (interface{}, errs.Error) {
	if val, ok := store[key]; ok {
		return val, nil
	}
	return nil, errs.NewErrorWithMsg(errs.InternalErr, fmt.Sprintf("必填参数 %s 未提交，请重新输入！", key))
}

func getRequiredString(key string, store map[string]interface{}) (string, errs.Error) {
	val, err := getRequiredValue(key, store)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", errs.NewErrorWithMsg(errs.InternalErr, fmt.Sprintf("参数 %s 格式错误", key))
}

func fetchCluster(db *gorm.DB, additional map[string]interface{}) (models.Cluster, errs.Error) {
	var clusterObj models.Cluster
	clusterIdVal, err := getRequiredValue("clusterId", additional)
	if err != nil {
		return clusterObj, err
	}

	clusterId, ok := clusterIdVal.(float64)
	if !ok {
		if id, ok := clusterIdVal.(int); ok {
			clusterId = float64(id)
		} else {
			return clusterObj, errs.NewErrorWithMsg(errs.InternalErr, "clusterId 格式错误")
		}
	}

	clusterObj.ID = uint(clusterId)
	if dbErr := db.First(&clusterObj).Error; dbErr != nil {
		return clusterObj, errs.NewErrorWithMsg(errs.InternalErr, fmt.Sprintf("查询集群失败: %v", dbErr))
	}
	return clusterObj, nil
}

func fetchTemplate(db *gorm.DB, templateId uint64) (models.EmailTemplate, errs.Error) {
	var templateObj models.EmailTemplate
	templateObj.ID = templateId
	if err := db.First(&templateObj).Error; err != nil {
		return templateObj, errs.NewErrorWithMsg(errs.InternalErr, fmt.Sprintf("查询邮件模板失败: %v", err))
	}
	return templateObj, nil
}

func fetchAddress(db *gorm.DB, addressId uint64) (models.EmailContact, errs.Error) {
	var addressObj models.EmailContact
	addressObj.ID = addressId
	if err := db.First(&addressObj).Error; err != nil {
		return addressObj, errs.NewErrorWithMsg(errs.InternalErr, fmt.Sprintf("查询邮件联系人失败: %v", err))
	}
	return addressObj, nil
}

func extractNodes(additional map[string]interface{}) []string {
	var nodes []string
	if nodesData, ok := additional["nodes"]; ok {
		if list, ok := nodesData.([]interface{}); ok {
			for _, ele := range list {
				if str, ok := ele.(string); ok {
					nodes = append(nodes, str)
				}
			}
		}
	}
	return nodes
}

func buildAttachments(files []AttachFile) []mailer.Attachment {
	attachments := make([]mailer.Attachment, 0, len(files))
	for _, file := range files {
		fileByte, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			logger.L().Error(fmt.Sprintf("解码附件失败: %v", err))
			continue
		}
		attachments = append(attachments, mailer.NewExcelAttachment(file.Name+".xlsx", fileByte))
	}
	return attachments
}

func processTableParams(fm *NoticeEmailFe, p MailGenReq, templateObj models.EmailTemplate, clusterObj models.Cluster, nodes []string, mailProperty *MailContentProperty) ([]AttachFile, errs.Error) {
	var files []AttachFile
	for _, param := range gjson.Get(templateObj.Params, "tables").Array() {
		file, err := handleTableParam(fm, param.String(), p, clusterObj, nodes, mailProperty)
		if err != nil {
			return nil, err
		}
		if file != nil {
			files = append(files, *file)
		}
	}
	return files, nil
}

func handleTableParam(fm *NoticeEmailFe, param string, p MailGenReq, clusterObj models.Cluster, nodes []string, mailProperty *MailContentProperty) (*AttachFile, errs.Error) {
	switch param {
	case PodValues:
		return handlePodValues(clusterObj.Name, nodes, mailProperty)
	case DeployValues:
		return handleDeployValues(clusterObj.Name, mailProperty)
	case HistoryPodValues:
		return handleHistoryPodValues(p, clusterObj, nodes, mailProperty)
	case OfflineDeployValues:
		return handleOfflineDeployValues(p, clusterObj, mailProperty)
	case OnlineDeployValues:
		return handleOnlineDeployValues(p, clusterObj, mailProperty)
	case CIDRValues:
		return handleCIDRValues(fm, clusterObj.Name, mailProperty)
	case ComponentValues:
		mailProperty.ComponentCheck = FetchExComponentCheckRst(clusterObj.Name)
		return nil, nil
	case NodeValues:
		mailProperty.Nodes = NodeFeatures(nodes, clusterObj.Name)
		return nil, nil
	default:
		return nil, nil
	}
}

// App handling helpers

func createAppAttachment(name string, apps []App, mType string, mailProperty *MailContentProperty) *AttachFile {
	boardApps, appIdMap := buildAppBoards(apps)
	mailProperty.Apps = boardApps
	mailProperty.AppIdCount = len(appIdMap)
	return &AttachFile{
		Name:    name,
		Content: base64.StdEncoding.EncodeToString(appExcel(boardApps, mType)),
	}
}

func handlePodValues(clusterName string, nodes []string, mailProperty *MailContentProperty) (*AttachFile, errs.Error) {
	podApps, err := InstantPods(clusterName, nodes)
	if err != nil {
		return nil, err
	}
	return createAppAttachment(fmt.Sprintf("%s列表", Pod), podApps, Pod, mailProperty), nil
}

func handleDeployValues(clusterName string, mailProperty *MailContentProperty) (*AttachFile, errs.Error) {
	deployApps := InstantDeploys(clusterName)
	return createAppAttachment(fmt.Sprintf("%s列表", Deploy), deployApps, Deploy, mailProperty), nil
}

func handleHistoryPodValues(p MailGenReq, clusterObj models.Cluster, nodes []string, mailProperty *MailContentProperty) (*AttachFile, errs.Error) {
	begin, err := getRequiredString("begin", p.Additional)
	if err != nil {
		return nil, err
	}

	podApps, err := HistoryPods(clusterObj.ID, begin, nodes)
	if err != nil {
		return nil, err
	}
	return createAppAttachment(fmt.Sprintf("%s列表", Pod), podApps, Pod, mailProperty), nil
}

func handleOfflineDeployValues(p MailGenReq, clusterObj models.Cluster, mailProperty *MailContentProperty) (*AttachFile, errs.Error) {
	historyApps, instantApps, err := fetchHistoryAndInstantDeploys(p, clusterObj)
	if err != nil {
		return nil, err
	}

	apps, _ := buildAppBoards(instantApps)
	mailProperty.Apps = apps
	mailProperty.DiffAppsBoard = DiffAppsBoard{
		Apps:               apps,
		HistoryDeployCount: len(historyApps),
		InstantDeployCount: len(instantApps),
	}

	return &AttachFile{
		Name:    fmt.Sprintf("待下线%s列表", Deploy),
		Content: base64.StdEncoding.EncodeToString(appExcel(apps, Deploy)),
	}, nil
}

func handleOnlineDeployValues(p MailGenReq, clusterObj models.Cluster, mailProperty *MailContentProperty) (*AttachFile, errs.Error) {
	historyApps, instantApps, err := fetchHistoryAndInstantDeploys(p, clusterObj)
	if err != nil {
		return nil, err
	}

	onlineApps, intersectionCount := diffApps(historyApps, instantApps)
	apps, _ := buildAppBoards(onlineApps)
	mailProperty.Apps = apps
	mailProperty.DiffAppsBoard = DiffAppsBoard{
		Apps:               apps,
		HistoryDeployCount: len(historyApps),
		InstantDeployCount: intersectionCount,
	}
	return &AttachFile{
		Name:    fmt.Sprintf("待上线%s列表", Deploy),
		Content: base64.StdEncoding.EncodeToString(appExcel(apps, Deploy)),
	}, nil
}

func fetchHistoryAndInstantDeploys(p MailGenReq, clusterObj models.Cluster) (historyApps []App, instantApps []App, err errs.Error) {
	begin, err := getRequiredString("upgradeDate", p.Additional)
	if err != nil {
		return
	}
	if historyApps, err = HistoryDeploys(clusterObj.ID, begin); err != nil {
		return
	}
	instantApps = InstantDeploys(clusterObj.Name)
	return
}

func diffApps(excludeSet []App, sourceSet []App) ([]App, int) {
	excludeMap := make(map[string]bool)
	for _, app := range excludeSet {
		excludeMap[app.Name] = true
	}

	var result []App
	var intersection int
	for _, app := range sourceSet {
		if _, exists := excludeMap[app.Name]; !exists {
			result = append(result, app)
		} else {
			intersection++
		}
	}
	return result, intersection
}

func handleCIDRValues(fm *NoticeEmailFe, clusterName string, mailProperty *MailContentProperty) (*AttachFile, errs.Error) {
	nodes, err := fm.nodeSyncMgr.ListNodes(clusterName)
	if err != nil {
		return nil, errs.NewErrorWithMsg(errs.InternalErr, fmt.Sprintf("获取节点失败: %v", err))
	}
	mailProperty.NodeCidr = extractCIDRsFromNodes(nodes)
	return nil, nil
}

func extractCIDRsFromNodes(nodes []corev1.Node) []string {
	cidrSet := make(map[string]bool)
	for _, node := range nodes {
		for _, addr := range node.Status.Addresses {
			if addr.Type != corev1.NodeInternalIP {
				continue
			}
			ip := net.ParseIP(addr.Address)
			if ip == nil {
				continue
			}
			ipv4 := ip.To4()
			if ipv4 == nil {
				continue
			}

			// Mask 24: /24, specifically modifying the 4th octet to 0
			ipv4[3] = 0
			ipNet := &net.IPNet{
				IP:   ipv4,
				Mask: net.CIDRMask(24, 32),
			}
			cidrSet[ipNet.String()] = true
		}
	}

	cidrs := make([]string, 0, len(cidrSet))
	for cidr := range cidrSet {
		cidrs = append(cidrs, cidr)
	}
	return cidrs
}

func buildEmailAddresses(p MailGenReq, addressObj models.EmailContact) ([]string, errs.Error) {
	addresses := strings.Split(addressObj.Address, ",")

	officerEmails, err := fetchOfficerEmails()
	if err != nil {
		return nil, err
	}
	addresses = append(addresses, officerEmails...)
	return addresses, nil
}

func fetchOfficerEmails() ([]string, errs.Error) {
	officers, err := narwhal.TodayOfficer()
	if err != nil {
		return nil, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	emails := make([]string, 0, len(officers))
	for _, user := range officers {
		if user == "" {
			continue
		}
		resp, err := orchid.QueryUser(user)
		if err != nil {
			return nil, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
		}
		if len(resp.Data.User) > 0 {
			emails = append(emails, resp.Data.User[0].Email)
		}
	}
	return emails, nil
}

func GenEmail(bodyTem, titleTem string, data MailContentProperty) (string, string) {
	render := func(name, tplStr string) string {
		t := template.New(name).Funcs(sprig.HtmlFuncMap())
		t, err := t.Parse(tplStr)
		if err != nil {
			logger.L().Error(fmt.Sprintf("Parse template %s error: %v", name, err))
			return ""
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			logger.L().Error(fmt.Sprintf("Execute template %s error: %v", name, err))
			return ""
		}
		return buf.String()
	}

	return render("title", titleTem), render("content", bodyTem)
}

// Excel generation

var excelColumns = []struct {
	letter string
	header string
	width  float64
}{
	{"A", "appid", 20},
	{"B", "重要等级", 20},
	{"C", "namespace", 40},
	{"D", "子系统名", 40},
	{"E", "", 70},
	{"F", "运营虚拟组", 40},
	{"G", "应用负责人", 40},
	{"H", "应用运维", 40},
}

func appExcel(boards AppSlice, resourceType string) []byte {
	f := excelize.NewFile()
	style, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Indent: 1, Vertical: "center",
		},
	})

	sheet := "应用列表"
	if _, err := f.NewSheet(sheet); err != nil {
		logger.L().Error(fmt.Sprintf("创建工作表失败: %v", err))
	}

	// Set Headers
	for i, col := range excelColumns {
		header := col.header
		// Set dynamic header for column E (index 4)
		if i == 4 && resourceType != "" {
			header = resourceType
		}

		if header != "" {
			cellRef := fmt.Sprintf("%s1", col.letter)
			if err := f.SetCellStr(sheet, cellRef, header); err != nil {
				logger.L().Error(fmt.Sprintf("设置表头失败: %v", err))
			}
		}

		// Set width
		if err := f.SetColWidth(sheet, col.letter, col.letter, col.width); err != nil {
			logger.L().Error(fmt.Sprintf("设置列宽失败: %v", err))
		}
	}

	// Set Data
	for index, app := range boards {
		row := index + 2
		vals := map[string]string{
			"A": app.AppId,
			"B": app.AppLevel,
			"C": app.NamespaceName,
			"D": app.NamespaceCNName,
			"E": app.DeploymentName,
			"F": app.Team,
			"G": app.Develops,
			"H": app.Operations,
		}
		for col, val := range vals {
			cellRef := fmt.Sprintf("%s%d", col, row)
			if err := f.SetCellStr(sheet, cellRef, val); err != nil {
				logger.L().Error(fmt.Sprintf("设置单元格值失败: %v", err))
			}
			if err := f.SetCellStyle(sheet, cellRef, cellRef, style); err != nil {
				logger.L().Error(fmt.Sprintf("设置单元格样式失败: %v", err))
			}
		}
	}

	buffer, _ := f.WriteToBuffer()
	return buffer.Bytes()
}
