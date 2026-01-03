package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"proxypool/internal/model"

	"github.com/joho/godotenv"
)

func setupTestDB(t *testing.T) *PostgresRepository {
	// Load env from project root
	// Assuming test is running from inside internal/storage so we look up 2 levels
	_ = godotenv.Load("../../.env")
	
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	repo, err := NewPostgresRepository(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	return repo
}

func TestSaveAndGet(t *testing.T) {
	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()

	// 1. Create dummy proxies with random details to avoid conflicts
	rnd := int(time.Now().UnixNano() % 10000)
	ip := fmt.Sprintf("127.0.%d.1", rnd%255)
	port := 8000 + rnd
	proxies := []*model.Proxy{
		{
			IP:       ip,
			Port:     port,
			Protocol: "http",
		},
	}

	// 2. Save Batch
	err := repo.SaveBatch(ctx, proxies)
	if err != nil {
		t.Fatalf("SaveBatch failed: %v", err)
	}

	// Verify insertion using direct query (since GetProxiesToCheck might return other pending proxies)
	var id int64
	query := `SELECT id FROM proxies WHERE ip=$1 AND port=$2`
	// We need to use the pool to query. Accessing private field 'pool'? 
	// The test is in 'storage' package, so it has access to 'pool' if it's in the same package scope.
	// postgres_test.go is package storage.
	err = repo.pool.QueryRow(ctx, query, ip, port).Scan(&id)
	if err != nil {
		t.Fatalf("Inserted proxy not found in DB: %v", err)
	}
	proxies[0].ID = id

	// 3. GetUnchecked
	// We verify it returns *something* if DB is populated, 
	// and specifically check if our proxy is there IF the DB was empty (hard to guarantee).
	// So we just check usage.
	toCheck, err := repo.GetProxiesToCheck(ctx, 10)
	if err != nil {
		t.Fatalf("GetProxiesToCheck failed: %v", err)
	}
	if len(toCheck) == 0 {
		// If DB was empty except our proxy, we should get it.
		// If we got 0, that's weird because we just inserted one.
		t.Fatalf("GetProxiesToCheck returned 0 items")
	}

	// 4. UpdateBatch Test
	// We will update the proxies we found (or just our inserted one if we want to be safe)
	// Let's test UpdateBatch on the inserted proxy specifically.
	now := time.Now()
	proxies[0].LatencyMS = 150
	proxies[0].LastCheckedAt = &now
	proxies[0].Country = "ID"
	
	err = repo.UpdateBatch(ctx, proxies)
	if err != nil {
		t.Fatalf("UpdateBatch failed: %v", err)
	}

	fmt.Println("Test passed: Inserted, verified, and Batch Updated proxy.")
}
