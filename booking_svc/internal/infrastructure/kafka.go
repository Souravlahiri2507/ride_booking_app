package infrastructure

import (
	"github.com/segmentio/kafka-go"
)

func NewKafkaProducer(brokers, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
	}
}

func NewKafkaConsumer(brokers, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{brokers},
		GroupID:     groupID,
		Topic:       topic,
		StartOffset: kafka.LastOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
	})
}
