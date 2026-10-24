// SmsFM 控制台后端：局域网扫描、连接保存、SmsForwarder 协议适配。
// 用法：smsfm-server [-addr :8080] [-data ./data] [-web ../web/dist]
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"smsfm/server/internal/api"
	"smsfm/server/internal/auth"
	"smsfm/server/internal/store"
)

//go:embed all:web
var embeddedWeb embed.FS

func main() {
	addr := flag.String("addr", ":8080", "HTTP 监听地址")
	dataDir := flag.String("data", "data", "数据目录（连接配置）")
	webDir := flag.String("web", "", "前端目录（优先于嵌入版本；默认嵌入）")
	flag.Parse()

	st, err := store.Open(*dataDir)
	if err != nil {
		log.Fatalf("打开数据目录失败：%v", err)
	}
	authStore, err := auth.Open(*dataDir)
	if err != nil {
		log.Fatalf("初始化账号失败：%v", err)
	}

	var webFS fs.FS
	if *webDir != "" {
		webFS = os.DirFS(filepath.Clean(*webDir))
	} else {
		sub, err := fs.Sub(embeddedWeb, "web")
		if err != nil {
			log.Fatalf("嵌入前端不可用（请先构建 web/dist 到 server/web）：%v", err)
		}
		webFS = sub
	}

	srv := api.New(st, authStore, webFS)
	log.Printf("SmsFM 控制台后端启动：http://localhost%s（数据目录 %s）", *addr, *dataDir)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "服务退出：%v\n", err)
		os.Exit(1)
	}
}
