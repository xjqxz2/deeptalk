package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"deeptalk/internal/audit"
	"deeptalk/internal/setting"

	"gorm.io/gorm"
)

// AuditRecorder 是本模块消费的审计能力。
type AuditRecorder interface {
	Record(ctx context.Context, in audit.Record) error
}

// Service 提供厂商、模型与路由的业务能力。
type Service struct {
	db       *gorm.DB         // 数据库句柄
	secret   string           // AES-GCM 密钥源
	devMode  bool             // 开发模式：允许 http 与本地地址
	settings *setting.Service // 平台设置（默认/兜底路由）
	audit    AuditRecorder    // 审计记录器
}

// NewService 创建厂商服务。
func NewService(db *gorm.DB, secret string, devMode bool, settings *setting.Service, recorder AuditRecorder) *Service {
	return &Service{db: db, secret: secret, devMode: devMode, settings: settings, audit: recorder}
}

// List 返回厂商列表与筛选计数。
func (s *Service) List(ctx context.Context, filter string) ([]Provider, Counts, error) {
	var builtin *bool
	switch filter {
	case "builtin":
		v := true
		builtin = &v
	case "custom":
		v := false
		builtin = &v
	}
	items, err := dalListProviders(ctx, s.db, builtin)
	if err != nil {
		return nil, Counts{}, err
	}
	counts, err := dalCounts(ctx, s.db)
	if err != nil {
		return nil, Counts{}, err
	}
	return items, counts, nil
}

// ByID 按主键查询厂商。
func (s *Service) ByID(ctx context.Context, id uint) (*Provider, error) {
	return dalFindProvider(ctx, s.db, id)
}

// Status 按启用与验证状态推导卡片状态胶囊。
func (s *Service) Status(p *Provider) string {
	// 状态优先表达“配置是否可用”，其次才是开关：
	// 未验证但有错误 → 连接失败；未验证且无错误 → 待验证；验证通过但关闭 → 已停用
	switch {
	case p.LastError != "":
		return StatusFailed
	case !p.Verified:
		return StatusPending
	case p.Enabled:
		return StatusActive
	default:
		return StatusDisabled
	}
}

