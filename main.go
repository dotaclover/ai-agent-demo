package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"agent-demo/api"
)

//go:embed web/*
var webFS embed.FS

func main() {
	host := os.Getenv("HOST")
	if host == "" {
		host = "" // 监听所有网络接口
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "58712"
	}

	mux := http.NewServeMux()

	// API 路由
	handler := api.NewHandler()
	handler.RegisterRoutes(mux)

	// 前端页面
	webSub, _ := fs.Sub(webFS, "web")
	fileServer := http.FileServer(http.FS(webSub))
	mux.Handle("/", fileServer)

	addr := host + ":" + port
	displayAddr := addr
	if host == "" {
		displayAddr = "localhost:" + port
	}
	
	log.Printf("AI 创意助手启动在 http://%s", displayAddr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
