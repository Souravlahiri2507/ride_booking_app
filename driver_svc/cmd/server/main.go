package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/souravlahiri/driver_svc/internal/api"
	"github.com/souravlahiri/driver_svc/internal/infrastructure"
	"github.com/souravlahiri/driver_svc/internal/utils"
)

func main() {
	logger := utils.NewLogger()
	mongoURI := getenv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getenv("MONGO_DB", "ride_app")
	kafkaBrokers := getenv("KAFKA_BROKERS", "localhost:9092")
	port := getenv("PORT", "8081")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := infrastructure.NewMongoClient(ctx, mongoURI)
	if err != nil {
		logger.Fatalf("mongo connect: %v", err)
	}

	driverRepo := infrastructure.NewDriverRepo(client.Database(dbName))
	jobRepo := infrastructure.NewJobRepo(client.Database(dbName))

	producer := infrastructure.NewKafkaProducer(kafkaBrokers, "booking.accepted")
	consumer := infrastructure.NewKafkaConsumer(kafkaBrokers, "booking.created", "driver_svc_group")

	svc := api.NewDriverService(driverRepo, jobRepo, producer, consumer, logger)
	router := gin.Default()
	api.RegisterDriverRoutes(router, svc)

	go svc.StartConsumer(ctx)

	go func() {
		if err := router.Run(":" + port); err != nil {
			logger.Fatalf("server run: %v", err)
		}
	}()
	logger.Infof("driver_svc running on :%s", port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down driver_svc...")
	cancel()
	time.Sleep(1 * time.Second)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
