package b

import (
	"fmt"

	"deeptalk/apis/requests"
	"deeptalk/apis/responses"
	"deeptalk/internal/audit"
	"deeptalk/internal/provider"

	"github.com/gin-gonic/gin"
)

// ProviderController 处理厂商、模型与路由管理。
type ProviderController struct {
	providers *provider.Service // 厂商能力
	audits    *audit.Service    // 审计
}

// NewProviderController 创建厂商控制器。
func NewProviderController(providers *provider.Service, audits *audit.Service) *ProviderController {
	return &ProviderController{providers: providers, audits: audits}
}

// List 返回厂商列表、筛选计数与接入健康。
func (h *ProviderController) List(c *gin.Context) {
	ctx := c.Request.Context()
	var req requests.ProviderListRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "筛选参数不合法")
		return
	}
	items, counts, err := h.providers.List(ctx, req.Filter)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	health, err := h.providers.Health(ctx)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	out := make([]responses.ProviderResponse, 0, len(items))
	for i := range items {
		out = append(out, responses.NewProvider(&items[i], h.providers.Status(&items[i])))
	}
	responses.OK(c, gin.H{
		"items":  out,
		"counts": counts,
		"health": health,
	})
}

// Create 新建厂商。
func (h *ProviderController) Create(c *gin.Context) {
	var req requests.ProviderSaveRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请填写厂商名称与 Base URL")
		return
	}
	p, err := h.providers.Create(c.Request.Context(), req.ToInput())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "provider.create", "provider", p.ID, "新增厂商 "+p.Name)
	responses.OK(c, gin.H{"provider": responses.NewProvider(p, h.providers.Status(p))})
}

// Update 编辑厂商。
func (h *ProviderController) Update(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "厂商 ID 不合法")
		return
	}
	id := idReq.ID
	var req requests.ProviderSaveRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请填写厂商名称与 Base URL")
		return
	}
	p, err := h.providers.Update(c.Request.Context(), id, req.ToInput())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "provider.update", "provider", p.ID, "更新厂商配置 "+p.Name)
	responses.OK(c, gin.H{"provider": responses.NewProvider(p, h.providers.Status(p))})
}

// Toggle 启用 / 停用厂商。
func (h *ProviderController) Toggle(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "厂商 ID 不合法")
		return
	}
	id := idReq.ID
	var req requests.EnabledRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "启用状态不合法")
		return
	}
	p, err := h.providers.SetEnabled(c.Request.Context(), id, req.Enabled)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	action := "provider.disable"
	detail := "停用厂商 " + p.Name
	if req.Enabled {
		action = "provider.enable"
		detail = "启用厂商 " + p.Name
	}
	recordAudit(c, h.audits, action, "provider", p.ID, detail)
	responses.OK(c, gin.H{"provider": responses.NewProvider(p, h.providers.Status(p))})
}

// Delete 删除自定义厂商。
func (h *ProviderController) Delete(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "厂商 ID 不合法")
		return
	}
	id := idReq.ID
	p, _ := h.providers.ByID(c.Request.Context(), id)
	if err := h.providers.Delete(c.Request.Context(), id); err != nil {
		responses.FromError(c, err)
		return
	}
	name := fmt.Sprintf("#%d", id)
	if p != nil {
		name = p.Name
	}
	recordAudit(c, h.audits, "provider.delete", "provider", id, "删除厂商 "+name)
	responses.OK(c, nil)
}

// Test 测试已保存厂商的连通性。
func (h *ProviderController) Test(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "厂商 ID 不合法")
		return
	}
	id := idReq.ID
	result, err := h.providers.Test(c.Request.Context(), id)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	p, _ := h.providers.ByID(c.Request.Context(), id)
	name := fmt.Sprintf("#%d", id)
	if p != nil {
		name = p.Name
	}
	if result.OK {
		recordAudit(c, h.audits, "provider.test.ok", "provider", id, name+" 连接测试通过")
	} else {
		recordAudit(c, h.audits, "provider.test.fail", "provider", id, name+" 连接测试失败："+result.Message)
	}
	responses.OK(c, result)
}

