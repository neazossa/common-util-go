package queue

import (
	"context"

	"github.com/neazzosa/common-util-go/monitor/monitor"
	"github.com/segmentio/kafka-go"
)

type (
	Kafka interface {
		WriteMessages(ctx context.Context, topic, groupId string, msg interface{}) error
		ReadMessages(ctx context.Context, topic, groupId string, handler func(ctx context.Context, d kafka.Message) error, retry bool) error

		Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) Kafka
	}

	Connection struct {
		Host          string
		ReaderTimeout int // in seconds
		WriterTimeout int // in seconds
	}
)
