package distributed_lock

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"

	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClient_TryLock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	client := &Client{
		c: rdb,
	}
	l, err := client.TryLock("d-lock", 20, context.Background(), time.Second*2)
	require.NoError(t, err)
	go l.AutoRefresh(time.Second)
	require.NoError(t, err)
	time.Sleep(time.Second * 5)
	err = l.Unlock()
	//rdb.Del(context.Background(), "d-lock")
	require.NoError(t, err)
}

func TestClient_ref(t *testing.T) {
	rc := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	client := &Client{
		c: rc,
	}
	l, err := client.TryLock("d-lock", 20, context.Background(), time.Second*2)
	require.NoError(t, err)
	go l.AutoRefresh(time.Second)
	time.Sleep(time.Second * 3)
	require.NoError(t, err)
	res, _ := rc.Get(context.Background(), "d-lock").Result()
	fmt.Println(res)
	err = l.Unlock()
	//rc.Del(context.Background(), "d-lock")
	require.NoError(t, err)
}

func TestLock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	c := &Client{c: rdb}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*900)
	defer cancel()
	l, err := c.Lock("key1", 2, ctx, time.Second, time.Millisecond*200)

	require.NoError(t, err)
	res, _ := rdb.Get(context.Background(), "key1").Result()
	fmt.Println(res)
	err = l.Unlock()
	require.NoError(t, err)
}