// Create 新建厂商与模型。
// 事务内完成厂商、模型、默认厂商切换，避免出现“有厂商无模型”的中间态。
func (s *Service) Create(ctx context.Context, in SaveInput) (*Provider, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	if err := s.ValidateBaseURL(baseURL); err != nil {
		return nil, err
	}
	kind := normalizeKind(in.Kind)
	apiStyle := NormalizeAPIStyle(in.APIStyle)
	timeout := in.TimeoutSec
	if timeout <= 0 {
		timeout = 60
	}
	concurrency := in.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	enc, err := EncryptAPIKey(s.secret, in.APIKey)
	if err != nil {
		return nil, err
	}
	headers, err := marshalHeaders(in.Headers)
	if err != nil {
		return nil, err
	}
	code := s.uniqueCode(ctx, name)
	p := &Provider{
		Name:           name,
		Code:           code,
		Kind:           kind,
		APIStyle:       apiStyle,
		BaseURL:        baseURL,
		APIKeyEnc:      enc,
		APIKeyMasked:   MaskAPIKey(in.APIKey),
		HeadersJSON:    headers,
		TimeoutSec:     clampInt(timeout, 5, 300),
		MaxConcurrency: clampInt(concurrency, 1, 10000),
		Enabled:        in.Enabled,
		Verified:       false,
		IsBuiltin:      false,
		IsDefault:      in.IsDefault,
		SortOrder:      100,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if p.IsDefault {
			if err := tx.Model(&Provider{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		if in.Models != nil {
			if err := s.replaceModels(ctx, tx, p.ID, *in.Models); err != nil {
				return err
			}
		}
		// 厂商并发变化时同步到该厂商的所有模型，保证卡片与模型表一致
		if in.MaxConcurrency > 0 {
			if err := tx.Model(&Model{}).Where("provider_id = ?", p.ID).
				Update("max_concurrency", p.MaxConcurrency).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 重新查询带模型的完整对象返回给前端
	return dalFindProvider(ctx, s.db, p.ID)
}

// Update 编辑厂商与模型；APIKey 为空表示保留原密钥。
func (s *Service) Update(ctx context.Context, id uint, in SaveInput) (*Provider, error) {
	p, err := dalFindProvider(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	if err := s.ValidateBaseURL(baseURL); err != nil {
		return nil, err
	}
	p.Name = name
	p.Kind = normalizeKind(in.Kind)
	p.APIStyle = NormalizeAPIStyle(in.APIStyle)
	p.BaseURL = baseURL
	if in.APIKey != "" {
		enc, err := EncryptAPIKey(s.secret, in.APIKey)
		if err != nil {
			return nil, err
		}
		p.APIKeyEnc = enc
		p.APIKeyMasked = MaskAPIKey(in.APIKey)
		// 更换密钥后必须重新验证
		p.Verified = false
		p.LastError = ""
	}
	if in.TimeoutSec > 0 {
		p.TimeoutSec = clampInt(in.TimeoutSec, 5, 300)
	}
	if in.MaxConcurrency > 0 {
		p.MaxConcurrency = clampInt(in.MaxConcurrency, 1, 10000)
	}
	p.Enabled = in.Enabled
	p.IsDefault = in.IsDefault
	if in.Headers != nil {
		headers, headerErr := marshalHeaders(in.Headers)
		if headerErr != nil {
			return nil, headerErr
		}
		p.HeadersJSON = headers
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if p.IsDefault {
			if err := tx.Model(&Provider{}).Where("id <> ? AND is_default = ?", p.ID, true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Save(p).Error; err != nil {
			return err
		}
		if in.Models != nil {
			if err := s.replaceModels(ctx, tx, p.ID, *in.Models); err != nil {
				return err
			}
		}
		// 厂商并发变化时同步到该厂商的所有模型，保证卡片与模型表一致
		if in.MaxConcurrency > 0 {
			if err := tx.Model(&Model{}).Where("provider_id = ?", p.ID).
				Update("max_concurrency", p.MaxConcurrency).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 删除模型后可能让默认/兜底路由悬空，需要修复
	if err := s.RepairRouting(ctx); err != nil {
		return nil, err
	}
	return dalFindProvider(ctx, s.db, p.ID)
}

// SetEnabled 启用 / 停用厂商；停用后其模型不再下发给 C 端。
func (s *Service) SetEnabled(ctx context.Context, id uint, enabled bool) (*Provider, error) {
	p, err := dalFindProvider(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	p.Enabled = enabled
	if err := s.db.WithContext(ctx).Model(&Provider{}).Where("id = ?", id).Update("enabled", enabled).Error; err != nil {
		return nil, err
	}
	if !enabled {
		if err := s.RepairRouting(ctx); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// Delete 删除自定义厂商及其模型；预置厂商禁止删除。
func (s *Service) Delete(ctx context.Context, id uint) error {
	p, err := dalFindProvider(ctx, s.db, id)
	if err != nil {
		return err
	}
	if p.IsBuiltin {
		return ErrBuiltinDelete
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("provider_id = ?", id).Delete(&Model{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Provider{}, id).Error
	})
	if err != nil {
		return err
	}
	return s.RepairRouting(ctx)
}

// ListModels 返回全部模型（含厂商名）并补齐默认 / 兜底标记。
func (s *Service) ListModels(ctx context.Context) ([]Model, error) {
	var items []Model
	err := s.db.WithContext(ctx).
		Table("models").
		Select("models.*, providers.name AS provider_name, providers.kind AS provider_kind").
		Joins("JOIN providers ON providers.id = models.provider_id").
		Order("models.sort_order ASC, models.id ASC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	routing := s.Routing(ctx)
	for i := range items {
		items[i].IsDefault = items[i].ID == routing.DefaultModelID
		items[i].IsFallback = items[i].ID == routing.FallbackModelID
	}
	return items, nil
}

// EnabledModels 返回 C 端可选模型（模型与厂商都必须启用）。
func (s *Service) EnabledModels(ctx context.Context) ([]Model, error) {
	var items []Model
	err := s.db.WithContext(ctx).
		Table("models").
		Select("models.*, providers.name AS provider_name, providers.kind AS provider_kind").
		Joins("JOIN providers ON providers.id = models.provider_id").
		Where("models.enabled = ? AND providers.enabled = ?", true, true).
		Order("models.is_default DESC, models.sort_order ASC, models.id ASC").
		Scan(&items).Error
	return items, err
}

// ResolveModel 解析模型与所属厂商，并校验两者都处于启用状态。
func (s *Service) ResolveModel(ctx context.Context, modelID uint) (*Model, *Provider, error) {
	var m Model
	if err := s.db.WithContext(ctx).First(&m, modelID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrModelNotFound
		}
		return nil, nil, err
	}
	p, err := dalFindProvider(ctx, s.db, m.ProviderID)
	if err != nil {
		return nil, nil, err
	}
	if !m.Enabled || !p.Enabled {
		return nil, nil, fmt.Errorf("模型已停用")
	}
	return &m, p, nil
}

// DefaultModel 返回平台默认模型；设置失效时回退到第一个启用模型。
func (s *Service) DefaultModel(ctx context.Context) (*Model, *Provider, error) {
	id := uint(s.settings.Int(ctx, setting.KeyDefaultModelID, 0))
	if id > 0 {
		if m, p, err := s.ResolveModel(ctx, id); err == nil {
			return m, p, nil
		}
	}
	items, err := s.EnabledModels(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(items) == 0 {
		return nil, nil, errors.New("当前没有可用模型，请联系管理员")
	}
	return s.ResolveModel(ctx, items[0].ID)
}

// Routing 读取默认与兜底模型路由。
func (s *Service) Routing(ctx context.Context) Routing {
	return Routing{
		DefaultModelID:  uint(s.settings.Int(ctx, setting.KeyDefaultModelID, 0)),
		FallbackModelID: uint(s.settings.Int(ctx, setting.KeyFallbackModelID, 0)),
	}
}

// SetRouting 设置默认与兜底模型；停用模型不可被选中。
func (s *Service) SetRouting(ctx context.Context, defaultID, fallbackID uint) error {
	if defaultID == 0 {
		return errors.New("default model is required")
	}
	if fallbackID > 0 && fallbackID == defaultID {
		return errors.New("兜底模型不能与默认模型相同")
	}
	if _, _, err := s.ResolveModel(ctx, defaultID); err != nil {
		return errors.New("默认模型不可用")
	}
	if fallbackID > 0 {
		if _, _, err := s.ResolveModel(ctx, fallbackID); err != nil {
			return errors.New("兜底模型不可用")
		}
	}
	if err := s.settings.SetMany(ctx, map[string]string{
		setting.KeyDefaultModelID:  strconv.FormatUint(uint64(defaultID), 10),
		setting.KeyFallbackModelID: strconv.FormatUint(uint64(fallbackID), 10),
	}); err != nil {
		return err
	}
	return s.syncRoutingFlags(ctx, defaultID, fallbackID)
}

// CreateModel 在指定厂商下新增模型。
func (s *Service) CreateModel(ctx context.Context, providerID uint, in ModelInput) (*Model, error) {
	if _, err := dalFindProvider(ctx, s.db, providerID); err != nil {
		return nil, err
	}
	m, err := buildModel(providerID, in, 100)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

// UpdateModel 编辑模型信息。
func (s *Service) UpdateModel(ctx context.Context, id uint, in ModelInput) (*Model, error) {
	var m Model
	if err := s.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, ErrModelNotFound
	}
	if strings.TrimSpace(in.ModelID) == "" {
		return nil, errors.New("model_id is required")
	}
	m.ModelID = strings.TrimSpace(in.ModelID)
	m.DisplayName = displayName(in)
	m.Description = strings.TrimSpace(in.Description)
	if in.ContextWindow > 0 {
		m.ContextWindow = in.ContextWindow
	}
	if in.MaxConcurrency > 0 {
		m.MaxConcurrency = clampInt(in.MaxConcurrency, 1, 10000)
	}
	m.Enabled = in.Enabled
	if err := s.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	if err := s.RepairRouting(ctx); err != nil {
		return nil, err
	}
	return &m, nil
}

// SetModelEnabled 启用 / 停用模型。
func (s *Service) SetModelEnabled(ctx context.Context, id uint, enabled bool) (*Model, error) {
	var m Model
	if err := s.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, ErrModelNotFound
	}
	if err := s.db.WithContext(ctx).Model(&Model{}).Where("id = ?", id).Update("enabled", enabled).Error; err != nil {
		return nil, err
	}
	m.Enabled = enabled
	if !enabled {
		if err := s.RepairRouting(ctx); err != nil {
			return nil, err
		}
	}
	return &m, nil
}

// RepairRouting 修复悬空的默认 / 兜底路由：优先保留原值，不可用时选第一个启用模型。
func (s *Service) RepairRouting(ctx context.Context) error {
	routing := s.Routing(ctx)
	defaultID := routing.DefaultModelID
	if defaultID == 0 {
		if items, err := s.EnabledModels(ctx); err == nil && len(items) > 0 {
			defaultID = items[0].ID
		}
	} else if _, _, err := s.ResolveModel(ctx, defaultID); err != nil {
		defaultID = 0
		if items, err := s.EnabledModels(ctx); err == nil && len(items) > 0 {
			defaultID = items[0].ID
		}
	}
	fallbackID := routing.FallbackModelID
	if fallbackID > 0 {
		if _, _, err := s.ResolveModel(ctx, fallbackID); err != nil || fallbackID == defaultID {
			fallbackID = 0
		}
	}
	if defaultID == 0 {
		return s.settings.SetMany(ctx, map[string]string{
			setting.KeyDefaultModelID:  "0",
			setting.KeyFallbackModelID: "0",
		})
	}
	return s.SetRouting(ctx, defaultID, fallbackID)
}

// EnabledCounts 返回已启用厂商与已启用模型数量，用于后台概览。
func (s *Service) EnabledCounts(ctx context.Context) (int64, int64, error) {
	var providers, models int64
	if err := s.db.WithContext(ctx).Model(&Provider{}).Where("enabled = ?", true).Count(&providers).Error; err != nil {
		return 0, 0, err
	}
	if err := s.db.WithContext(ctx).Model(&Model{}).Where("enabled = ?", true).Count(&models).Error; err != nil {
		return 0, 0, err
	}
	return providers, models, nil
}

// Health 汇总接入健康数据。首字节 P95 用已启用厂商的最近延迟最大值近似。
func (s *Service) Health(ctx context.Context) (Health, error) {
	health := Health{UpdatedAt: time.Now()}
	var providers []Provider
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Order("sort_order ASC").Find(&providers).Error; err != nil {
		return health, err
	}
	health.UpstreamsTotal = len(providers)
	items := make([]HealthItem, 0, len(providers))
	maxLatency := 0
	for _, p := range providers {
		ok := p.Verified
		if ok {
			health.UpstreamsAvailable++
		}
		if p.LastLatencyMS > maxLatency {
			maxLatency = p.LastLatencyMS
		}
		items = append(items, HealthItem{ProviderID: p.ID, Name: p.Name, OK: ok, LatencyMS: p.LastLatencyMS})
	}
	health.Items = items
	health.FirstByteP95MS = maxLatency
	// 近 1 小时的失败测试次数由审计日志统计，保证与“最近变更”一致
	since := time.Now().Add(-time.Hour)
	var count int64
	_ = s.db.WithContext(ctx).Model(&audit.Log{}).
		Where("action = ? AND created_at >= ?", "provider.test.fail", since).
		Count(&count).Error
	health.Errors1H = count
	return health, nil
}

// Test 测试已保存厂商的连通性，并回写验证状态。
func (s *Service) Test(ctx context.Context, id uint) (TestResult, error) {
	p, err := dalFindProvider(ctx, s.db, id)
	if err != nil {
		return TestResult{}, err
	}
	key, err := DecryptAPIKey(s.secret, p.APIKeyEnc)
	if err != nil {
		return TestResult{}, errors.New("密钥解密失败，请重新填写 API Key")
	}
	result := testEndpoint(ctx, p.BaseURL, key, ProviderHeaders(p))
	now := time.Now()
	p.LastTestedAt = &now
	p.LastLatencyMS = result.LatencyMS
	if result.OK {
		p.Verified = true
		p.LastError = ""
	} else {
		p.Verified = false
		p.LastError = truncate(result.Message, 250)
	}
	_ = s.db.WithContext(ctx).Model(&Provider{}).Where("id = ?", id).Updates(map[string]any{
		"verified":        p.Verified,
		"last_tested_at":  p.LastTestedAt,
		"last_latency_ms": p.LastLatencyMS,
		"last_error":      p.LastError,
	}).Error
	return result, nil
}

// TestDraft 测试尚未保存的配置；id 非 0 时可复用已保存的 Key。
func (s *Service) TestDraft(ctx context.Context, id uint, in SaveInput) (TestResult, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	if err := s.ValidateBaseURL(baseURL); err != nil {
		return TestResult{OK: false, Message: err.Error()}, nil
	}
	key := strings.TrimSpace(in.APIKey)
	if key == "" && id > 0 {
		if existing, err := dalFindProvider(ctx, s.db, id); err == nil {
			key, _ = DecryptAPIKey(s.secret, existing.APIKeyEnc)
		}
	}
	headers := in.Headers
	if headers == nil && id > 0 {
		if existing, err := dalFindProvider(ctx, s.db, id); err == nil {
			headers = ProviderHeaders(existing)
		}
	}
	return testEndpoint(ctx, baseURL, key, headers), nil
}

// ValidateBaseURL 校验 Base URL 并做基础 SSRF 防护。
// 开发模式允许 http 与 127.0.0.1；生产要求 HTTPS 且禁止私网地址。
func (s *Service) ValidateBaseURL(raw string) error {
	if raw == "" {
		return ErrInvalidBaseURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ErrInvalidBaseURL
	}
	if u.User != nil {
		return fmt.Errorf("%w: 不能包含账号信息", ErrInvalidBaseURL)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("%w: 必须使用 http 或 https", ErrInvalidBaseURL)
	}
	if !s.devMode && u.Scheme != "https" {
		return fmt.Errorf("%w: 生产环境必须使用 HTTPS", ErrInvalidBaseURL)
	}
	if s.devMode {
		return nil
	}
	host := u.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("%w: 域名无法解析", ErrInvalidBaseURL)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return fmt.Errorf("%w: 生产环境禁止使用私网地址", ErrInvalidBaseURL)
		}
	}
	return nil
}

// replaceModels 全量替换某厂商的模型列表。
func (s *Service) replaceModels(ctx context.Context, tx *gorm.DB, providerID uint, inputs []ModelInput) error {
	if err := tx.WithContext(ctx).Where("provider_id = ?", providerID).Delete(&Model{}).Error; err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(inputs))
	for idx, in := range inputs {
		m, err := buildModel(providerID, in, (idx+1)*10)
		if err != nil {
			return err
		}
		if _, ok := seen[m.ModelID]; ok {
			continue
		}
		seen[m.ModelID] = struct{}{}
		if err := tx.WithContext(ctx).Create(m).Error; err != nil {
			return err
		}
	}
	return nil
}

// syncRoutingFlags 把默认 / 兜底标记同步到模型行，便于列表直接展示。
func (s *Service) syncRoutingFlags(ctx context.Context, defaultID, fallbackID uint) error {
	if err := s.db.WithContext(ctx).Model(&Model{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Model(&Model{}).Where("is_fallback = ?", true).Update("is_fallback", false).Error; err != nil {
		return err
	}
	if defaultID > 0 {
		if err := s.db.WithContext(ctx).Model(&Model{}).Where("id = ?", defaultID).Update("is_default", true).Error; err != nil {
			return err
		}
	}
	if fallbackID > 0 {
		if err := s.db.WithContext(ctx).Model(&Model{}).Where("id = ?", fallbackID).Update("is_fallback", true).Error; err != nil {
			return err
		}
	}
	return nil
}

// uniqueCode 由名称生成唯一 code；冲突时追加序号。
func (s *Service) uniqueCode(ctx context.Context, name string) string {
	base := slugify(name)
	if base == "" {
		base = "provider"
	}
	code := base
	for i := 2; i < 100; i++ {
		var count int64
		_ = s.db.WithContext(ctx).Model(&Provider{}).Where("code = ?", code).Count(&count).Error
		if count == 0 {
			return code
		}
		code = fmt.Sprintf("%s-%d", base, i)
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

// testEndpoint 调用 OpenAI 兼容的 GET /models 做连通性测试。
// 返回结果不包含 Key；失败时给出可读原因。
func testEndpoint(ctx context.Context, baseURL, apiKey string, headers map[string]string) TestResult {
	start := time.Now()
	if strings.TrimSpace(apiKey) == "" {
		return TestResult{OK: false, Message: "未配置 API Key，无法完成连接测试"}
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	endpoint := strings.TrimRight(baseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return TestResult{OK: false, Message: "Base URL 不合法"}
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		return TestResult{OK: false, LatencyMS: latency, Message: "无法连接上游：" + sanitizeNetErr(err)}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return TestResult{OK: false, LatencyMS: latency, Message: fmt.Sprintf("连接失败（HTTP %d）：%s", resp.StatusCode, sanitizeBody(body))}
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return TestResult{OK: false, LatencyMS: latency, Message: "上游返回格式不是 OpenAI 兼容的模型列表"}
	}
	models := make([]string, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		if item.ID != "" {
			models = append(models, item.ID)
		}
	}
	return TestResult{
		OK:          true,
		LatencyMS:   latency,
		ModelsFound: len(models),
		Models:      models,
		Message:     fmt.Sprintf("连接成功，发现 %d 个模型", len(models)),
	}
}

// sanitizeBody 截断并清洗上游错误体，避免把敏感信息透给前端。
func sanitizeBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	text = strings.ReplaceAll(text, "\n", " ")
	if len([]rune(text)) > 120 {
		text = string([]rune(text)[:120]) + "…"
	}
	if text == "" {
		return "上游未返回错误详情"
	}
	return text
}

// sanitizeNetErr 隐藏具体网络错误中的内部地址细节，只保留可读方向。
func sanitizeNetErr(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline") {
		return "连接超时（10 秒）"
	}
	if strings.Contains(msg, "no such host") {
		return "域名无法解析"
	}
	if strings.Contains(msg, "connection refused") {
		return "连接被拒绝"
	}
	return "网络不可达"
}

// buildModel 校验并构造模型行。
func buildModel(providerID uint, in ModelInput, sortOrder int) (*Model, error) {
	modelID := strings.TrimSpace(in.ModelID)
	if modelID == "" {
		return nil, errors.New("model_id is required")
	}
	contextWindow := in.ContextWindow
	if contextWindow <= 0 {
		contextWindow = 32768
	}
	concurrency := in.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 2
	}
	concurrency = clampInt(concurrency, 1, 10000)
	return &Model{
		ProviderID:     providerID,
		ModelID:        modelID,
		DisplayName:    displayName(in),
		Description:    strings.TrimSpace(in.Description),
		ContextWindow:  contextWindow,
		MaxConcurrency: concurrency,
		Enabled:        in.Enabled,
		SortOrder:      sortOrder,
	}, nil
}

// displayName 返回模型展示名，缺省时用上游 ID。
func displayName(in ModelInput) string {
	if name := strings.TrimSpace(in.DisplayName); name != "" {
		return name
	}
	return strings.TrimSpace(in.ModelID)
}

// normalizeKind 收敛未知厂商类型。
func normalizeKind(kind string) string {
	switch kind {
	case KindDeepSeek, KindOpenAI, KindAliyun, KindMoonshot, KindCustom:
		return kind
	default:
		return KindCustom
	}
}

// slugify 把名称转为 code 可用的 ASCII 标识；中文名回退为 provider。
func slugify(name string) string {
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	lower := strings.ToLower(strings.TrimSpace(name))
	code := reg.ReplaceAllString(lower, "-")
	code = strings.Trim(code, "-")
	if len(code) > 32 {
		code = code[:32]
	}
	return code
}

// clampInt 将数值限制在 [min, max]。
func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// truncate 按字符截断，用于错误信息落库。
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

// ProviderHeaders 解析厂商的自定义请求头；数据损坏时返回空，不影响调用。
func ProviderHeaders(p *Provider) map[string]string {
	if p == nil || strings.TrimSpace(p.HeadersJSON) == "" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(p.HeadersJSON), &headers); err != nil {
		return nil
	}
	return headers
}

// marshalHeaders 校验并序列化自定义请求头。
// 名字必须是合法的 HTTP 头字段名，值里禁止换行，避免请求头注入。
func marshalHeaders(headers map[string]string) (string, error) {
	clean := make(map[string]string, len(headers))
	for name, value := range headers {
		key := strings.TrimSpace(name)
		if key == "" {
			continue
		}
		if !validHeaderName(key) {
			return "", fmt.Errorf("非法的请求头名称: %s", key)
		}
		if strings.ContainsAny(value, "\r\n") {
			return "", fmt.Errorf("请求头 %s 的值不能包含换行", key)
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		// 这些头由 HTTP 客户端自己管理，交给用户填会直接把请求写坏
		switch strings.ToLower(key) {
		case "content-length", "transfer-encoding", "connection", "host":
			continue
		}
		clean[key] = trimmed
	}
	if len(clean) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// validHeaderName 按 RFC 7230 的 token 规则校验头名称。
func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", r):
		default:
			return false
		}
	}
	return true
}

// DeleteModel 删除单个模型；预置模型只允许停用，不能删除。
func (s *Service) DeleteModel(ctx context.Context, id uint) error {
	var model Model
	if err := s.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrModelNotFound
		}
		return err
	}
	var prov Provider
	if err := s.db.WithContext(ctx).First(&prov, model.ProviderID).Error; err != nil {
		return err
	}
	if prov.IsBuiltin && prov.IsDefault && model.IsDefault {
		return ErrBuiltinDelete
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if model.IsDefault || model.IsFallback {
			if err := s.SetRouting(ctx, 0, 0); err != nil {
				return err
			}
		}
		return tx.Delete(&Model{}, id).Error
	})
}

// NormalizeAPIStyle 校验接入方式，非法值回退到 chat（OpenAI 兼容）。
func NormalizeAPIStyle(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case APIStyleResponses:
		return APIStyleResponses
	case APIStyleAnthropic:
		return APIStyleAnthropic
	default:
		return APIStyleChat
	}
}
