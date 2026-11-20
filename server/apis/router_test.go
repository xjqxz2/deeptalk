package apis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

// TestServeFrontendFilePWAHeaders 锁定 PWA 文件不能被长期缓存：
// sw.js / manifest 一旦被 immutable 缓存，发版后会持续运行旧版本。
func TestServeFrontendFilePWAHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	files := fstest.MapFS{
		"sw.js":                &fstest.MapFile{Data: []byte("self.addEventListener('install',()=>{})")},
		"manifest.webmanifest": &fstest.MapFile{Data: []byte(`{"name":"DeepTalk"}`)},
		"offline.html":         &fstest.MapFile{Data: []byte("<html></html>")},
		"icons/icon-192.png":   &fstest.MapFile{Data: []byte("png")},
		"assets/app.js":        &fstest.MapFile{Data: []byte("console.log(1)")},
	}
	router := gin.New()
	router.GET("/*path", func(c *gin.Context) {
		serveFrontendFile(c, files, strings.TrimPrefix(c.Param("path"), "/"), true)
	})

	cases := []struct {
		path        string
		cache       string
		contentType string
		workerScope string
	}{
		{path: "/sw.js", cache: "no-cache", contentType: "text/javascript", workerScope: "/"},
		{path: "/manifest.webmanifest", cache: "no-cache", contentType: "application/manifest+json"},
		{path: "/offline.html", cache: "no-cache", contentType: "text/html"},
		{path: "/icons/icon-192.png", cache: "max-age=604800"},
		{path: "/assets/app.js", cache: "immutable"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", recorder.Code)
			}
			if got := recorder.Header().Get("Cache-Control"); !strings.Contains(got, tc.cache) {
				t.Fatalf("Cache-Control = %q, want contains %q", got, tc.cache)
			}
			if tc.contentType != "" {
				if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, tc.contentType) {
					t.Fatalf("Content-Type = %q, want contains %q", got, tc.contentType)
				}
			}
			if tc.workerScope != "" {
				if got := recorder.Header().Get("Service-Worker-Allowed"); got != tc.workerScope {
					t.Fatalf("Service-Worker-Allowed = %q, want %q", got, tc.workerScope)
				}
			}
		})
	}
}

// TestServeFrontendFileAdminKeepsImmutable 管理后台不注册 PWA，仍使用长期缓存。
func TestServeFrontendFileAdminKeepsImmutable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	files := fstest.MapFS{"assets/app.js": &fstest.MapFile{Data: []byte("console.log(1)")}}
	router := gin.New()
	router.GET("/*path", func(c *gin.Context) {
		serveFrontendFile(c, files, strings.TrimPrefix(c.Param("path"), "/"), false)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if got := recorder.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Fatalf("Cache-Control = %q, want immutable", got)
	}
}
