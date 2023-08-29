package distributed_lock

import (
	"context"
	_ "embed"
	"errors"
	"github.com/redis/go-redis/v9"
	"time"
)

var (
	errGetLockFailed     = errors.New("D-Lock: Get lock failed! ")
	errKeyNotExist       = errors.New("D-lock: Key not exists! ")
	errRefreshLockFailed = errors.New("refresh lock failed! ")
	//go:embed lua/refresh.lua
	luaRefresh string
)

type Client struct {
	c *redis.Client
}

func (c *Client) TryLock(key string, value string, ctx context.Context, timeout time.Duration) (*Lock, error) {
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
		duration:    timeout,
		timeoutChan: make(chan struct{}, 1),
		unlockChan:  make(chan struct{}, 1),
		ctx:         ctx,
	}, nil
}

type Lock struct {
	key         string
	value       string
	c           *redis.Client
	duration    time.Duration
	ctx         context.Context
	timeoutChan chan struct{}
	unlockChan  chan struct{}
}

func (l *Lock) Unlock() (bool, error) {
	_, err := l.c.Del(l.ctx, l.key).Result()
	if err != nil {
		l.unlockChan <- struct{}{}
		return false, err
	}
	return true, nil
}

func (l *Lock) Refresh() (bool, error) {
	res, err := l.c.Eval(l.ctx, luaRefresh, []string{l.key}, l.duration).Result()
	if err != nil {
		return false, err
	}
	if res != "OK" {
		return false, errRefreshLockFailed
	}
	return true, nil
}

func (l *Lock) AutoRefresh(maxCnt int) (bool, error) {

}
