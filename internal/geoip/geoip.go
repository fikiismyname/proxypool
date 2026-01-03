package geoip

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

type Service struct {
	db *geoip2.Reader
}

func New(dbPath string) (*Service, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open geoip db: %w", err)
	}

	return &Service{db: db}, nil
}

func (s *Service) Close() error {
	return s.db.Close()
}

func (s *Service) Lookup(ipStr string) (string, string, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", "", fmt.Errorf("invalid IP address: %s", ipStr)
	}

	record, err := s.db.City(ip)
	if err != nil {
		return "", "", fmt.Errorf("geoip lookup failed: %w", err)
	}

	return record.Country.IsoCode, record.Country.Names["en"], nil
}
