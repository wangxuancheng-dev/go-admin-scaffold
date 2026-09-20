# 缓存系统

## 概述

系统支持多种缓存驱动，包括 **Redis** 和 **文件缓存**。

## 配置

在 `configs/config.yaml` 中配置缓存：

### Redis 缓存配置

```yaml
cache:
  driver: "redis"  # 使用Redis缓存
  prefix: "goadmin:"
  options:
    ttl: 3600  # 默认过期时间(秒)
    host: "localhost"
    port: 6379
    password: ""
    db: 0
```

### 文件缓存配置

```yaml
cache:
  driver: "file"  # 使用文件缓存
  prefix: "goadmin:"
  options:
    ttl: 3600  # 默认过期时间(秒)
    file_path: "storage/cache"  # 缓存文件存储目录
```

## 缓存驱动对比

| 特性 | 文件缓存 | Redis缓存 |
|------|----------|-----------|
| 性能 | 较快 | 非常快 |
| 持久化 | 自动持久化到磁盘 | 可配置持久化 |
| 内存使用 | 低 | 中等 |
| 并发支持 | 支持(文件锁) | 优秀 |
| 分布式 | 不支持 | 支持 |
| 依赖 | 无外部依赖 | 需要Redis服务 |
| 适用场景 | 单机部署、小规模应用 | 高并发、分布式应用 |

## 基本使用

### 1. 获取缓存实例

```go
// internal/services/user_service.go
type UserService struct {
    cache  cache.Store
    repo   *repositories.UserRepository
}

func NewUserService(cacheStore cache.Store, repo *repositories.UserRepository) *UserService {
    return &UserService{
        cache: cacheStore,
        repo:  repo,
    }
}
```

### 2. 基本操作

```go
ctx := context.Background()

// 设置缓存
err := cache.Set(ctx, "key", value, time.Hour)

// 获取缓存
value, err := cache.Get(ctx, "key")

// 删除缓存
err := cache.Delete(ctx, "key")

// 检查是否存在
exists := cache.Has(ctx, "key")

// 递增/递减
err = cache.Increment(ctx, "counter")
err = cache.Decrement(ctx, "counter")

// 获取或设置 - 如果缓存不存在就执行getter函数
value, err := cache.Remember(ctx, "key", time.Hour, func() (interface{}, error) {
    // 如果缓存不存在，这个函数会被调用
    return calculateExpensiveValue(), nil
})

// 清空所有缓存
err = cache.Clear(ctx)
```

## 缓存标签

支持使用标签组织和管理缓存：

```go
// 使用标签
tags := cache.Tags([]string{"users", "permissions"})

// 设置带标签的缓存
err := tags.Set("user:1", userData, time.Hour)

// 获取带标签的缓存
value, err := tags.Get("user:1")

// 清除特定标签的所有缓存
err := cache.Tags([]string{"users"}).Flush()
```

## 缓存驱动

### 1. Redis 驱动

```go
// pkg/cache/redis_driver.go
type RedisCache struct {
    client *redis.Client
    prefix string
}

func (c *RedisCache) Get(key string) (interface{}, error) {
    value, err := c.client.Get(ctx, c.prefix+key).Result()
    if err == redis.Nil {
        return nil, cache.ErrCacheMiss
    }
    return value, err
}

func (c *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
    return c.client.Set(ctx, c.prefix+key, value, ttl).Err()
}
```

### 2. 内存缓存

```go
// pkg/cache/memory_driver.go
type MemoryCache struct {
    store  map[string]*cacheItem
    mutex  sync.RWMutex
    prefix string
}

type cacheItem struct {
    value      interface{}
    expiration time.Time
}

func (c *MemoryCache) Get(key string) (interface{}, error) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()

    item, exists := c.store[c.prefix+key]
    if !exists || item.isExpired() {
        return nil, cache.ErrCacheMiss
    }
    return item.value, nil
}
```

### 3. 文件缓存

```go
// pkg/cache/file_driver.go
type FileCache struct {
    path      string
    prefix    string
    extension string
}

func (c *FileCache) Get(key string) (interface{}, error) {
    path := c.getPath(key)
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, cache.ErrCacheMiss
    }
    
    item := &cacheItem{}
    if err := json.Unmarshal(data, item); err != nil {
        return nil, err
    }
    
    if item.isExpired() {
        os.Remove(path)
        return nil, cache.ErrCacheMiss
    }
    
    return item.value, nil
}
```

## 缓存策略

### 1. 缓存预热

```go
func (s *UserService) WarmUpCache() error {
    users, err := s.repo.GetAll()
    if err != nil {
        return err
    }

    for _, user := range users {
        key := fmt.Sprintf("user:%d", user.ID)
        if err := s.cache.Set(key, user, time.Hour); err != nil {
            return err
        }
    }
    return nil
}
```

