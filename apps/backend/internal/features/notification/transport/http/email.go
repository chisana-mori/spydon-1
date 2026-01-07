package http

import (
	"strconv"

	"robusta-web/backend/internal/features/notification/services"
	"robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// EmailHandler 邮件相关 HTTP Handler
type EmailHandler struct {
	templateService     *services.EmailTemplateService
	contactService      *services.EmailContactService
	notificationService *services.EmailNotificationService
}

// NewEmailHandler 创建 EmailHandler
func NewEmailHandler(
	templateService *services.EmailTemplateService,
	contactService *services.EmailContactService,
	notificationService *services.EmailNotificationService,
) *EmailHandler {
	return &EmailHandler{
		templateService:     templateService,
		contactService:      contactService,
		notificationService: notificationService,
	}
}

// RegisterRoutes 注册路由
func (h *EmailHandler) RegisterRoutes(rg *gin.RouterGroup) {
	email := rg.Group("/email")
	{
		// 模板路由
		templates := email.Group("/templates")
		{
			templates.GET("", h.ListTemplates)
			templates.GET("/:id", h.GetTemplate)
			templates.POST("", h.CreateTemplate)
			templates.PUT("/:id", h.UpdateTemplate)
			templates.DELETE("/:id", h.DeleteTemplate)
		}

		// 联系人路由
		contacts := email.Group("/contacts")
		{
			contacts.GET("", h.ListContacts)
			contacts.GET("/:id", h.GetContact)
			contacts.POST("", h.CreateContact)
			contacts.PUT("/:id", h.UpdateContact)
			contacts.DELETE("/:id", h.DeleteContact)
		}

		// 发送相关路由
		email.POST("/preview", h.PreviewEmail)
		email.POST("/send", h.SendEmail)
		email.GET("/affected-resources", h.GetAffectedResources)
	}
}

// ========================================
// Template Handlers
// ========================================

// ListTemplates 获取模板列表
// @Summary 获取邮件模板列表
// @Description 分页查询系统中已定义的邮件通知模板。返回结果包含模板名称、标题预览、关联的业务场景以及启用状态。该接口支撑了邮件通知页面的模板管理模块，允许运维人员快速查找并复用现有的通知文案。
// @Tags Notification,Email
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Success 200 {object} httpx.Response{data=[]services.EmailTemplate}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /email/templates [get]
func (h *EmailHandler) ListTemplates(c *gin.Context) {
	var query services.EmailTemplateListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.BadRequest(c, "", "参数错误: "+err.Error())
		return
	}

	templates, total, err := h.templateService.ListTemplates(query)
	if err != nil {
		httpx.InternalError(c, "", err.Error())
		return
	}

	httpx.SuccessPaginated(c, templates, &httpx.Pagination{Total: total, Page: query.Page, PageSize: query.Size})
}

// GetTemplate 获取模板详情
// @Summary 获取邮件模板详情
// @Description 根据指定的ID加载特定邮件模板的详细定义。返回内容包括完整的HTML正文、预留的模板变量占位符及其默认值描述。该接口用于前端渲染模板编辑界面，或在发送邮件前进行内容预览的数据回显。
// @Tags Notification,Email
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} httpx.Response{data=services.EmailTemplate}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /email/templates/{id} [get]
func (h *EmailHandler) GetTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的模板ID")
		return
	}

	template, err := h.templateService.GetTemplate(id)
	if err != nil {
		httpx.NotFound(c, "", err.Error())
		return
	}

	httpx.Success(c, template)
}

// CreateTemplate 创建模板
// @Summary 新增邮件通知模板
// @Description 在系统中持久化一个新的邮件通知模板。需提供模板的可辨识名称、邮件主题以及支持变量替换的HTML格式正文。创建成功的模板可被后续的自动化告警流程或手动通知任务所引用，提升通知排版的效率与一致性。
// @Tags Notification,Email
// @Accept json
// @Produce json
// @Param request body services.CreateEmailTemplateRequest true "模板创建参数"
// @Success 201 {object} httpx.Response{data=services.EmailTemplate}
// @Failure 400 {object} httpx.ErrorResponse
// @Router /email/templates [post]
func (h *EmailHandler) CreateTemplate(c *gin.Context) {
	var req services.CreateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "", "参数错误: "+err.Error())
		return
	}

	template, err := h.templateService.CreateTemplate(req)
	if err != nil {
		httpx.BadRequest(c, "", err.Error())
		return
	}

	httpx.Created(c, template)
}

