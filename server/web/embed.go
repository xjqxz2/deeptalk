// Package web 内嵌两个前端的构建产物：dist/c（C 端）与 dist/b（B 端）。
// 构建时由 Makefile 将 ui/c/dist 与 ui/b/dist 拷贝进来；仓库中保留占位文件保证 go build 可用。
package web

import "embed"

// FS 是前端静态资源文件系统。
//
//go:embed all:dist/c all:dist/b
var FS embed.FS
