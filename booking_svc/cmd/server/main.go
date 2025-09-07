package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/souravlahiri/booking_svc/internal/api"
	"github.com/souravlahiri/booking_svc/internal/infrastructure"
	"github.com/souravlahiri/booking_svc/internal/utils"
)

func main() {
	logger := utils.NewLogger()
	// loading envs
	mongoURI := getenv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getenv("MONGO_DB", "ride_app")
	kafkaBrokers := getenv("KAFKA_BROKERS", "localhost:9092")
	port := getenv("PORT", "8080")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Mongo connect
	client, err := infrastructure.NewMongoClient(ctx, mongoURI)
	if err != nil {
		logger.Fatalf("mongo connect: %v", err)
	}
	repo := infrastructure.NewBookingRepo(client.Database(dbName))

	// Kafka producer + consumer
	producer := infrastructure.NewKafkaProducer(kafkaBrokers, "booking.created")
	consumer := infrastructure.NewKafkaConsumer(kafkaBrokers, "booking.accepted", "booking_svc_group")

	// service and handlers
	srv := api.NewBookingService(repo, producer, consumer, logger)
	router := gin.Default()
	api.RegisterBookingRoutes(router, srv)

	// listening for booking.accepted
	go srv.StartConsumer(ctx)

	// graceful shutdown
	go func() {
		if err := router.Run(":" + port); err != nil {
			logger.Fatalf("server run: %v", err)
		}
	}()
	logger.Infof("booking_svc running on :%s", port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down booking_svc...")

	cancel()
	// providing some time to cleanup
	time.Sleep(1 * time.Second)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