// UpdateTemplate 更新模板
// @Summary 修改邮件模板配置
// @Description 编辑已有邮件模板的属性信息。管理员可以修改邮件主题、调整正文中的HTML布局或增删模板变量。更新后的模板将立即生效于下次发送请求，确保通知内容能够根据业务需求的变更进行快速灵活的调整。
// @Tags Notification,Email
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param request body services.UpdateEmailTemplateRequest true "模板更新参数"
// @Success 200 {object} httpx.Response{data=services.EmailTemplate}
// @Failure 400 {object} httpx.ErrorResponse
// @Router /email/templates/{id} [put]
func (h *EmailHandler) UpdateTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的模板ID")
		return
	}

	var req services.UpdateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "", "参数错误: "+err.Error())
		return
	}

	template, err := h.templateService.UpdateTemplate(id, req)
	if err != nil {
		httpx.BadRequest(c, "", err.Error())
		return
	}

	httpx.Success(c, template)
}

// DeleteTemplate 删除模板
// @Summary 物理删除邮件模板
// @Description 根据模板ID从数据库中彻底移除相关的配置记录。删除操作不可逆，且会导致正在使用该模板的自动化流程发生异常。请在执行删除前确保该模板已不再与任何活跃的热点告警或定期报告推送任务相关联。
// @Tags Notification,Email
// @Param id path int true "模板ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Router /email/templates/{id} [delete]
func (h *EmailHandler) DeleteTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的模板ID")
		return
	}

	if err := h.templateService.DeleteTemplate(id); err != nil {
		httpx.BadRequest(c, "", err.Error())
		return
	}

	httpx.Success(c, gin.H{"message": "删除成功"})
}

// ========================================
// Contact Handlers
// ========================================

// ListContacts 获取联系人列表
// @Summary 获取邮件联系人名录
// @Description 分页检索已保存的邮件通知接收人信息。返回数据包含联系人姓名、电子邮箱地址、所属团队以及订阅的告警级别。此接口支撑了发送邮件时的联想输入功能，方便运维人员快速选择对应的负责人员。
// @Tags Notification,Email
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Success 200 {object} httpx.Response{data=[]services.EmailContact}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /email/contacts [get]
func (h *EmailHandler) ListContacts(c *gin.Context) {
	var query services.EmailContactListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.BadRequest(c, "", "参数错误: "+err.Error())
		return
	}

	contacts, total, err := h.contactService.ListContacts(query)
	if err != nil {
		httpx.InternalError(c, "", err.Error())
		return
	}

	httpx.SuccessPaginated(c, contacts, &httpx.Pagination{Total: total, Page: query.Page, PageSize: query.Size})
}

// GetContact 获取联系人详情
// @Summary 获取联系人详细资料
// @Description 根据ID查询特定通知接收人的完整档案。涵盖其主送邮箱、抄送偏好以及在紧急故障发生时的主要联系方式。这些数据确保了通知分层分级体系的准确性，是实现精确通知推送的技术基础保障。
// @Tags Notification,Email
// @Produce json
// @Param id path int true "联系人ID"
// @Success 200 {object} httpx.Response{data=services.EmailContact}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /email/contacts/{id} [get]
func (h *EmailHandler) GetContact(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的联系人ID")
		return
	}

	contact, err := h.contactService.GetContact(id)
	if err != nil {
		httpx.NotFound(c, "", err.Error())
		return
	}

	httpx.Success(c, contact)
}

// CreateContact 创建联系人
// @Summary 新增邮件通知接收人
// @Description 在全局通讯录中添加一个新的通知对象。用户需要录入有效的邮件地址和姓名。新添加的联系人将立即出现在通知候选列表中，可用于接收来自系统的各类业务告警、变更公告或定期生成的运维分析报表。
// @Tags Notification,Email
// @Accept json
// @Produce json
// @Param request body services.CreateEmailContactRequest true "联系人基本信息"
// @Success 201 {object} httpx.Response{data=services.EmailContact}
// @Failure 400 {object} httpx.ErrorResponse
// @Router /email/contacts [post]
func (h *EmailHandler) CreateContact(c *gin.Context) {
	var req services.CreateEmailContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "", "参数错误: "+err.Error())
		return
	}

	contact, err := h.contactService.CreateContact(req)
	if err != nil {
		httpx.BadRequest(c, "", err.Error())
		return
	}

	httpx.Created(c, contact)
}

