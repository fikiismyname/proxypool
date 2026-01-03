package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"proxypool/configs"
	"proxypool/internal/checker"
	"proxypool/internal/engine"
	"proxypool/internal/geoip"
	"proxypool/internal/scraper"
	"proxypool/internal/scraper/sources"
	"proxypool/internal/storage"
)

func main() {
	// 1. Setup Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Load Config
	cfg, err := configs.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 3. Init Storage
	repo, err := storage.NewPostgresRepository(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	// 4. Init Components
	sourcesList := []scraper.Source{
		// TheSpeedX
		sources.NewGithubRawSource("TheSpeedX-HTTP", "https://raw.githubusercontent.com/TheSpeedX/PROXY-LIST/master/http.txt", "http"),
		sources.NewGithubRawSource("TheSpeedX-SOCKS4", "https://raw.githubusercontent.com/TheSpeedX/PROXY-LIST/master/socks4.txt", "socks4"),
		sources.NewGithubRawSource("TheSpeedX-SOCKS5", "https://raw.githubusercontent.com/TheSpeedX/PROXY-LIST/master/socks5.txt", "socks5"),
		// ProxyScraper
		sources.NewGithubRawSource("ProxyScraper-HTTP", "https://raw.githubusercontent.com/ProxyScraper/ProxyScraper/refs/heads/main/http.txt", "http"),
		sources.NewGithubRawSource("ProxyScraper-SOCKS4", "https://raw.githubusercontent.com/ProxyScraper/ProxyScraper/refs/heads/main/socks4.txt", "socks4"),
		sources.NewGithubRawSource("ProxyScraper-SOCKS5", "https://raw.githubusercontent.com/ProxyScraper/ProxyScraper/refs/heads/main/socks5.txt", "socks5"),
		// monosans
		sources.NewGithubRawSource("monosans-HTTP", "https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/http.txt", "http"),
		sources.NewGithubRawSource("monosans-SOCKS4", "https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks4.txt", "socks4"),
		sources.NewGithubRawSource("monosans-SOCKS5", "https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt", "socks5"),
		// komutan234
		sources.NewGithubRawSource("komutan234-HTTP", "https://raw.githubusercontent.com/komutan234/Proxy-List-Free/main/proxies/http.txt", "http"),
		sources.NewGithubRawSource("komutan234-SOCKS4", "https://raw.githubusercontent.com/komutan234/Proxy-List-Free/main/proxies/socks4.txt", "socks4"),
		sources.NewGithubRawSource("komutan234-SOCKS5", "https://raw.githubusercontent.com/komutan234/Proxy-List-Free/main/proxies/socks5.txt", "socks5"),
		// hookzof
		sources.NewGithubRawSource("hookzof-SOCKS5", "https://raw.githubusercontent.com/hookzof/socks5_list/master/proxy.txt", "socks5"),
		// sunny9577
		sources.NewGithubRawSource("sunny9577-HTTP", "https://sunny9577.github.io/proxy-scraper/generated/http_proxies.txt", "http"),
		sources.NewGithubRawSource("sunny9577-SOCKS4", "https://sunny9577.github.io/proxy-scraper/generated/socks4_proxies.txt", "socks4"),
		sources.NewGithubRawSource("sunny9577-SOCKS5", "https://sunny9577.github.io/proxy-scraper/generated/socks5_proxies.txt", "socks5"),
	}

	chk := checker.NewChecker("http://google.com", 5*time.Second)

	// Init GeoIP
	geo, err := geoip.New("data/GeoLite2-City.mmdb")
	if err != nil {
		slog.Warn("GeoIP disabled (DB not found or invalid)", "error", err)
	} else {
		defer geo.Close()
		slog.Info("GeoIP enabled")
	}

	// 5. Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutting down...")
		cancel()
	}()

	// 7. Initialize Engine
	eng := engine.New(repo, sourcesList, chk, geo, engine.Config{
		NumWorkers: 1000,
		BatchSize:  500,
	})

	// 8. Run Engine
	slog.Info("Starting ProxyPool Engine", "workers", 1000, "batch_size", 500)
	// Run blocking until context is cancelled
	eng.Run(ctx)
	
	slog.Info("Shutdown complete")
}
