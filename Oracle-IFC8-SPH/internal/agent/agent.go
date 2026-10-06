package agent

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
)

func Main() {
	envPath := flag.String("env", defaultEnvPath(), "Arquivo .env")
	once := flag.Bool("once", false, "Verifica uma vez, processa alertas e encerra")
	dryRun := flag.Bool("dry-run", false, "Simula envios sem Telegram")
	testTelegram := flag.Bool("test-telegram", false, "Envia uma mensagem de teste e encerra")
	flag.Parse()
	required := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "env" {
			required = true
		}
	})
	if err := execute(*envPath, required, *once, *dryRun, *testTelegram); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func execute(envPath string, required, once, dryRun, testTelegram bool) error {
	if dryRun && testTelegram {
		return fmt.Errorf("use -dry-run ou -test-telegram, nao ambos")
	}
	if err := loadEnvFile(envPath, required); err != nil {
		return err
	}
	cfg, err := loadConfig(dryRun)
	if err != nil {
		return err
	}
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("nao foi possivel obter hostname")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	send := func(_ context.Context, message string) error { fmt.Printf("[SIMULACAO]\n%s\n", message); return nil }
	if !dryRun {
		client := newTelegramClient(cfg.token, cfg.chatID, cfg.threadID)
		defer client.client.CloseIdleConnections()
		send = client.send
	}
	if testTelegram {
		return send(ctx, "Teste do monitor IFC8 - host: "+hostname)
	}
	fmt.Printf("Monitor IFC8 iniciado; intervalo %s; topico %s (ID %d).\n", cfg.interval, cfg.topic, cfg.threadID)
	state := alertState{}
	for {
		sample := collectHealth(ctx, cfg)
		if ctx.Err() != nil {
			return nil
		}
		now := time.Now()
		fmt.Printf("%s [%s] %s\n", now.Format("2006-01-02 15:04:05"), sample.status, sample.reason)
		err := state.notify(ctx, sample, cfg, hostname, now, send)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			log.Printf("Falha de envio: %v", err)
		}
		if once {
			return err
		}
		timer := time.NewTimer(cfg.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}
