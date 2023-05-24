package goredis

import (
	"context"
	"encoding"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/neazossa/common-util-go/cache/cache"
	"github.com/neazossa/common-util-go/logger/logger"
	"github.com/neazossa/common-util-go/monitor/monitor"
	"github.com/pkg/errors"
)

type (
	Option struct {
		Host               string
		Port               string
		Password           string
		DB                 int
		PoolSize           int
		MinIdleCons        int
		MaxDelPerOperation int64
		DialTimeout        time.Duration
		PoolTimeout        time.Duration
		ReadTimeout        time.Duration
		WriteTimeout       time.Duration
		MaxConnAge         time.Duration
	}

	Redis struct {
		client             *redis.Client
		logger             logger.Logger
		maxDelPerOperation int64
		isMonitor          bool
		monitor            monitor.Monitor
		context            context.Context
		isCaptureError     bool
		requestId          string
	}

	mSetChan struct {
		key string
		err error
	}
)

func NewRedisConnection(opt Option, logger logger.Logger) (cache.Cache, error) {

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", opt.Host, opt.Port),
		Password:     opt.Password,
		DB:           opt.DB,
		DialTimeout:  opt.DialTimeout,
		ReadTimeout:  opt.ReadTimeout,
		WriteTimeout: opt.WriteTimeout,
		PoolSize:     opt.PoolSize,
		MinIdleConns: opt.MinIdleCons,
		MaxConnAge:   opt.MaxConnAge,
		PoolTimeout:  opt.PoolTimeout,
	})

	_, err := client.Ping().Result()
	if err != nil {
		logger.Errorf("failed ping new redis connection [%s:%s] : %s", opt.Host, opt.Port, err.Error())
		return nil, err
	}

	return &Redis{
		client:             client,
		logger:             logger,
		maxDelPerOperation: opt.MaxDelPerOperation,
	}, nil
}

func (r *Redis) Ping() error {
	_, err := r.client.Ping().Result()
	return err
}

func (r *Redis) Get(key string, result interface{}) error {
	defer r.doMonitor("Get", key)()
	if err := canUnmarshal(key, result); err != nil {
		return r.captureError(err)
	}
	value, err := r.client.Get(key).Result()

	if err == redis.Nil {
		return r.captureError(errors.Wrapf(err, "key %s does not exits", key))
	}

	if err != nil {
		return r.captureError(err)
	}
	return r.captureError(handleUnmarshal(value, result))
}

func (r *Redis) Set(key string, value interface{}, duration time.Duration) error {
	defer r.doMonitor("Set", key)()
	return r.captureError(r.client.Set(key, value, duration).Err())
}

func (r *Redis) SetNX(key string, value interface{}, duration time.Duration) (bool, error) {
	defer r.doMonitor("SetNX", key)()
	result, err := r.client.SetNX(key, value, duration).Result()
	return result, r.captureError(err)
}

func (r *Redis) HGetAll(key string) (map[string]string, error) {
	defer r.doMonitor("HGetAll", key)()
	result, err := r.client.HGetAll(key).Result()
	return result, r.captureError(err)
}

func (r *Redis) HGet(key string, field string, result interface{}) error {
	defer r.doMonitor("HGet", key)()

	if err := canUnmarshal(key, result); err != nil {
		return r.captureError(err)
	}

	value, err := r.client.HGet(key, field).Result()
	if err == redis.Nil {
		return r.captureError(errors.Wrapf(err, "field %s in key %s does not exits", field, key))
	}

	if err != nil {
		return r.captureError(err)
	}

	return r.captureError(handleUnmarshal(value, result))
}

func (r *Redis) HSet(key string, field string, value interface{}, duration time.Duration) error {
	defer r.doMonitor("HSet", key)()
	if err := r.client.HSet(key, field, value).Err(); err != nil {
		return r.captureError(err)
	}

	return r.captureError(r.client.Expire(key, duration).Err())
}

func (r *Redis) HDel(key string, fields ...string) error {
	defer r.doMonitor("HDel", key)()
	return r.captureError(r.client.HDel(key, fields...).Err())
}

func (r *Redis) HMGet(key string, fields ...string) ([]interface{}, error) {
	defer r.doMonitor("HMGet", key)()
	result, err := r.client.HMGet(key, fields...).Result()

	if err == redis.Nil {
		return nil, r.captureError(errors.Wrapf(err, "key %s does not exits", key))
	}

	if err != nil {
		return nil, r.captureError(err)
	}

	return result, nil
}

func (r *Redis) HMSet(key string, value map[string]interface{}, duration time.Duration) error {
	defer r.doMonitor("HMSet", key)()
	_, err := r.client.HMSet(key, value).Result()
	if err != nil {
		return r.captureError(err)
	}

	_, err = r.client.Expire(key, duration).Result()
	return r.captureError(err)
}

