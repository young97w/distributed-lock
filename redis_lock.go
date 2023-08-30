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
)

type Client struct {
	c *redis.Client
}

func (c *Client) TryLock(key string, maxCount int, ctx context.Context, timeout time.Duration) (*Lock, error) {
	value := uuid.New().String()
	ok, err := c.c.SetNX(ctx, key, value, timeout).Result()
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
		timeout:     timeout,
		maxCount:    maxCount,
		timeoutChan: make(chan struct{}, 1),
		unlockChan:  make(chan struct{}, 1),
		ctx:         ctx,
	}, nil
}

type Lock struct {
	key         string
	value       string
	c           *redis.Client
	timeout     time.Duration
	maxCount    int
	ctx         context.Context
	timeoutChan chan struct{}
	unlockChan  chan struct{}
}

func (l *Lock) Unlock() error {
	res, err := l.c.Eval(l.ctx, luaUnlock, []string{l.key}).Result()
	if err != nil {
		l.unlockChan <- struct{}{}
		return err
	}
	if res != "OK" {
		return errUnlockKeyFailed
	}
	return nil
}

func (l *Lock) Refresh() error {
	res, err := l.c.Eval(l.ctx, luaRefresh, []string{l.key}, l.timeout).Result()
	if err != nil {
		return err
	}
	if res != "OK" {
		return errRefreshLockFailed
	}
	return nil
}

func (l *Lock) AutoRefresh(maxCnt int) error {
	ticker := time.NewTicker(l.timeout)
	count := 0
	for {
		select {
		case <-ticker.C:
			err := l.Refresh()
			if errors.Is(err, context.DeadlineExceeded) {
				l.timeoutChan <- struct{}{}
			}
			return err
		case <-l.timeoutChan:
			count++
			if count > l.maxCount {
				return errRetryTooManyTimes
			}
			err := l.Refresh()
			if errors.Is(err, context.DeadlineExceeded) {
				ticker.Reset(l.timeout)
				l.timeoutChan <- struct{}{}
			}
			return err
		case <-l.unlockChan:
			return nil
		}
	}
}
