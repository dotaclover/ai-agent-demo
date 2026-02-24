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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// API 路由
	handler := api.NewHandler()
	handler.RegisterRoutes(mux)

	// 前端页面
	webSub, _ := fs.Sub(webFS, "web")
	fileServer := http.FileServer(http.FS(webSub))
	mux.Handle("/", fileServer)

	log.Printf("AI 创意助手启动在 http://localhost:%s", port)
	if err := http.ListenAndServe("localhost:"+port, mux); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