### 2. 缓存穿透保护

```go
func (s *UserService) GetUser(id uint) (*models.User, error) {
    key := fmt.Sprintf("user:%d", id)
    
    // 使用空值缓存防止缓存穿透
    user, err := s.cache.Remember(key, time.Hour, func() interface{} {
        user, err := s.repo.Find(id)
        if err != nil {
            if err == gorm.ErrRecordNotFound {
                return nil // 缓存空值
            }
            return err
        }
        return user
    })
    
    if user == nil {
        return nil, gorm.ErrRecordNotFound
    }
    
    return user.(*models.User), nil
}
```

### 3. 缓存击穿保护

```go
func (s *UserService) GetHotData(key string) (interface{}, error) {
    // 使用互斥锁防止缓存击穿
    value, err := s.cache.Get(key)
    if err == nil {
        return value, nil
    }

    s.mutex.Lock()
    defer s.mutex.Unlock()

    // 双重检查
    value, err = s.cache.Get(key)
    if err == nil {
        return value, nil
    }

    // 从数据源获取
    value, err = s.getFromSource(key)
    if err != nil {
        return nil, err
    }

    // 设置缓存
    s.cache.Set(key, value, time.Hour)
    return value, nil
}
```

## 最佳实践

1. 缓存设计：
   - 选择合适的缓存驱动
   - 设置合理的过期时间
   - 使用有意义的键名前缀

2. 性能优化：
   - 批量获取和存储
   - 合理使用标签
   - 实现缓存预热

3. 数据一致性：
   - 及时清理过期数据
   - 实现缓存同步机制
   - 处理并发更新

4. 安全性：
   - 限制缓存大小
   - 防止缓存穿透
   - 保护敏感数据

## 监控和维护

### 1. 缓存统计

```go
type CacheStats struct {
    Hits        uint64
    Misses      uint64
    Size        uint64
    ItemsCount  uint64
}

func (c *Cache) GetStats() *CacheStats {
    return &CacheStats{
        Hits:       atomic.LoadUint64(&c.hits),
        Misses:     atomic.LoadUint64(&c.misses),
        Size:       c.size(),
        ItemsCount: c.count(),
    }
}
```

### 2. 缓存清理

```go
// 清理过期项
func (c *Cache) GC() {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    for key, item := range c.store {
        if item.isExpired() {
            delete(c.store, key)
        }
    }
}
```

## 缓存驱动实现

### 1. 文件缓存驱动

文件缓存将数据序列化后存储到本地文件系统，具有以下特点：

- **持久化**: 数据自动保存到磁盘，重启后仍然有效
- **无依赖**: 不需要外部服务，开箱即用
- **轻量级**: 内存占用小，适合资源受限的环境
- **自动清理**: 内置定时清理过期项的机制

```go
// pkg/cache/file.go
type FileStore struct {
    dir     string              // 缓存目录
    prefix  string              // 键前缀
    items   map[string]item     // 内存中的缓存项索引
    mu      sync.RWMutex        // 读写锁
    janitor *janitor            // 清理过期项的守护进程
}

// 缓存项结构
type item struct {
    Value      interface{}  // 缓存值
    Expiration int64        // 过期时间(纳秒)
}
```

### 2. Redis 缓存驱动

Redis缓存使用Redis服务器存储数据：

```go
// pkg/cache/redis.go  
type RedisStore struct {
    client *redis.Client  // Redis客户端
    prefix string         // 键前缀
}

func (s *RedisStore) Get(ctx context.Context, key string) (interface{}, error) {
    val, err := s.client.Get(ctx, s.getKey(key)).Result()
    if err == redis.Nil {
        return nil, fmt.Errorf("key not found: %s", key)
    }
    if err != nil {
        return nil, err
    }

    var value interface{}
    if err := json.Unmarshal([]byte(val), &value); err != nil {
        return nil, err
    }

    return value, nil
}
```

## 使用示例

### 缓存用户数据

