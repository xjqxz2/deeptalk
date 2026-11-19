// DeepTalk 服务端入口：只负责拉起 bootstrap，不写路由、不连数据库。
package main

import (
	"fmt"
	"os"

	"deeptalk/bootstrap"
)

// 由 Makefile 通过 -ldflags 注入的构建信息。
var (
	version   = "dev"     // 版本号
	commit    = "unknown" // 提交号
	buildTime = "unknown" // 构建时间
)

// main 启动服务并等待退出信号。
func main() {
	fmt.Fprintf(os.Stderr, "DeepTalk %s (%s, %s)\n", version, commit, buildTime)
	app := bootstrap.New()
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "startup failed: %v\n", err)
		os.Exit(1)
	}
	app.Waiting()
}
