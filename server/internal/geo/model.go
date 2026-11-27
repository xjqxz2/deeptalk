// Package geo 把客户端公网 IP 解析成「用户在哪里」，供对话注入与登录记录使用。
// 只做粗粒度到城市一级的定位：不采集精确坐标，也不依赖浏览器授权。
package geo

import "time"

// Location 是一次定位结果。
type Location struct {
	City     string // 城市，如「杭州」
	Region   string // 省/州，如「浙江」
	Country  string // 国家，如「中国」
	Timezone string // IANA 时区，如 Asia/Shanghai
}

// Display 返回给用户看的字符串，例如「中国 浙江 杭州」。
func (l Location) Display() string {
	parts := make([]string, 0, 3)
	for _, part := range []string{l.Country, l.Region, l.City} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += " " + part
	}
	return out
}

// Label 返回简短的地区名，优先城市，其次省份。
func (l Location) Label() string {
	if l.City != "" {
		return l.City
	}
	if l.Region != "" {
		return l.Region
	}
	return l.Country
}

// cacheEntry 是定位结果的进程内缓存。
type cacheEntry struct {
	location Location
	expires  time.Time
}
