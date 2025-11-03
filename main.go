package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
)

var ipAddr string

const (
	DEFAULT_MODE        = 1
	DEFAULT_DIR         = "."
	DEFAULT_HOST        = "0.0.0.0"
	DEFAULT_PORT        = 8090
	DEFAULT_IPORT       = 8091
	DEFAULT_ROOT_PREFIX = ""
)

type ProxyServer struct {
	Name         string     `json:"name"`
	ID           string     `json:"id"`
	Location     string     `json:"location"`
	ProviderName string     `json:"providerName"`
	ProviderLink string     `json:"providerLink"`
	Plan         string     `json:"plan"`
	SpeedRate    string     `json:"speedRate"`
	Limit        string     `json:"limit"`
	InfoLink     string     `json:"infoLink"`
	Proxy        ProxyLinks `json:"proxy"`
}

type ProxyLinks struct {
	Vless []string `json:"vless"`
	Http  []string `json:"http"`
	Socks []string `json:"socks"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		envfile, err := os.Create(".env")
		if err != nil {
			log.Fatalf("Failed to create env file: %v", err)
		}
		envfile.WriteString(`TELEGRAM_BOT_TOKEN=0
TELEGRAM_BOT_OWNER_ID=0
TELEGRAM_BOT_ACCESS_CODE=0`)
		log.Fatal("Error loading .env file")
	}

	fmt.Println(flag.Args())

	dir := flag.String("dir", DEFAULT_DIR, "server directory")
	host := flag.String("host", DEFAULT_HOST, "server host")
	port := flag.Int("port", DEFAULT_PORT, "server port")
	prefix := flag.String("prefix", DEFAULT_ROOT_PREFIX, "server root prefix")

	ihost := flag.String("ihost", DEFAULT_HOST, "info server host")
	iport := flag.Int("iport", DEFAULT_IPORT, "info server port")

	mode := flag.Int("mode", DEFAULT_MODE, "mode")

	flag.Parse()

	resp, err := http.Get("https://ifconfig.me/ip")
	if err != nil {
		log.Fatalf("Failed to get ifconfig request: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read body: %v", err)
	}

	ipAddr = strings.TrimSpace(string(data))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go RunInfoServer(ctx, stop, &InfoServerParams{
		Host: *ihost,
		Port: *iport,
	})

	if *mode > 1 {
		telebotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
		if telebotToken == "" {
			log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
		}
		telebotOwnerID := os.Getenv("TELEGRAM_BOT_OWNER_ID")
		if telebotOwnerID == "" {
			log.Fatal("TELEGRAM_BOT_OWNER_ID environment variable is not set")
		}
		telebotAccessCode := os.Getenv("TELEGRAM_BOT_ACCESS_CODE")
		if telebotAccessCode == "" {
			log.Fatal("TELEGRAM_BOT_ACCESS_CODE environment variable is not set")
		}

		go RunServer(ctx, stop, &ServerParams{
			Dir:    *dir,
			Host:   *host,
			Port:   *port,
			Prefix: *prefix,
		})

		go RunTelebot(ctx, stop, &TelebotParams{
			Token:         telebotToken,
			OwnerID:       telebotOwnerID,
			AccessCode:    telebotAccessCode,
			WebApp:        "https://core.telegram.org/",
			UsersFilePath: "telebotusers.db",
		})
	}

	<-ctx.Done()
	log.Println("Application exited cleanly.")
}
