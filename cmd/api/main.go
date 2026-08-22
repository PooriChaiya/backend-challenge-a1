// Composition root: reads env, builds adapters, wires them into the service,
// serves HTTP, runs the 10s user-count logger, shuts down on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PooriChaiya/backend-challenge-a1/internal/adapter/bcryptpw"
	"github.com/PooriChaiya/backend-challenge-a1/internal/adapter/httpapi"
	"github.com/PooriChaiya/backend-challenge-a1/internal/adapter/jwtauth"
	"github.com/PooriChaiya/backend-challenge-a1/internal/adapter/mongorepo"
	"github.com/PooriChaiya/backend-challenge-a1/internal/service"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	var (
		mongoURI  = envOr("MONGO_URI", "mongodb://localhost:27017")
		mongoDB   = envOr("MONGO_DB", "usersdb")
		jwtSecret = envOr("JWT_SECRET", "devsecret")
		port      = envOr("PORT", "8080")
	)

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Mongo client (10s connect budget).
	connectCtx, connectCancel := context.WithTimeout(rootCtx, 10*time.Second)
	defer connectCancel()
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(connectCtx, nil); err != nil {
		log.Fatalf("mongo ping: %v", err)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	repo, err := mongorepo.New(rootCtx, client.Database(mongoDB))
	if err != nil {
		log.Fatalf("repo init: %v", err)
	}

	svc := service.New(repo, bcryptpw.New(0), jwtauth.New(jwtSecret, 24*time.Hour))
	tokens := jwtauth.New(jwtSecret, 24*time.Hour) // ponytail: same secret, used as verifier too

	handler := httpapi.NewRouter(httpapi.NewHandlers(svc), tokens)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Background: log user count every 10s.
	go userCountLogger(rootCtx, svc)

	// Serve.
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-rootCtx.Done():
		log.Println("shutdown signal received")
	case err := <-serverErr:
		log.Printf("server error: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func userCountLogger(ctx context.Context, svc *service.UserService) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c, err := svc.Count(ctx)
			if err != nil {
				log.Printf("user-count: error=%v", err)
				continue
			}
			log.Printf("user-count: users=%d", c)
		}
	}
}
