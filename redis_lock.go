package distributed_lock

import (
	"context"
	_ "embed"
	"errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"time"
)

var (
	errGetLockFailed     = errors.New("D-Lock: Get lock failed! ")
	errKeyNotExist       = errors.New("D-lock: Key not exists! ")
	errRefreshLockFailed = errors.New("D-lock: Refresh lock failed! ")
	errUnlockKeyFailed   = errors.New("D-lock: Unlock key failed! ")
	errRetryTooManyTimes = errors.New("D-lock: Retry too many times! ")
	//go:embed lua/refresh.lua
	luaRefresh string
	//go:embed lua/unlock.lua
	luaUnlock string
	//go:embed lua/lock.lua
	luaLock string
)

type Client struct {
	c *redis.Client
}

func (c *Client) TryLock(key string, maxCount int, ctx context.Context, duration time.Duration) (*Lock, error) {
	value := uuid.New().String()
	ok, err := c.c.SetNX(ctx, key, value, duration).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errGetLockFailed
	}
	return &Lock{
		key:         key,
		value:       value,
		c:           c.c,
		duration:    duration,
		maxCount:    maxCount,
		timeoutChan: make(chan struct{}, 1),
		unlockChan:  make(chan struct{}, 1),
		ctx:         ctx,
	}, nil
}

func (c *Client) Lock(key string, maxCount int, ctx context.Context, duration, timeout time.Duration) (*Lock, error) {
	var ticker *time.Ticker
	cnt := 0
	value := uuid.New().String()
	for cnt < maxCount {
		cnt++
		lctx, cancel := context.WithTimeout(ctx, timeout)
		res, err := c.c.Eval(lctx, luaLock, []string{key}, value, duration.Seconds()).Result()
		cancel()
		if err != nil && errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if res == "OK" {
			return &Lock{
				key:         key,
				value:       value,
				c:           c.c,
				duration:    duration,
				maxCount:    maxCount,
				ctx:         ctx,
				timeoutChan: make(chan struct{}, 1),
				unlockChan:  make(chan struct{}, 1),
			}, nil
		}

		if ticker == nil {
			ticker = time.NewTicker(timeout)
		} else {
			ticker.Reset(timeout)
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, errRetryTooManyTimes
}

type Lock struct {
	key         string
	value       string
	c           *redis.Client
	duration    time.Duration
	maxCount    int
	ctx         context.Context
	timeoutChan chan struct{}
	unlockChan  chan struct{}
}

func (l *Lock) Unlock() error {
	res, err := l.c.Eval(l.ctx, luaUnlock, []string{l.key}, l.value).Result()
	if err != nil {
		l.unlockChan <- struct{}{}
		return err
	}
	if res == 0 {
		l.unlockChan <- struct{}{}
		return errKeyNotExist
	}
	return nil
}

func (l *Lock) Refresh(ctx context.Context) error {
	res, err := l.c.Eval(ctx, luaRefresh, []string{l.key}, l.value, l.duration.Seconds()).Result()
	if err != nil {
		return err
	}
	if res == 0 {
		return err
	}
	return nil
}

func (l *Lock) AutoRefresh(duration time.Duration) error {
	ticker := time.NewTicker(duration)
	count := 0
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(l.ctx, l.duration)
			err := l.Refresh(ctx)
			cancel()
			if errors.Is(err, context.DeadlineExceeded) {
				l.timeoutChan <- struct{}{}
				continue
			}
			if err != nil {
				return err
			}
			count = 0
		case <-l.timeoutChan:
			count++
			if count > l.maxCount {
				return context.DeadlineExceeded
			}
			ctx, cancel := context.WithTimeout(l.ctx, l.duration)
			err := l.Refresh(ctx)
			cancel()
			if errors.Is(err, context.DeadlineExceeded) {
				ticker.Reset(l.duration)
				l.timeoutChan <- struct{}{}
				continue
			}
			if err != nil {
				return err
			}
		case <-l.unlockChan:
			return nil
		}
	}
}
