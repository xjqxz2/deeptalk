package geo

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Options 是定位服务的构造参数。
type Options struct {
	Endpoint string        // 端点模板，{ip} 会被替换；默认 ipapi.co
	Timeout  time.Duration // 单次查询超时
	TTL      time.Duration // 结果缓存时长
	Fallback Location      // 私网或查询失败时的兜底位置
}

// DefaultEndpoint 是默认的免费定位服务，无需 Key。
const DefaultEndpoint = "https://ipapi.co/{ip}/json/"

// Service 把 IP 解析成地理位置，带进程内缓存与超时保护。
type Service struct {
	mu       sync.Mutex
	cache    map[string]cacheEntry
	endpoint string
	timeout  time.Duration
	ttl      time.Duration
	fallback Location
	client   *http.Client
}

// NewService 创建定位服务。
func NewService(opts Options) *Service {
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &Service{
		cache:    make(map[string]cacheEntry),
		endpoint: endpoint,
		timeout:  timeout,
		ttl:      ttl,
		fallback: opts.Fallback,
		client:   &http.Client{Timeout: timeout},
	}
}

// Fallback 返回兜底位置。
func (s *Service) Fallback() Location { return s.fallback }

// IsPublicIP 判断 IP 是否是值得查询的公网地址。
// 本机、私网、链路本地地址一律不查，直接走兜底。
func IsPublicIP(raw string) net.IP {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return nil
	}
	return ip
}

// Lookup 解析 IP 对应的位置；私网、查询失败或超时时返回兜底位置。
// 返回值第二项表示是否命中了真实定位（而不是兜底）。
func (s *Service) Lookup(ctx context.Context, rawIP string) (Location, bool) {
	ip := IsPublicIP(rawIP)
	if ip == nil {
		return s.fallback, false
	}
	key := ip.String()
	if cached, ok := s.cached(key); ok {
		return cached, true
	}
	location, ok := s.query(ctx, key)
	if !ok {
		return s.fallback, false
	}
	s.store(key, location)
	return location, true
}

// cached 读取未过期的缓存。
func (s *Service) cached(key string) (Location, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cache[key]
	if !ok || time.Now().After(entry.expires) {
		return Location{}, false
	}
	return entry.location, true
}

// store 写入缓存，顺便清掉过期条目，避免长期运行后无界增长。
func (s *Service) store(key string, location Location) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if len(s.cache) > 4096 {
		for k, entry := range s.cache {
			if now.After(entry.expires) {
				delete(s.cache, k)
			}
		}
	}
	s.cache[key] = cacheEntry{location: location, expires: now.Add(s.ttl)}
}

// query 调用定位服务。
func (s *Service) query(ctx context.Context, ip string) (Location, bool) {
	lookupCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	endpoint := strings.ReplaceAll(s.endpoint, "{ip}", url.PathEscape(ip))
	req, err := http.NewRequestWithContext(lookupCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Location{}, false
	}
	req.Header.Set("User-Agent", "DeepTalk/1.0 (+https://github.com/xjqxz2/deeptalk)")
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return Location{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Location{}, false
	}
	var payload struct {
		City        string `json:"city"`
		Region      string `json:"region"`
		CountryName string `json:"country_name"`
		Timezone    string `json:"timezone"`
		Error       bool   `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&payload); err != nil {
		return Location{}, false
	}
	if payload.Error {
		return Location{}, false
	}
	location := Location{
		City:     strings.TrimSpace(payload.City),
		Region:   strings.TrimSpace(payload.Region),
		Country:  strings.TrimSpace(payload.CountryName),
		Timezone: strings.TrimSpace(payload.Timezone),
	}
	if location.Display() == "" {
		return Location{}, false
	}
	return location, true
}