```go
package services

import (
    "context"
    "fmt"
    "time"
    "go-admin-scaffold/internal/models"
    "go-admin-scaffold/pkg/cache"
)

type UserService struct {
    cache cache.Store
    repo  *UserRepository
}

// 获取用户信息（带缓存）
func (s *UserService) GetUser(ctx context.Context, userID uint) (*models.User, error) {
    cacheKey := fmt.Sprintf("user:%d", userID)
    
    // 尝试从缓存获取
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
        if user, ok := cached.(*models.User); ok {
            return user, nil
        }
    }
    
    // 缓存未命中，从数据库查询
    user, err := s.repo.FindByID(userID)
    if err != nil {
        return nil, err
    }
    
    // 存入缓存，1小时过期
    s.cache.Set(ctx, cacheKey, user, time.Hour)
    
    return user, nil
}

// 使用Remember方法简化缓存逻辑
func (s *UserService) GetUserWithRemember(ctx context.Context, userID uint) (*models.User, error) {
    cacheKey := fmt.Sprintf("user:%d", userID)
    
    result, err := s.cache.Remember(ctx, cacheKey, time.Hour, func() (interface{}, error) {
        return s.repo.FindByID(userID)
    })
    
    if err != nil {
        return nil, err
    }
    
    return result.(*models.User), nil
}

// 更新用户时清除缓存
func (s *UserService) UpdateUser(ctx context.Context, userID uint, updates map[string]interface{}) error {
    if err := s.repo.Update(userID, updates); err != nil {
        return err
    }
    
    // 清除用户缓存
    cacheKey := fmt.Sprintf("user:%d", userID)
    s.cache.Delete(ctx, cacheKey)
    
    return nil
}
```

### 缓存配置和初始化

```go
package main

import (
    "log"
    "go-admin-scaffold/internal/config"
    "go-admin-scaffold/pkg/cache"
)

func initCache(cfg *config.Config) cache.Store {
    // 创建缓存管理器
    manager := cache.NewManager(&cache.Config{
        Driver:  cfg.Cache.Driver,
        Prefix:  cfg.Cache.Prefix,
        Options: cfg.Cache.Options,
    })
    
    // 获取缓存存储实例
    store, err := manager.Store(cfg.Cache.Driver)
    if err != nil {
        log.Fatalf("Failed to initialize cache: %v", err)
    }
    
    return store
}
```

## 文件缓存配置选项

### 基本配置

```yaml
cache:
  driver: "file"
  prefix: "myapp:"
  options:
    file_path: "storage/cache"  # 缓存文件目录
    ttl: 3600                   # 默认过期时间
```

### 高级配置

```go
// 自定义文件缓存配置
type FileStoreConfig struct {
    Directory       string        // 缓存目录
    Prefix          string        // 键前缀  
    CleanupInterval time.Duration // 清理间隔
    DefaultTTL      time.Duration // 默认TTL
}

func NewCustomFileStore(config *FileStoreConfig) *FileStore {
    store := &FileStore{
        dir:    config.Directory,
        prefix: config.Prefix,
        items:  make(map[string]item),
    }
    
    // 自定义清理间隔
    store.janitor = &janitor{
        store:    store,
        interval: config.CleanupInterval,
        stopChan: make(chan bool),
    }
    
    store.janitor.start()
    return store
}
```

## 最佳实践

### 1. 选择合适的缓存驱动

```go
// 开发环境使用文件缓存
if cfg.App.Env == "development" {
    cfg.Cache.Driver = "file"
}

// 生产环境使用Redis缓存
if cfg.App.Env == "production" {
    cfg.Cache.Driver = "redis"
}
```

### 2. 缓存键命名规范

```go
// 使用有意义的前缀和层次结构
const (
    UserCachePrefix      = "user:"
    PermissionCachePrefix = "perm:"
    ConfigCachePrefix    = "config:"
)

func getUserCacheKey(userID uint) string {
    return fmt.Sprintf("%s%d", UserCachePrefix, userID)
}

func getPermissionCacheKey(roleID uint) string {
    return fmt.Sprintf("%s%d", PermissionCachePrefix, roleID)
}
```

### 3. 缓存失效策略

```go
// 标签式缓存清理
func (s *UserService) ClearUserCache(userID uint) {
    ctx := context.Background()
    
    // 清理相关的缓存键
    keys := []string{
        fmt.Sprintf("user:%d", userID),
        fmt.Sprintf("user:%d:profile", userID),
        fmt.Sprintf("user:%d:permissions", userID),
    }
    
    for _, key := range keys {
        s.cache.Delete(ctx, key)
    }
}
```

### 4. 错误处理

```go
func (s *UserService) GetUser(ctx context.Context, userID uint) (*models.User, error) {
    cacheKey := fmt.Sprintf("user:%d", userID)
    
    // 缓存错误不应该影响主要业务逻辑
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
        if user, ok := cached.(*models.User); ok {
            return user, nil
        }
    }
    
    // 从数据库获取
    user, err := s.repo.FindByID(userID)
    if err != nil {
        return nil, err
    }
    
    // 尝试缓存，但不处理缓存错误
    s.cache.Set(ctx, cacheKey, user, time.Hour)
    
    return user, nil
}
```

文件缓存为单机部署和资源受限的环境提供了一个简单可靠的缓存解决方案，而Redis缓存则更适合高并发和分布式场景。根据你的具体需求选择合适的缓存驱动。 