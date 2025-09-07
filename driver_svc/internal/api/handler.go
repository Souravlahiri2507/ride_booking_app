package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"github.com/souravlahiri/driver_svc/internal/domain"
	"github.com/souravlahiri/driver_svc/internal/infrastructure"
	"github.com/souravlahiri/driver_svc/internal/utils"
)

type DriverService struct {
	driverRepo *infrastructure.DriverRepo
	jobRepo    *infrastructure.JobRepo
	producer   *kafka.Writer
	consumer   *kafka.Reader
	logger     *utils.Logger
}

func NewDriverService(d *infrastructure.DriverRepo, j *infrastructure.JobRepo, prod *kafka.Writer, cons *kafka.Reader, logger *utils.Logger) *DriverService {
	// seed drivers
	_ = d.SeedIfEmpty(context.Background())
	return &DriverService{driverRepo: d, jobRepo: j, producer: prod, consumer: cons, logger: logger}
}

func RegisterDriverRoutes(r *gin.Engine, svc *DriverService) {
	r.GET("/drivers", svc.ListDrivers)
	r.GET("/jobs", svc.ListJobs)
	r.POST("/jobs/:booking_id/accept", svc.AcceptJob)
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
}

func (s *DriverService) ListDrivers(c *gin.Context) {
	list, err := s.driverRepo.List(context.Background())
	if err != nil {
		s.logger.Errorf("drivers list: %v", err)
		c.JSON(500, gin.H{"error": "db"})
		return
	}
	c.JSON(200, list)
}

func (s *DriverService) ListJobs(c *gin.Context) {
	list, err := s.jobRepo.ListOpenJobs(context.Background())
	if err != nil {
		s.logger.Errorf("jobs list: %v", err)
		c.JSON(500, gin.H{"error": "db"})
		return
	}
	c.JSON(200, list)
}

// POST /jobs/:booking_id/accept
func (s *DriverService) AcceptJob(c *gin.Context) {
	bookingID := c.Param("booking_id")
	var in struct {
		DriverID string `json:"driver_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": "bad request"})
		return
	}
	ctx := context.Background()
	ok, err := s.jobRepo.ClaimJob(ctx, bookingID, in.DriverID)
	if err != nil {
		s.logger.Errorf("claim: %v", err)
		c.JSON(500, gin.H{"error": "db"})
		return
	}
	if !ok {
		c.JSON(409, gin.H{"error": "already taken or not found"})
		return
	}
	// produce booking.accepted
	event := map[string]string{
		"booking_id": bookingID,
		"driver_id":  in.DriverID,
	}
	payload, _ := json.Marshal(event)
	_ = s.producer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(bookingID),
		Value: payload,
	})
	c.JSON(200, gin.H{"ok": true})
}

// Consumer: listens for booking.created and stores job
func (s *DriverService) StartConsumer(ctx context.Context) {
	for {
		m, err := s.consumer.ReadMessage(ctx)
		if err != nil {
			s.logger.Infof("driver consumer err: %v", err)
			return
		}
		var b domain.Job
		if err := json.Unmarshal(m.Value, &b); err != nil {
			s.logger.Warnf("bad booking event: %v", err)
			continue
		}
		// store as open job (taken=false)
		b.Taken = false
		b.CreatedAt = time.Now().UTC()
		if err := s.jobRepo.InsertJob(ctx, &b); err != nil {
			s.logger.Warnf("insert job: %v", err)
		} else {
			s.logger.Infof("job inserted: %s", b.BookingID)
		}
	}
}
