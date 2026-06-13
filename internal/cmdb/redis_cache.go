package cmdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/stellhub/stellar"
)

type RedisCacheOptions struct {
	Prefix               string
	ApplicationListTTL   time.Duration
	ApplicationOwnersTTL time.Duration
}

type RedisCache struct {
	client               *stellar.RedisClient
	prefix               string
	applicationListTTL   time.Duration
	applicationOwnersTTL time.Duration
}

func NewRedisCache(client *stellar.RedisClient, options RedisCacheOptions) Cache {
	if client == nil {
		return nil
	}
	prefix := strings.Trim(strings.TrimSpace(options.Prefix), ":")
	if prefix == "" {
		prefix = "cmdb"
	}
	if options.ApplicationListTTL <= 0 {
		options.ApplicationListTTL = 30 * time.Second
	}
	if options.ApplicationOwnersTTL <= 0 {
		options.ApplicationOwnersTTL = time.Minute
	}
	return &RedisCache{
		client:               client,
		prefix:               prefix,
		applicationListTTL:   options.ApplicationListTTL,
		applicationOwnersTTL: options.ApplicationOwnersTTL,
	}
}

func (c *RedisCache) GetApplicationList(ctx context.Context, query ApplicationListQuery) ([]ApplicationSummary, bool, error) {
	var items []ApplicationSummary
	ok, err := c.getJSON(ctx, c.applicationListKey(query), &items)
	return items, ok, err
}

func (c *RedisCache) SetApplicationList(ctx context.Context, query ApplicationListQuery, items []ApplicationSummary) error {
	return c.setJSON(ctx, c.applicationListKey(query), items, c.applicationListTTL)
}

func (c *RedisCache) GetApplicationOwners(ctx context.Context, appID string) ([]PersonRelation, bool, error) {
	var items []PersonRelation
	ok, err := c.getJSON(ctx, c.applicationOwnersKey(appID), &items)
	return items, ok, err
}

func (c *RedisCache) SetApplicationOwners(ctx context.Context, appID string, items []PersonRelation) error {
	return c.setJSON(ctx, c.applicationOwnersKey(appID), items, c.applicationOwnersTTL)
}

func (c *RedisCache) InvalidateApplications(ctx context.Context, identifiers ...string) error {
	if c == nil || c.client == nil {
		return nil
	}

	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, c.prefix+":app:list:v1:*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}

	ownerKeys := make([]string, 0, len(identifiers))
	for _, identifier := range identifiers {
		identifier = strings.TrimSpace(identifier)
		if identifier == "" {
			continue
		}
		ownerKeys = append(ownerKeys, c.applicationOwnersKey(identifier))
	}
	if len(ownerKeys) > 0 {
		return c.client.Del(ctx, ownerKeys...).Err()
	}
	return nil
}

func (c *RedisCache) getJSON(ctx context.Context, key string, target any) (bool, error) {
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if exists == 0 {
		return false, nil
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return false, err
	}
	return true, nil
}

func (c *RedisCache) setJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisCache) applicationListKey(query ApplicationListQuery) string {
	raw := fmt.Sprintf("env=%s|status=%s|search=%s|limit=%d|offset=%d",
		query.Environment,
		query.Status,
		query.Search,
		query.Limit,
		query.Offset,
	)
	return c.prefix + ":app:list:v1:" + cacheDigest(raw)
}

func (c *RedisCache) applicationOwnersKey(appID string) string {
	return c.prefix + ":app:" + strings.TrimSpace(appID) + ":owners:v1"
}

func cacheDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