func (r *Redis) MSet(data []cache.MSetData) error {

	var (
		keys          = make([]string, 0)
		pairs         = make([]interface{}, 0)
		setDurationCh = make(chan mSetChan, len(data))
		failedKeys    = make([]string, 0)
	)

	for _, datum := range data {
		keys = append(keys, datum.Key)
		pairs = append(pairs, datum.Key, datum.Value)
	}
	defer r.doMonitor("MSet", keys...)()

	if err := r.client.MSet(pairs).Err(); err != nil {
		return r.captureError(errors.WithStack(err))
	}

	for _, datum := range data {
		go func(key string, ttl time.Duration, data chan mSetChan) {
			if _, err := r.client.Expire(key, ttl).Result(); err != nil {
				r.client.Del(key)
				r.logger.Errorf("failed when set expired for key: %s", key)
				data <- mSetChan{
					key: key,
					err: err,
				}
			}
			data <- mSetChan{
				key: key,
				err: nil,
			}
		}(datum.Key, datum.Ttl, setDurationCh)
	}

	for ch := range setDurationCh {
		if ch.err != nil {
			failedKeys = append(failedKeys, ch.key)
		}
	}

	if len(failedKeys) > 0 {
		return r.captureError(errors.New("failed when insert keys " + strings.Join(failedKeys, ",")))
	}

	return nil
}

func (r *Redis) MGet(keys []string) ([]interface{}, error) {
	defer r.doMonitor("MGet", keys...)()
	val, err := r.client.MGet(keys...).Result()
	if err == redis.Nil {
		return nil, r.captureError(errors.Wrapf(err, "key %s does not exits", keys))
	}

	if err != nil {
		return nil, r.captureError(errors.Wrapf(err, "failed to get key %s!", keys))
	}

	return val, nil
}

func (r *Redis) Keys(pattern string) ([]string, error) {
	defer r.doMonitor("Keys", pattern)()
	result, err := r.client.Keys(pattern).Result()
	return result, r.captureError(err)
}

func (r *Redis) Remove(keys ...string) error {
	defer r.doMonitor("Remove", keys...)()
	return r.captureError(r.client.Del(keys...).Err())
}

func (r *Redis) RemoveByPattern(pattern string) error {
	var (
		wg     = &sync.WaitGroup{}
		failed = make([]string, 0)
	)
	defer r.doMonitor("RemoveByPattern", pattern)()
	for {
		keys, _, err := r.client.Scan(0, pattern, r.maxDelPerOperation).Result()
		if err != nil {
			return r.captureError(errors.Wrapf(err, "failed to scan redis pattern %s!", pattern))
		}

		if len(keys) == 0 {
			break
		}

		wg.Add(1)
		go func() {
			if err := r.client.Del(keys...).Err(); err != nil {
				failed = append(failed, keys...)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if len(failed) > 0 {
		return r.captureError(errors.Wrapf(
			errors.New(fmt.Sprintf("cannot delete keys[%s]", strings.Join(failed, ","))),
			"failed to scan redis pattern %s!", pattern))
	}

	return nil
}

func (r *Redis) FlushDB() error {
	return r.client.FlushDB().Err()
}

func (r *Redis) FlushAll() error {
	return r.client.FlushAll().Err()
}

func (r *Redis) Close() error {
	if err := r.client.Close(); err != nil {
		return errors.Wrap(err, "failed to close redis connection")
	}

	return nil
}

func (r *Redis) Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) cache.Cache {
	return &Redis{
		client:             r.client,
		logger:             r.logger,
		maxDelPerOperation: r.maxDelPerOperation,
		isMonitor:          true,
		monitor:            mntr,
		context:            ctx,
		isCaptureError:     captureError,
		requestId:          requestId,
	}
}

func canUnmarshal(key string, result interface{}) error {
	if _, ok := result.(encoding.BinaryUnmarshaler); !ok {
		return errors.New(fmt.Sprintf("can't unmarshal result value for key %s", key))
	}
	return nil
}

func handleUnmarshal(value string, result interface{}) error {
	return result.(encoding.BinaryUnmarshaler).UnmarshalBinary([]byte(value))
}

func (r *Redis) startMonitor(action string, keys ...string) monitor.Transaction {
	tags := []monitor.Tag{
		{"requestId", r.requestId},
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

	return r.monitor.NewTransactionFromContext(r.context, monitor.Tick{
		Operation:       "redis",
		TransactionName: action,
		Tags:            tags,
	})
}

func (r *Redis) finishMonitor(transaction monitor.Transaction) {
	transaction.Finish()
}

func (r *Redis) doMonitor(action string, keys ...string) func() {
	if r.isMonitor {
		tr := r.startMonitor(action, keys...)
		return func() {
			r.finishMonitor(tr)
		}
	}
	return func() {}
}

func (r *Redis) captureError(err error) error {
	if r.isCaptureError {
		r.monitor.Capture(err)
	}
	return err
}
