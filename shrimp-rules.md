# Development Guidelines

## Project Overview

This is a **World-Scale Proxy Pool** system written in Go.
**Goal**: Scrape, validate, store, and serve millions of proxies from various internet sources.
**Tech Stack**: Go (Golang), Supabase (Postgres), Redis (optional for hot cache), Colly/Chromedp/Playwright (Scraping).

## Project Architecture

Adhere to the **Standard Go Project Layout**:

- `cmd/`: Application entry points (e.g., `cmd/proxypool/main.go`).
- `internal/`: Private application and business logic.
  - `internal/model`: Domain models (Proxy, CheckResult).
  - `internal/scraper`: Scraper engine and source adapters.
  - `internal/checker`: Validation logic (latency, anonymity, stability).
  - `internal/storage`: Database repositories (Supabase/Postgres).
  - `internal/api`: REST/gRPC server handlers.
- `configs/`: Configuration management (env vars, config structs).

## Code Standards

### Go Specific

1.  **Context Usage**: ALL network operations (scraping, database, checking) **MUST** accept `context.Context` as the first argument. Simplifies timeout and cancellation management.
2.  **Error Handling**:
    - Use `fmt.Errorf("op: %w", err)` to wrap errors.
    - Don't ignore errors with `_` unless explicitly documented why it's safe.
3.  **Concurrency**:
    - Use `errgroup` or `sync.WaitGroup` to manage goroutines.
    - Avoid goroutine leaks by ensuring exit conditions.
4.  **Logging**: Use `log/slog` for structured logging.

### Functionality Implementation

1.  **Scraper Interface**: Implement a `Source` interface for all proxy sources. This allows easy plugin-like addition of new sites.
2.  **Validation**: Proxies must be validated for:
    - **Liveness**: Connect to a known target (e.g., Google/Cloudflare).
    - **Protocol**: Detect HTTP/S, SOCKS4, SOCKS5 automatically.
    - **Anonymity**: Check headers to verify High Anon vs Transparent.

## Tools & MCP

1.  **Supabase**: Use `supabase-mcp-server` for DB migrations and queries.
2.  **Task Management**: Strictly follow `shrimp-task-manager` workflows.
3.  **Browser**: Use `Playwright` via MCP for hard-to-scrape dynamic sites (like ones with heavy JS obfuscation).

## File Management

- **Config**: Use `.env` for secrets. NEVER commit secrets.
- **Root**: Keep root directory clean. Only `go.mod`, `go.sum`, `README.md`, `shrimp-rules.md`, `task.md` live here.