// UpdateContact 更新联系人
// @Summary 修改联系人信息
// @Description 更新已有联系人的邮箱地址、全名或所属组织架构详情。系统会同步校验新邮箱格式的合法性。该操作通常用于应对人员离职、职位调整或邮箱变更等组织变动场景，确保护航通知能够始终送达到正确的人。
// @Tags Notification,Email
// @Accept json
// @Produce json
// @Param id path int true "联系人ID"
// @Param request body services.UpdateEmailContactRequest true "联系人更新内容"
// @Success 200 {object} httpx.Response{data=services.EmailContact}
// @Failure 400 {object} httpx.ErrorResponse
// @Router /email/contacts/{id} [put]
func (h *EmailHandler) UpdateContact(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的联系人ID")
		return
	}

	var req services.UpdateEmailContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "", "参数错误: "+err.Error())
		return
	}

	contact, err := h.contactService.UpdateContact(id, req)
	if err != nil {
		httpx.BadRequest(c, "", err.Error())
		return
	}

	httpx.Success(c, contact)
}

// DeleteContact 删除联系人
// @Summary 物理删除联系人记录
// @Description 从系统中移除指定的邮件通知接收人。执行删除后，该联系人将不再出现在任何通知任务的候选项中。建议在删除前检查该联系人是否为某些核心业务的唯一通知端点，以避免关键告警发送失败。
// @Tags Notification,Email
// @Param id path int true "联系人ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Router /email/contacts/{id} [delete]
func (h *EmailHandler) DeleteContact(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的联系人ID")
		return
	}

	if err := h.contactService.DeleteContact(id); err != nil {
		httpx.BadRequest(c, "", err.Error())
		return
	}

	httpx.Success(c, gin.H{"message": "删除成功"})
}

// ========================================
// Email Sending Handlers
// ========================================

// PreviewEmail 预览邮件
// @Summary 实时预览邮件正文
// @Description 在实际发送前，根据选定的模板及填充的业务变量（通过K8s资源转换等方式获得），生成并返回最终的HTML渲染内容。此功能允许用户即时检查邮件排版和数据填充的正确性，有效避免发送错误的通知。
// @Tags Notification,Email
// @Accept json
// @Produce json
// @Param request body services.PreviewEmailRequest true "预览请求参数"
// @Success 200 {object} httpx.Response{data=services.PreviewEmailResponse}
// @Failure 400 {object} httpx.ErrorResponse
// @Router /email/preview [post]
func (h *EmailHandler) PreviewEmail(c *gin.Context) {
	var req services.PreviewEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "", "参数错误: "+err.Error())
		return
	}

	preview, err := h.notificationService.PreviewEmail(req)
	if err != nil {
		httpx.BadRequest(c, "", err.Error())
		return
	}

	httpx.Success(c, preview)
}

// SendEmail 发送邮件
// @Summary 触发邮件通知发送
// @Description 发起一次异步或同步的邮件发送请求。该接口集成了模板引擎渲染及SMTP底层传输协议，支持向多个接收人发送高度定制化的业务通知。发送结果将记录在系统的审计日志中，便于后续追踪通知的触达情况。
// @Tags Notification,Email
// @Accept json
// @Produce json
// @Param request body services.SendEmailRequest true "发送邮件详情"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Router /email/send [post]
func (h *EmailHandler) SendEmail(c *gin.Context) {
	var req services.SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "", "参数错误: "+err.Error())
		return
	}

	if err := h.notificationService.SendEmail(req); err != nil {
		httpx.BadRequest(c, "", err.Error())
		return
	}

	httpx.Success(c, gin.H{"message": "邮件发送成功"})
}

// GetAffectedResources 获取受影响资源
// @Summary 提取通知关联资源
// @Description 根据CI列表和资源类型，反向解析并查询受当前运维操作影响的所有K8s或基础设施资源列表。该接口为邮件发送提供动态的数据源，确保通知邮件能够自动携带当前变更所涉及的精确资源清单详情数据。
// @Tags Notification,Email
// @Produce json
// @Param ci_codes query []string true "设备CI列表"
// @Param resource_type query string true "资源类型 (如 Pod, Namespace)"
// @Success 200 {object} httpx.Response{data=[]object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /email/affected-resources [get]
func (h *EmailHandler) GetAffectedResources(c *gin.Context) {
	var req services.GetAffectedResourcesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.BadRequest(c, "", "参数错误: "+err.Error())
		return
	}

	resources, err := h.notificationService.GetAffectedResources(req)
	if err != nil {
		httpx.InternalError(c, "", err.Error())
		return
	}

	httpx.Success(c, resources)
}
