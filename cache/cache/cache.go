package cache

import (
	"context"
	"time"

	"github.com/neazzosa/common-util-go/monitor/monitor"
)

type (
	MSetData struct {
		Key   string
		Value interface{}
		Ttl   time.Duration
	}

	Cache interface {
		Ping() error

		Get(string, interface{}) error
		Set(string, interface{}, time.Duration) error

		SetNX(string, interface{}, time.Duration) (bool, error)

		HGetAll(string) (map[string]string, error)
		HGet(string, string, interface{}) error
		HSet(string, string, interface{}, time.Duration) error
		HDel(string, ...string) error

		HMGet(string, ...string) ([]interface{}, error)
		HMSet(string, map[string]interface{}, time.Duration) error

		MSet([]MSetData) error
		MGet([]string) ([]interface{}, error)

		Keys(string) ([]string, error)
		Remove(...string) error
		RemoveByPattern(string) error

		FlushDB() error
		FlushAll() error
		Close() error

		Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) Cache
	}
)
