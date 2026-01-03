package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"proxypool/internal/checker"
	"proxypool/internal/geoip"
	"proxypool/internal/model"
	"proxypool/internal/scraper"
	"proxypool/internal/storage"
)

type Config struct {
	NumWorkers int
	BatchSize  int
}

type Engine struct {
	repo    storage.ProxyRepository
	sources []scraper.Source
	chk     *checker.Checker
	geo     *geoip.Service
	cfg     Config
}

func New(repo storage.ProxyRepository, srcList []scraper.Source, chk *checker.Checker, geo *geoip.Service, cfg Config) *Engine {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 50
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	return &Engine{
		repo:    repo,
		sources: srcList,
		chk:     chk,
		geo:     geo,
		cfg:     cfg,
	}
}

func (e *Engine) Run(ctx context.Context) {
	var wg sync.WaitGroup

	// 1. Scrape Scheduler (Runs periodically)
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.runScrapingLoop(ctx)
	}()

	// Pipeline: DB -> (jobs) -> Workers -> (results) -> DB Writer

	// Channels
	jobChan := make(chan *model.Proxy, e.cfg.BatchSize*2)
	resultChan := make(chan *model.Proxy, e.cfg.BatchSize*2)

	// 2. DB Producer (Fetches unchecked proxies)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(jobChan)
		e.runProducer(ctx, jobChan)
	}()

	// 3. Check Workers
	// Start N workers
	workerWg := &sync.WaitGroup{}
	workerWg.Add(e.cfg.NumWorkers)
	for i := 0; i < e.cfg.NumWorkers; i++ {
		go func() {
			defer workerWg.Done()
			e.runWorker(ctx, jobChan, resultChan)
		}()
	}

	// Wait for workers to finish (when jobChan closes), then close resultChan
	go func() {
		workerWg.Wait()
		close(resultChan)
	}()

	// 4. DB Writer (Batch Updates)
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.runWriter(ctx, resultChan)
	}()

	wg.Wait()
	slog.Info("Engine Stopped")
}

// runScrapingLoop periodically scrapes proxies
func (e *Engine) runScrapingLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	// Run once immediately
	e.scrapeAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.scrapeAll(ctx)
		}
	}
}

func (e *Engine) scrapeAll(ctx context.Context) {
	for _, src := range e.sources {
		if ctx.Err() != nil {
			return
		}
		slog.Info("Scraping", "source", src.Name())
		proxies, err := src.Fetch(ctx)
		if err != nil {
			slog.Error("Scrape failed", "source", src.Name(), "error", err)
			continue
		}
		if len(proxies) > 0 {
			if err := e.repo.SaveBatch(ctx, proxies); err != nil {
				slog.Error("SaveBatch failed", "error", err)
			} else {
				slog.Info("Saved proxies", "count", len(proxies), "source", src.Name())
			}
		}
	}
}

// runProducer fetches proxies from DB and sends to jobChan
func (e *Engine) runProducer(ctx context.Context, jobChan chan<- *model.Proxy) {
	ticker := time.NewTicker(1 * time.Second) // Poll DB frequently
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Fetch batch
			proxies, err := e.repo.GetProxiesToCheck(ctx, e.cfg.BatchSize)
			if err != nil {
				slog.Error("Producer fetch failed", "error", err)
				continue
			}
			if len(proxies) == 0 {
				continue // Nothing to do, wait for next tick
			}
			
			// Push to queue. Blocking if queue full (which provides backpressure to DB fetching)
			for _, p := range proxies {
				select {
				case jobChan <- p:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// runWorker reads jobs, checks proxy, sends to resultChan
func (e *Engine) runWorker(ctx context.Context, jobChan <-chan *model.Proxy, resultChan chan<- *model.Proxy) {
	for p := range jobChan {
		if ctx.Err() != nil {
			return
		}

		res, err := e.chk.Check(ctx, p)
		now := time.Now()
		p.LastCheckedAt = &now

		if err != nil || !res.Alive {
			p.LatencyMS = 0 // Dead
		} else {
			p.LatencyMS = res.LatencyMS
			if e.geo != nil {
				iso, _, err := e.geo.Lookup(p.IP)
				if err == nil && iso != "" {
					p.Country = iso
				}
			}
		}

		select {
		case resultChan <- p:
		case <-ctx.Done():
			return
		}
	}
}

// runWriter collects results and periodically batch updates DB
func (e *Engine) runWriter(ctx context.Context, resultChan <-chan *model.Proxy) {
	batch := make([]*model.Proxy, 0, e.cfg.BatchSize)
	ticker := time.NewTicker(5 * time.Second) // Force flush interval
	defer ticker.Stop()

	flush := func() {
		if len(batch) > 0 {
			if err := e.repo.UpdateBatch(ctx, batch); err != nil {
				slog.Error("Writer batch update failed", "count", len(batch), "error", err)
			} else {
				slog.Info("Updated batch", "count", len(batch))
			}
			batch = batch[:0] // clear
		}
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush()
		case p, ok := <-resultChan:
			if !ok {
				flush()
				return
			}
			batch = append(batch, p)
			if len(batch) >= e.cfg.BatchSize {
				flush()
			}
		}
	}
}
