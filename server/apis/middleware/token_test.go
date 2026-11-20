package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"deeptalk/internal/audit"
	"deeptalk/internal/setting"
	"deeptalk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestAuthAcceptsBearerToken 校验小程序/APP 可以用 Authorization 头鉴权。
func TestAuthAcceptsBearerToken(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "mw.db") + "?_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	for _, migrate := range []func(*gorm.DB) error{user.Migrate, setting.Migrate, audit.Migrate} {
		if err := migrate(db); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
	settings := setting.NewService(db)
	users := user.NewService(db, settings, audit.NewService(db), user.Options{SessionTTL: time.Hour, LockThreshold: 3, LockMinutes: 10})
	_, token, err := users.Register(context.Background(), user.RegisterInput{Email: "mw@example.com", Password: "pass-1234"}, user.RequestMeta{})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/whoami", Auth(users, Config{CookieName: "dt_session"}), func(c *gin.Context) {
		current := CurrentUser(c)
		if current == nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"email": current.Email})
	})

	cases := []struct {
		name   string
		header map[string]string
		want   int
	}{
		{"bearer", map[string]string{"Authorization": "Bearer " + token}, http.StatusOK},
		{"session-header", map[string]string{"X-Session-Token": token}, http.StatusOK},
		{"cookie", map[string]string{"Cookie": "dt_session=" + token}, http.StatusOK},
		{"missing", nil, http.StatusUnauthorized},
		{"wrong", map[string]string{"Authorization": "Bearer nope"}, http.StatusUnauthorized},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
		for key, value := range tc.header {
			req.Header.Set(key, value)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != tc.want {
			t.Fatalf("%s: status = %d, want %d", tc.name, recorder.Code, tc.want)
		}
	}
}
