package utils

import (
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// ReadCachePrefix keys every cached read — the lookups, the master lists, the
// RBAC lists. One prefix for all of them so a writer that changes master data
// clears the lot with a single Forget instead of naming keys it might get
// wrong. It lives here rather than in a domain package because the packages
// that invalidate it (app/master, app/rbac, app/user) and the package that
// fills it (app/lookup) already import each other in one direction only.
const ReadCachePrefix = "read:"

// How long a cached read may outlive the row behind it. These are a backstop,
// not the invalidation mechanism: every write in this app drops the cache
// immediately, so a TTL only bounds a row changed outside this process.
const (
	// regions, areas and cash_and_banks are this app's own tables — nothing
	// else can change them behind our back.
	OwnTableCacheTTL = time.Hour

	// users, roles and permissions may also be written by something other than
	// this process. Those writes cannot call Forget, so this bounds how long a
	// response may disagree with the database.
	SharedTableCacheTTL = 5 * time.Minute
)

type cacheValue struct {
	value     any
	expiresAt time.Time
}

var (
	cacheMu    sync.RWMutex
	cacheStore = map[string]cacheValue{}
)

// Remember serves a value from an in-process map, calling load only on a miss
// or once ttl has passed. It is for master data — the lookup dropdowns — whose
// rows change a few times a month while every page load asks for them again.
//
// Deliberately not Redis: the app runs as a single container (docker-compose.yml,
// the same assumption the in-process notification and activitylog hubs already
// make), so a network round trip plus a decode would cost more than the small
// SELECT it replaces. If this ever runs behind multiple replicas those hubs need
// Redis pub/sub first, and this can follow through these same two functions.
//
// The cached value is shared, not copied — callers must treat what they get
// back as read only.
//
// ponytail: concurrent misses each call load, no singleflight — load here is one
// small SELECT, so a duplicate costs less than coordinating. Upgrade to
// golang.org/x/sync/singleflight if a cached load ever stops being cheap:
// a query that scans, or one that reaches another service, turns every
// concurrent miss into a stampede against it.
func Remember[T any](key string, ttl time.Duration, load func() (T, error)) (T, error) {
	cacheMu.RLock()
	entry, found := cacheStore[key]
	cacheMu.RUnlock()

	if found && time.Now().Before(entry.expiresAt) {
		if value, ok := entry.value.(T); ok {
			return value, nil
		}
	}

	value, err := load()
	if err != nil {
		return value, err
	}

	cacheMu.Lock()
	cacheStore[key] = cacheValue{value: value, expiresAt: time.Now().Add(ttl)}
	cacheMu.Unlock()

	return value, nil
}

// Forget drops every entry whose key starts with prefix. Writers pass the
// prefix rather than an exact key: blowing away a sibling lookup costs one
// cheap SELECT, while forgetting to name the right key ships a stale dropdown.
func Forget(prefix string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	for key := range cacheStore {
		if strings.HasPrefix(key, prefix) {
			delete(cacheStore, key)
		}
	}
}

// CacheEntry is one live key, for the maintenance screen. Count is how many
// items the cached value holds, so the screen can say "27 options" rather than
// only naming the key.
type CacheEntry struct {
	Key              string `json:"key"`
	Count            int    `json:"count"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

// CacheEntries lists what is cached right now, sorted by key. Expired entries
// are left out: they are already dead to Remember, so showing them would invite
// clearing something that is not there.
func CacheEntries() []CacheEntry {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	now := time.Now()
	entries := make([]CacheEntry, 0, len(cacheStore))
	for key, entry := range cacheStore {
		if !now.Before(entry.expiresAt) {
			continue
		}
		entries = append(entries, CacheEntry{
			Key:              key,
			Count:            countOf(entry.value),
			ExpiresInSeconds: int(entry.expiresAt.Sub(now).Seconds()),
		})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })

	return entries
}

// countOf sizes a cached value without knowing its type. Everything cached today
// is a slice; anything else counts as the one value it is.
func countOf(value any) int {
	switch reflect.ValueOf(value).Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return reflect.ValueOf(value).Len()
	default:
		return 1
	}
}
