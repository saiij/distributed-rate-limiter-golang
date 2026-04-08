package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"saiij.distributed.rate.limiter/internal/ratelimiter"
)

var _ ratelimiter.Store = (*RedisStore)(nil)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

var fixedWindowScript = redis.NewScript(`
    local count = redis.call("INCR", KEYS[1])
    if count == 1 then
        redis.call("EXPIRE", KEYS[1], ARGV[2])
    end
    return count
`)

var slidingWindowScript = redis.NewScript(`
    local data = redis.call("HGETALL", KEYS[1])
    if #data == 0 then
        redis.call("HSET", KEYS[1],
            "currentCounter", 1,
            "prevCounter",    0,
            "window",         ARGV[1]
        )
        redis.call("EXPIRE", KEYS[1], ARGV[2] * 2)
        return {1, 0}
    end
    local hash = {}
    for i = 1, #data, 2 do
        hash[data[i]] = tonumber(data[i+1])
    end
    local currentCounter = hash["currentCounter"]
    local prevCounter    = hash["prevCounter"]
    local window         = hash["window"]
    local currentWindow  = tonumber(ARGV[1])
    if window == currentWindow then
        currentCounter = currentCounter + 1
        redis.call("HSET", KEYS[1], "currentCounter", currentCounter)
    else
        local windowDiff = currentWindow - window
        if windowDiff == 1 then
            prevCounter = currentCounter
        else
            prevCounter = 0
        end
        currentCounter = 1
        redis.call("HSET", KEYS[1],
            "currentCounter", currentCounter,
            "prevCounter",    prevCounter,
            "window",         currentWindow
        )
        redis.call("EXPIRE", KEYS[1], ARGV[2] * 2)
    end
    return {currentCounter, prevCounter}
`)

// KEYS[1] = key del cliente
// ARGV[1] = maxTokenCapacity
// ARGV[2] = rechargeRate
// ARGV[3] = tiempo actual en unix (segundos)
// ARGV[4] = TTL (maxCapacity / rechargeRate * 2)
var tokenBucketScript = redis.NewScript(`
    local data = redis.call("HGETALL", KEYS[1])

    local currentTokens
    local lastRequest

    if #data == 0 then
        -- primer request: bucket lleno
        currentTokens = tonumber(ARGV[1])
        lastRequest   = tonumber(ARGV[3])
    else
        local hash = {}
        for i = 1, #data, 2 do
            hash[data[i]] = tonumber(data[i+1])
        end
        currentTokens = hash["totalTokens"]
        lastRequest   = hash["lastRequest"]

        -- recargar tokens según tiempo transcurrido
        local timePassed = tonumber(ARGV[3]) - lastRequest
        local newTokens  = timePassed * tonumber(ARGV[2])
        currentTokens = math.min(tonumber(ARGV[1]), currentTokens + newTokens)
        lastRequest   = tonumber(ARGV[3])
    end

    local allowed = 0
    if currentTokens >= 1 then
        currentTokens = currentTokens - 1
        allowed = 1
    end

    redis.call("HSET", KEYS[1],
        "totalTokens", currentTokens,
        "lastRequest", lastRequest
    )
    redis.call("EXPIRE", KEYS[1], ARGV[4])

    -- retorna {allowed, currentTokens}
    return {allowed, currentTokens}
`)

func (r *RedisStore) FixedWindow(ctx context.Context, key string, windowSize int64, maxRequestPerWindow int) (*ratelimiter.AllowResponse, error) {
	response := ratelimiter.NewAllowResponse()
	currentTime := time.Now()
	currentWindow := currentTime.Unix() / windowSize
	redisKey := fmt.Sprintf("fixed:%s:%d", key, currentWindow)

	result, err := fixedWindowScript.Run(ctx, r.client, []string{redisKey}, maxRequestPerWindow, windowSize).Int()
	if err != nil {
		return nil, err
	}

	response.CanAccess = result <= maxRequestPerWindow
	response.RequestRemain = max(0, maxRequestPerWindow-result)
	response.ResetRequestAt = currentTime.Add(time.Duration(windowSize) * time.Second)
	response.RetryIn = response.ResetRequestAt.Sub(currentTime)
	return response, nil
}

func (r *RedisStore) SlidingWindow(ctx context.Context, key string, windowSize int64, maxRequestPerWindow int) (*ratelimiter.AllowResponse, error) {
	response := ratelimiter.NewAllowResponse()
	currentTime := time.Now()
	currentWindow := currentTime.Unix() / windowSize
	redisKey := fmt.Sprintf("sliding:%s", key)

	results, err := slidingWindowScript.Run(ctx, r.client, []string{redisKey}, currentWindow, windowSize).Int64Slice()
	if err != nil {
		return nil, err
	}

	currentCounter := results[0]
	prevCounter := results[1]

	secElapsed := currentTime.Unix() - (currentWindow * windowSize)
	perElapsed := float64(secElapsed) / float64(windowSize)
	counter := int(float64(prevCounter)*(1-perElapsed) + float64(currentCounter))

	response.CanAccess = counter <= maxRequestPerWindow
	response.RequestRemain = max(0, maxRequestPerWindow-counter)
	response.ResetRequestAt = currentTime.Add(time.Duration(windowSize) * time.Second)
	response.RetryIn = response.ResetRequestAt.Sub(currentTime)
	return response, nil
}

func (r *RedisStore) TokenBucket(ctx context.Context, key string, maxTokenCapacity int, rechargeRate int) (*ratelimiter.AllowResponse, error) {
	response := ratelimiter.NewAllowResponse()
	currentTime := time.Now()
	redisKey := fmt.Sprintf("token:%s", key)

	ttl := (maxTokenCapacity / rechargeRate) * 2

	results, err := tokenBucketScript.Run(
		ctx, r.client, []string{redisKey},
		maxTokenCapacity,
		rechargeRate,
		currentTime.Unix(),
		ttl,
	).Int64Slice()
	if err != nil {
		return nil, err
	}

	allowed := results[0]
	currentTokens := int(results[1])

	response.CanAccess = allowed == 1
	response.RequestRemain = min(maxTokenCapacity, currentTokens)

	if allowed == 1 {
		response.ResetRequestAt = currentTime
		response.RetryIn = 0
	} else {
		secsUntilToken := time.Duration(float64(time.Second) / float64(rechargeRate))
		response.RetryIn = secsUntilToken
		response.ResetRequestAt = currentTime.Add(secsUntilToken)
	}

	return response, nil
}
