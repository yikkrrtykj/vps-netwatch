package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yikkrrtykj/vps-netwatch/internal/agent"
	"github.com/yikkrrtykj/vps-netwatch/internal/config"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config yaml")
	agentID := flag.String("id", "main-vps", "agent id")
	once := flag.Bool("once", false, "collect once and exit")
	printOnly := flag.Bool("print", false, "print payload instead of pushing")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runner := agent.New(cfg)
	interval := agent.Interval(cfg, *agentID)
	run := func() {
		collectCtx, cancel := context.WithTimeout(ctx, interval)
		defer cancel()
		push := runner.Collect(collectCtx, *agentID)
		if *printOnly {
			_ = json.NewEncoder(os.Stdout).Encode(push)
			return
		}
		if err := runner.Push(collectCtx, push, agent.DashboardURL(cfg, *agentID)); err != nil {
			log.Printf("push failed: %v", err)
			return
		}
		log.Printf("pushed agent status for %s", *agentID)
	}

	run()
	if *once {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

