package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gnemet/datagrid/database/cursorpool"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	dbMode := os.Getenv("DB_MODE")
	if dbMode == "" {
		dbMode = cursorpool.ModeConnectless
	}

	fmt.Printf("Starting TestApp in %s mode...\n", dbMode)

	// DB connection string from env
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	pool, err := cursorpool.NewCursorPool(connStr, 10, 5*time.Minute, 1*time.Hour)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	testQuery := "SELECT id, name, email FROM datagrid.personnel LIMIT 100"

	if dbMode == cursorpool.ModeRefCursor {
		runRefCursorTest(ctx, pool, testQuery)
	} else {
		runConnectlessTest(ctx, pool, testQuery)
	}
}

func runConnectlessTest(ctx context.Context, pool *cursorpool.CursorPool, query string) {
	fmt.Println("Running Connectless Test...")
	results, err := pool.QueryDirect(ctx, query)
	if err != nil {
		log.Fatalf("Connectless test failed: %v", err)
	}

	fmt.Printf("Fetched %d rows directly.\n", len(results))
	for i, row := range results {
		if i >= 3 {
			break
		}
		fmt.Printf("  Row %d: %v\n", i+1, row)
	}
}

func runRefCursorTest(ctx context.Context, pool *cursorpool.CursorPool, query string) {
	fmt.Println("Running RefCursor Test...")
	sid := "test-session"

	_, err := pool.InitializeCursor(ctx, sid, query)
	if err != nil {
		log.Fatalf("Failed to initialize cursor: %v", err)
	}

	fmt.Println("Fetching first 10 rows...")
	results, err := pool.FetchNext(ctx, sid, 10)
	if err != nil {
		log.Fatalf("Fetch 1 failed: %v", err)
	}
	fmt.Printf("  Got %d rows.\n", len(results))

	fmt.Println("Fetching next 10 rows...")
	results2, err := pool.FetchNext(ctx, sid, 10)
	if err != nil {
		log.Fatalf("Fetch 2 failed: %v", err)
	}
	fmt.Printf("  Got %d rows.\n", len(results2))

	if len(results) > 0 && len(results2) > 0 && results[0]["id"] == results2[0]["id"] {
		fmt.Println("Warning: Fetched the same data! Something is wrong with cursor movement.")
	} else {
		fmt.Println("Success: Fetched different pages.")
	}
}