// TestDraft 测试尚未保存的厂商配置。
func (h *ProviderController) TestDraft(c *gin.Context) {
	var req requests.ProviderTestRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "测试配置不合法")
		return
	}
	result, err := h.providers.TestDraft(c.Request.Context(), req.ID, req.ToInput())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, result)
}

// Models 返回模型路由表与当前默认 / 兜底路由。
func (h *ProviderController) Models(c *gin.Context) {
	items, err := h.providers.ListModels(c.Request.Context())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	out := make([]responses.AdminModelResponse, 0, len(items))
	for i := range items {
		out = append(out, responses.NewAdminModel(&items[i]))
	}
	responses.OK(c, gin.H{
		"items":   out,
		"routing": h.providers.Routing(c.Request.Context()),
	})
}

// CreateModel 新增模型。
func (h *ProviderController) CreateModel(c *gin.Context) {
	var req requests.ModelSaveRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请填写模型 ID")
		return
	}
	if req.ProviderID == 0 {
		responses.Fail(c, responses.CodeInvalidArgument, "请选择厂商")
		return
	}
	m, err := h.providers.CreateModel(c.Request.Context(), req.ProviderID, req.ToInput())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "model.create", "model", m.ID, "新增模型 "+m.ModelID)
	responses.OK(c, gin.H{"model": responses.NewAdminModel(m)})
}

// UpdateModel 编辑模型。
func (h *ProviderController) UpdateModel(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "模型 ID 不合法")
		return
	}
	id := idReq.ID
	var req requests.ModelSaveRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请填写模型 ID")
		return
	}
	m, err := h.providers.UpdateModel(c.Request.Context(), id, req.ToInput())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "model.update", "model", m.ID, "更新模型 "+m.ModelID)
	responses.OK(c, gin.H{"model": responses.NewAdminModel(m)})
}

// ToggleModel 启用 / 停用模型。
func (h *ProviderController) ToggleModel(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "模型 ID 不合法")
		return
	}
	id := idReq.ID
	var req requests.EnabledRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "启用状态不合法")
		return
	}
	m, err := h.providers.SetModelEnabled(c.Request.Context(), id, req.Enabled)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	action, detail := "model.disable", "停用模型 "+m.ModelID
	if req.Enabled {
		action, detail = "model.enable", "启用模型 "+m.ModelID
	}
	recordAudit(c, h.audits, action, "model", m.ID, detail)
	responses.OK(c, gin.H{"model": responses.NewAdminModel(m)})
}

// Routing 返回默认 / 兜底路由。
func (h *ProviderController) Routing(c *gin.Context) {
	responses.OK(c, h.providers.Routing(c.Request.Context()))
}

// SetRouting 更新默认 / 兜底路由。
func (h *ProviderController) SetRouting(c *gin.Context) {
	var req requests.RoutingRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请选择默认模型")
		return
	}
	if err := h.providers.SetRouting(c.Request.Context(), req.DefaultModelID, req.FallbackModelID); err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "routing.update", "setting", 0,
		fmt.Sprintf("默认模型 #%d，兜底模型 #%d", req.DefaultModelID, req.FallbackModelID))
	responses.OK(c, h.providers.Routing(c.Request.Context()))
}

// Health 返回接入健康数据。
func (h *ProviderController) Health(c *gin.Context) {
	health, err := h.providers.Health(c.Request.Context())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, health)
}

// DeleteModel 删除单个模型；预置模型不可删除。
func (h *ProviderController) DeleteModel(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "模型 ID 不合法")
		return
	}
	id := idReq.ID
	if err := h.providers.DeleteModel(c.Request.Context(), id); err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "model.delete", "model", id, "删除模型")
	responses.OK(c, nil)
}
