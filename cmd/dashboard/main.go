package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/yikkrrtykj/vps-netwatch/internal/config"
	"github.com/yikkrrtykj/vps-netwatch/internal/server"
	"github.com/yikkrrtykj/vps-netwatch/internal/store"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := store.Open(cfg.Dashboard.DataPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()

	srv := server.New(cfg, db)
	log.Printf("vps-netwatch dashboard listening on %s", cfg.Dashboard.Listen)
	if err := http.ListenAndServe(cfg.Dashboard.Listen, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}

