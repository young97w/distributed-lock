package distributed_lock

import (
	"context"
	"github.com/redis/go-redis/v9"

	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClient_TryLock(t *testing.T) {
	rc := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	client := &Client{
		c: rc,
	}
	l, err := client.TryLock("d-lock", 20, context.Background(), time.Second*2)
	require.NoError(t, err)
	go l.AutoRefresh(time.Second)
	require.NoError(t, err)
	time.Sleep(time.Second * 5)
	err = l.Unlock()
	//rc.Del(context.Background(), "d-lock")
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
	err = l.Refresh(context.Background())
	require.NoError(t, err)
	err = l.Unlock()
	//rc.Del(context.Background(), "d-lock")
	require.NoError(t, err)
}
