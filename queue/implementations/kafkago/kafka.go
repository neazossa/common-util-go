package kafkago

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/neazossa/common-util-go/logger/logger"
	"github.com/neazossa/common-util-go/monitor/monitor"
	"github.com/neazossa/common-util-go/queue/queue"
)

type (
	Queue struct {
		Logger         logger.Logger
		kafka          *kafka.Conn
		connection     queue.Connection
		isMonitor      bool
		monitor        monitor.Monitor
		ctx            context.Context
		isCaptureError bool
		requestId      string
	}
)

func NewKafkaConnection(connection queue.Connection, logger logger.Logger) (queue.Kafka, error) {
	conn, err := kafka.Dial("tcp", connection.Host)
	if err != nil {
		logger.Fatal("failed to dial leader:", err)
	}
	conn.SetWriteDeadline(time.Now().Add(time.Duration(connection.WriterTimeout) * time.Second))
	conn.SetReadDeadline(time.Now().Add(time.Duration(connection.ReaderTimeout) * time.Second))

	mq := Queue{
		Logger:     logger,
		kafka:      conn,
		connection: connection,
	}
	return &mq, err
}

func (mq *Queue) WriteMessages(ctx context.Context, topic, groupId string, msg interface{}) error {
	defer mq.doMonitor("WRITE", topic, groupId)()

	brokers := strings.Split(mq.connection.Host, ",")
	producer := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     topic,
		Topic:       groupId,
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})

	// parse data
	dataParse, err := json.Marshal(msg)
	if err != nil {
		return mq.captureError(err)
	}

	// publish message
	if err := producer.CommitMessages(context.Background(), kafka.Message{Value: []byte(dataParse)}); err != nil {
		return mq.captureError(err)
	}

	return nil
}

func (mq *Queue) ReadMessages(ctx context.Context, topic, groupId string, handler func(ctx context.Context, d kafka.Message) error, retry bool) error {
	defer mq.doMonitor("READ", topic, groupId)()

	brokers := strings.Split(mq.connection.Host, ",")
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     topic,
		Topic:       groupId,
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})

	// consume
	for {
		message, errFetch := reader.FetchMessage(ctx)

		if errFetch != nil {
			mq.Logger.Error("failed to fetch messages: ", errFetch)
		}
		mq.Logger.Info("success to fetch message", string(message.Value), message.Offset)

		err := handler(ctx, message)
		// retrying
		if err != nil && retry {
			mq.Logger.Info("retrying message...")
			if err := reader.CommitMessages(ctx, message); err != nil {
				mq.Logger.Error("failed to commit messages: ", err)
			}
		}
	}

	return nil
}

func (mq *Queue) Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) queue.Kafka {
	return &Queue{
		Logger:         mq.Logger,
		kafka:          mq.kafka,
		isMonitor:      true,
		monitor:        mntr,
		ctx:            ctx,
		requestId:      requestId,
		isCaptureError: captureError,
	}
}

func (mq *Queue) doMonitor(action, topic, groupID string) func() {
	if mq.isMonitor {
		tr := mq.startMonitor(action, topic, groupID)
		return func() {
			mq.finishMonitor(tr)
		}
	}
	return func() {}
}

func (mq *Queue) finishMonitor(transaction monitor.Transaction) {
	transaction.Finish()
}

func (mq *Queue) startMonitor(action, topic, groupID string) monitor.Transaction {
	tags := []monitor.Tag{
		{"requestId", mq.requestId},
		{"action", action},
		{"topic", topic},
		{"groupId", groupID},
	}

	return mq.monitor.NewTransactionFromContext(mq.ctx, monitor.Tick{
		Operation:       "kafka",
		TransactionName: action,
		Tags:            tags,
	})
}

func (mq *Queue) captureError(err error) error {
	if mq.isCaptureError {
		mq.monitor.Capture(err)
	}
	return err
}
