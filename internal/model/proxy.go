package model

import (
	"fmt"
	"time"
)

// Protocol types
const (
	ProtocolHTTP   = "http"
	ProtocolHTTPS  = "https"
	ProtocolSOCKS4 = "socks4"
	ProtocolSOCKS5 = "socks5"
)

// Anonymity levels
const (
	AnonymityTransparent = "transparent"
	AnonymityAnonymous   = "anonymous"
	AnonymityElite       = "elite"
)

// Proxy represents a proxy server entity.
type Proxy struct {
	ID            int64      `json:"id" db:"id"`
	IP            string     `json:"ip" db:"ip"`
	Port          int        `json:"port" db:"port"`
	Protocol      string     `json:"protocol" db:"protocol"`
	Country       string     `json:"country" db:"country"`
	Anonymity     string     `json:"anonymity" db:"anonymity"`
	LatencyMS     int        `json:"latency_ms" db:"latency_ms"` // Latency in milliseconds
	LastCheckedAt *time.Time `json:"last_checked_at" db:"last_checked_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// Address returns the "ip:port" string.
func (p *Proxy) Address() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

// URL returns the full URL representation (e.g., "http://ip:port").
// If protocol is unknown, defaults to http.
func (p *Proxy) URL() string {
	proto := p.Protocol
	if proto == "" {
		proto = ProtocolHTTP
	}
	return fmt.Sprintf("%s://%s:%d", proto, p.IP, p.Port)
}
