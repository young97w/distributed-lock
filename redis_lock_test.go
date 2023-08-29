package distributed_lock

import (
	"context"
	"github.com/redis/go-redis/v9"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClient_TryLock(t *testing.T) {
	rc := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	client := &Client{c: rc}
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()
	_, err := client.TryLock("d-lock", "value", context.Background(), time.Second*120)
	require.NoError(t, err)
	res, err := rc.Del(ctx, "d-lock").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), res)
}
