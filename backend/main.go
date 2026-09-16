package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"jade-gamble/backend/api"
	"jade-gamble/backend/store"
)

func main() {
	port := getenv("PORT", "3002")
	dbPath := getenv("DB_PATH", "./data/jade.db")
	if err := os.MkdirAll(dbPath[:strings.LastIndex(dbPath, "/")], 0o755); err != nil {
		// windows paths may use backslashes
		_ = err
	}
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer st.Close()

	// 維護指令：在容器內執行（單一寫入者），清掉測試帳號與 bot
	if len(os.Args) > 1 && os.Args[1] == "-purge-mock" {
		res, err := st.PurgeMockUsers()
		if err != nil {
			log.Fatalf("purge: %v", err)
		}
		log.Printf("已清理: %v", res)
		return
	}

	a := &api.API{
		Store:               st,
		BaseURL:             getenv("BASE_URL", "http://localhost:3002"),
		DiscordClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		DiscordClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		DiscordRedirectURI:  os.Getenv("DISCORD_REDIRECT_URI"),
		DiscordGuildID:      os.Getenv("DISCORD_GUILD_ID"),
		MockAuth:            os.Getenv("MOCK_AUTH") == "1",
		AdminDiscordIDs:     splitIDs(os.Getenv("ADMIN_DISCORD_IDS")),
		// 喵喵錢莊的 AI：金鑰只從環境變數來（.env，已 gitignore），永不進 repo
		AIBaseURL: getenv("AI_BASE_URL", "https://api.deepseek.com/v1"),
		AIAPIKey:  os.Getenv("AI_API_KEY"),
		AIModel:   getenv("AI_MODEL", "deepseek-chat"),
	}

	mux := a.Routes()
	mux.Handle("/", spaHandler())

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 拍賣 bot 的背景節奏：就算沒人開市場頁，放太久的料也會被收走
	go func() {
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for range t.C {
			a.SettleMarketBots()
		}
	}()

	go func() {
		log.Printf("jade-gamble listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// spaHandler serves the embedded frontend (prod) — index.html, app.js,
// stone-render.js — and falls back to index.html for unknown non-API paths.
func spaHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		data, err := frontendFS.ReadFile("frontend/" + name)
		if err != nil {
			// SPA fallback to index
			data, err = frontendFS.ReadFile("frontend/index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			name = "index.html"
		}
		ctype := "application/octet-stream"
		switch {
		case strings.HasSuffix(name, ".html"):
			ctype = "text/html; charset=utf-8"
		case strings.HasSuffix(name, ".js"):
			ctype = "text/javascript; charset=utf-8"
		case strings.HasSuffix(name, ".css"):
			ctype = "text/css"
		}
		w.Header().Set("Content-Type", ctype)
		// no-store：前端沒有 build step，改了就必須立刻生效。
		// （no-cache 只是「可以重用但需驗證」，沒有 ETag 時瀏覽器可能繼續用舊檔，
		//   玩家就會看到新 HTML 配舊 JS——導航按鈕變空白就是這樣來的。）
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		_, _ = w.Write(data)
	})
}

//go:embed frontend
var frontendFS embed.FS

var _ = strconv.Itoa

// splitIDs: 把 "123,456" 這種管理員名單拆開。
func splitIDs(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
