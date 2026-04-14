package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// SpatialCache provides Redis GEO-based caching for spatial queries.
// It caches memory IDs at their locations for fast radius searches.
type SpatialCache interface {
	// AddMemory adds a memory's location to the spatial index.
	AddMemory(ctx context.Context, memoryID int64, lat, lng float64) error

	// RemoveMemory removes a memory from the spatial index.
	RemoveMemory(ctx context.Context, memoryID int64) error

	// SearchNearby returns memory IDs within radius meters of (lat, lng).
	SearchNearby(ctx context.Context, lat, lng float64, radius int) ([]int64, error)

	// Invalidate removes all cached entries in a region (used after writes).
	Invalidate(ctx context.Context, lat, lng float64, radius int) error
}

type redisSpatialCache struct {
	client *redis.Client
	key    string // Redis key for the GEO set
	ttl    time.Duration
}

func NewSpatialCache(client *redis.Client) SpatialCache {
	return &redisSpatialCache{
		client: client,
		key:    "spatial:memories",
		ttl:    5 * time.Minute,
	}
}

func (c *redisSpatialCache) AddMemory(ctx context.Context, memoryID int64, lat, lng float64) error {
	// GEOADD key longitude latitude member
	member := fmt.Sprintf("%d", memoryID)
	if err := c.client.GeoAdd(ctx, c.key, &redis.GeoLocation{
		Name:      member,
		Latitude:  lat,
		Longitude: lng,
	}).Err(); err != nil {
		return fmt.Errorf("geoadd memory: %w", err)
	}

	// Set expiration on the key
	c.client.Expire(ctx, c.key, c.ttl)
	return nil
}

func (c *redisSpatialCache) RemoveMemory(ctx context.Context, memoryID int64) error {
	member := fmt.Sprintf("%d", memoryID)
	if err := c.client.ZRem(ctx, c.key, member).Err(); err != nil {
		return fmt.Errorf("remove memory from cache: %w", err)
	}
	return nil
}

func (c *redisSpatialCache) SearchNearby(ctx context.Context, lat, lng float64, radius int) ([]int64, error) {
	// GEOSEARCH key FROMLONLAT lng lat BYRADIUS r m
	results, err := c.client.GeoSearch(ctx, c.key, &redis.GeoSearchQuery{
		Longitude:  lng,
		Latitude:   lat,
		Radius:     float64(radius),
		RadiusUnit: "m",
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("geosearch: %w", err)
	}

	ids := make([]int64, 0, len(results))
	for _, member := range results {
		id, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			log.Warn().Str("member", member).Msg("invalid member in spatial cache")
			continue
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (c *redisSpatialCache) Invalidate(ctx context.Context, lat, lng float64, radius int) error {
	// Find all members within the invalidation radius and remove them
	members, err := c.client.GeoSearch(ctx, c.key, &redis.GeoSearchQuery{
		Longitude:  lng,
		Latitude:   lat,
		Radius:     float64(radius),
		RadiusUnit: "m",
	}).Result()

	if err != nil {
		return fmt.Errorf("search for invalidation: %w", err)
	}

	if len(members) > 0 {
		if err := c.client.ZRem(ctx, c.key, members).Err(); err != nil {
			return fmt.Errorf("invalidate cache: %w", err)
		}
	}

	return nil
}

// Ensure redisSpatialCache implements SpatialCache
var _ SpatialCache = (*redisSpatialCache)(nil)

// noOpSpatialCache is a no-op cache used when Redis is unavailable.
type noOpSpatialCache struct{}

func NewNoOpSpatialCache() SpatialCache {
	return &noOpSpatialCache{}
}

func (c *noOpSpatialCache) AddMemory(_ context.Context, _ int64, _, _ float64) error {
	return nil
}

func (c *noOpSpatialCache) RemoveMemory(_ context.Context, _ int64) error {
	return nil
}

func (c *noOpSpatialCache) SearchNearby(_ context.Context, _, _ float64, _ int) ([]int64, error) {
	return nil, nil
}

func (c *noOpSpatialCache) Invalidate(_ context.Context, _, _ float64, _ int) error {
	return nil
}

var _ SpatialCache = (*noOpSpatialCache)(nil)
