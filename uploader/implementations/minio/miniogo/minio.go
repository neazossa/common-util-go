package miniogo

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/neazossa/common-util-go/logger/logger"
	"github.com/neazossa/common-util-go/monitor/monitor"
	"github.com/neazossa/common-util-go/uploader/uploader"
	"github.com/pkg/errors"
)

type (
	Minio struct {
		client         *minio.Client
		option         Option
		logger         logger.Logger
		getOpt         minio.GetObjectOptions
		putOpt         minio.PutObjectOptions
		rmvOpt         minio.RemoveObjectOptions
		isMonitor      bool
		monitor        monitor.Monitor
		context        context.Context
		isCaptureError bool
		requestId      string
	}

	Option struct {
		Host       string
		Port       string
		BucketName string
		AccessKey  string
		SecretKey  string
	}
)

func (m *Minio) SetPutOption(opt interface{}) uploader.Uploader {
	options, ok := opt.(minio.PutObjectOptions)
	if !ok {
		m.logger.Fatalf("Minio Uploader Utils: failed to cast Put Option")
		return m
	}

	return &Minio{
		client:         m.client,
		option:         m.option,
		logger:         m.logger,
		getOpt:         m.getOpt,
		putOpt:         options,
		rmvOpt:         m.rmvOpt,
		monitor:        m.monitor,
		context:        m.context,
		isCaptureError: m.isCaptureError,
		requestId:      m.requestId,
	}
}

func (m *Minio) Put(file []byte, fileName string) error {
	// byte slice to bytes.Reader, which implements the io.Reader interface
	reader := bytes.NewReader(file)

	// check if bucket is exist
	err := m.bucketCheck()
	if err != nil {
		return m.captureError(err)
	}

	defer m.doMonitor("PutObject", fileName)()
	_, err = m.client.PutObject(context.Background(), m.option.BucketName, fileName, reader, reader.Size(), m.putOpt)
	if err != nil {
		return m.captureError(errors.Wrap(err, "failed send file to bucket"))
	}
	return nil
}

func (m *Minio) bucketCheck() error {
	defer m.doMonitor("BucketExists")()
	found, err := m.client.BucketExists(context.Background(), m.option.BucketName)
	if err != nil {
		return m.captureError(errors.Wrap(err, "error when check bucket to minio"))
	}
	if !found {
		defer m.doMonitor("MakeBucket")()
		err = m.client.MakeBucket(context.Background(), m.option.BucketName, minio.MakeBucketOptions{Region: "us-east-1", ObjectLocking: true})
		if err != nil {
			return m.captureError(errors.Wrap(err, "error when make bucket"))
		}
	}
	return nil
}

func (m *Minio) SetGetOption(opt interface{}) uploader.Uploader {
	options, ok := opt.(minio.GetObjectOptions)
	if !ok {
		m.logger.Fatalf("Minio Uploader Utils: failed to cast Get Option")
		return m
	}

	return &Minio{
		client:         m.client,
		option:         m.option,
		logger:         m.logger,
		getOpt:         options,
		putOpt:         m.putOpt,
		rmvOpt:         m.rmvOpt,
		isMonitor:      true,
		monitor:        m.monitor,
		context:        m.context,
		isCaptureError: m.isCaptureError,
		requestId:      m.requestId,
	}
}

func (m *Minio) Get(fileName string) ([]byte, error) {
	var (
		buf bytes.Buffer
	)

	// check if bucket is exist
	err := m.bucketCheck()
	if err != nil {
		return []byte{}, err
	}

	defer m.doMonitor("GetObject", fileName)()
	object, err := m.client.GetObject(context.Background(), m.option.BucketName, fileName, m.getOpt)
	if err != nil {
		return []byte{}, m.captureError(errors.Wrap(err, "failed get file from bucket"))
	}
	_, err = io.Copy(&buf, object)
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

func (m *Minio) SetRemoveOption(opt interface{}) uploader.Uploader {
	options, ok := opt.(minio.RemoveObjectOptions)
	if !ok {
		m.logger.Fatalf("Minio Uploader Utils: failed to cast Get Option")
		return m
	}

	return &Minio{
		client:         m.client,
		option:         m.option,
		logger:         m.logger,
		getOpt:         m.getOpt,
		putOpt:         m.putOpt,
		rmvOpt:         options,
		isMonitor:      true,
		monitor:        m.monitor,
		context:        m.context,
		isCaptureError: m.isCaptureError,
		requestId:      m.requestId,
	}
}

func (m *Minio) Remove(fileName string) error {
	// check if bucket is exist
	err := m.bucketCheck()
	if err != nil {
		return m.captureError(err)
	}

	defer m.doMonitor("RemoveObject", fileName)()
	err = m.client.RemoveObject(context.Background(), m.option.BucketName, fileName, m.rmvOpt)
	if err != nil {
		return m.captureError(err)
	}

	return nil
}

func NewMinioUploader(ctx context.Context, logger logger.Logger, option Option) (uploader.Uploader, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(option.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(option.AccessKey, option.SecretKey, ""),
		Secure: false, // sometimes can be changed
	})
	if err != nil {
		logger.Fatalf("failed to start connection to minio : %v", err)
		return nil, err
	}

	return &Minio{
		client: minioClient,
		option: option,
		logger: logger,
	}, nil
}

func (m *Minio) Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) uploader.Uploader {
	return &Minio{
		client:         m.client,
		option:         m.option,
		logger:         m.logger,
		getOpt:         m.getOpt,
		putOpt:         m.putOpt,
		rmvOpt:         m.rmvOpt,
		isMonitor:      true,
		monitor:        mntr,
		context:        ctx,
		isCaptureError: captureError,
		requestId:      requestId,
	}
}

func (m *Minio) startMonitor(action string, keys ...string) monitor.Transaction {
	tags := []monitor.Tag{
		{"requestId", m.requestId},
		{"action", action},
	}

	if keys != nil {
		if len(keys) == 1 {
			tags = append(tags, monitor.Tag{Key: "key", Value: keys[0]})
		} else {
			for i, key := range keys {
				tags = append(tags, monitor.Tag{Key: fmt.Sprintf("key[%d]", i), Value: key})
			}
		}
	}

	return m.monitor.NewTransactionFromContext(m.context, monitor.Tick{
		Operation:       "redis",
		TransactionName: action,
		Tags:            tags,
	})
}

func (m *Minio) finishMonitor(transaction monitor.Transaction) {
	transaction.Finish()
}

func (m *Minio) doMonitor(action string, keys ...string) func() {
	if m.isMonitor {
		tr := m.startMonitor(action, keys...)
		return func() {
			m.finishMonitor(tr)
		}
	}
	return func() {}
}

func (m *Minio) captureError(err error) error {
	if m.isCaptureError {
		m.monitor.Capture(err)
	}
	return m.captureError(err)
}
