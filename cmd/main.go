package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"saiij.distributed.rate.limiter/internal/api"
	"saiij.distributed.rate.limiter/internal/ratelimiter"
	"saiij.distributed.rate.limiter/internal/store"
)

func main() {
	// 1. Cliente Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer redisClient.Close()

	// Verificar conexión
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	log.Println("connected to redis at", redisAddr)

	// 2. Store
	redisStore := store.NewRedisStore(redisClient)

	// 3. Rate limiter
	cfg := &ratelimiter.RateLimiterConfig{
		Type:                ratelimiter.FixedWindow,
		WindowSize:          60, // segundos
		MaxRequestPerWindow: 100,
		Now:                 time.Now,
	}
	limiter := ratelimiter.NewRedisRateLimiter(redisStore, cfg)

	// 4. Servidor HTTP
	server := api.NewServer(limiter)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}

	log.Println("server stopped")
}
