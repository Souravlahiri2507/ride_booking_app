package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/souravlahiri/booking_svc/internal/domain"
	"github.com/souravlahiri/booking_svc/internal/infrastructure"
	"github.com/souravlahiri/booking_svc/internal/utils"
)

type BookingService struct {
	repo     *infrastructure.BookingRepo
	producer *kafka.Writer
	consumer *kafka.Reader
	logger   *utils.Logger
}

func NewBookingService(repo *infrastructure.BookingRepo, producer *kafka.Writer, consumer *kafka.Reader, logger *utils.Logger) *BookingService {
	return &BookingService{repo: repo, producer: producer, consumer: consumer, logger: logger}
}

func RegisterBookingRoutes(r *gin.Engine, svc *BookingService) {
	r.POST("/bookings", svc.CreateBooking)
	r.GET("/bookings", svc.ListBookings)
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
}

// POST /bookings
func (s *BookingService) CreateBooking(c *gin.Context) {
	var in struct {
		PickUp  domain.Location `json:"pickuploc"`
		DropOff domain.Location `json:"dropoff"`
		Price   float64         `json:"price"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		s.logger.Warnf("bad input: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b := domain.Booking{
		BookingID:  uuid.New().String(),
		PickUp:     in.PickUp,
		DropOff:    in.DropOff,
		Price:      in.Price,
		RideStatus: "Requested",
		DriverID:   nil,
		CreatedAt:  time.Now().UTC(),
	}
	ctx := context.Background()
	if err := s.repo.Insert(ctx, &b); err != nil {
		s.logger.Errorf("insert: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// produce booking.created
	payload, _ := json.Marshal(b)
	_ = s.producer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(b.BookingID),
		Value: payload,
	})

	c.JSON(http.StatusCreated, b)
}

// GET /bookings
func (s *BookingService) ListBookings(c *gin.Context) {
	ctx := context.Background()
	list, err := s.repo.List(ctx)
	if err != nil {
		s.logger.Errorf("list: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Consumer: listen booking.accepted and update DB
func (s *BookingService) StartConsumer(ctx context.Context) {
	for {
		m, err := s.consumer.ReadMessage(ctx)
		if err != nil {
			s.logger.Infof("consumer read err: %v", err)
			return
		}
		var ev struct {
			BookingID string `json:"booking_id"`
			DriverID  string `json:"driver_id"`
		}
		if err := json.Unmarshal(m.Value, &ev); err != nil {
			s.logger.Warnf("bad event: %v", err)
			continue
		}
		if err := s.repo.AcceptBooking(ctx, ev.BookingID, ev.DriverID); err != nil {
			s.logger.Infof("accept apply: %v", err)
		} else {
			s.logger.Infof("booking %s accepted by %s", ev.BookingID, ev.DriverID)
		}
	}
}
