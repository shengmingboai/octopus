package op

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"

	"github.com/shengmingboai/octopus/internal/db"
	"github.com/shengmingboai/octopus/internal/model"
	"github.com/shengmingboai/octopus/internal/utils/cache"
	"gorm.io/gorm"
)

var apiKeyCache = cache.New[int, model.APIKey](16)
var apiKeyIDMap = cache.New[string, int](16)
var apiKeyMu sync.RWMutex // 保护主键缓存和凭据索引的成组更新。

func APIKeyCreate(key *model.APIKey, ctx context.Context) error {
	apiKeyMu.Lock()
	defer apiKeyMu.Unlock()
	if key.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if _, exists := apiKeyIDMap.Get(key.APIKey); exists {
		return fmt.Errorf("API key already exists")
	}
	if err := db.GetDB().WithContext(ctx).Create(key).Error; err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	apiKeyCache.Set(key.ID, cloneAPIKey(*key))
	apiKeyIDMap.Set(key.APIKey, key.ID)
	return nil
}

func APIKeyUpdate(key *model.APIKey, ctx context.Context) error {
	apiKeyMu.Lock()
	defer apiKeyMu.Unlock()
	existing, ok := apiKeyCache.Get(key.ID)
	if !ok {
		return fmt.Errorf("API key not found")
	}
	if key.APIKey == "" {
		key.APIKey = existing.APIKey
	}
	if id, exists := apiKeyIDMap.Get(key.APIKey); exists && id != key.ID {
		return fmt.Errorf("API key already exists")
	}
	if err := db.GetDB().WithContext(ctx).Save(key).Error; err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}
	if key.APIKey != existing.APIKey {
		apiKeyIDMap.Del(existing.APIKey)
		apiKeyIDMap.Set(key.APIKey, key.ID)
	}
	apiKeyCache.Set(key.ID, cloneAPIKey(*key))
	return nil
}

// APIKeyList 返回全部 API Key, 按主键升序定序。
// 设置页不提供排序开关, 而缓存遍历顺序随机, 故顺序须由此处定稿。
func APIKeyList(ctx context.Context) ([]model.APIKey, error) {
	apiKeyMu.RLock()
	defer apiKeyMu.RUnlock()
	keys := make([]model.APIKey, 0, apiKeyCache.Len())
	for _, apiKey := range apiKeyCache.GetAll() {
		keys = append(keys, cloneAPIKey(apiKey))
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].ID < keys[j].ID })
	return keys, nil
}

func APIKeyGet(id int, ctx context.Context) (model.APIKey, error) {
	apiKeyMu.RLock()
	defer apiKeyMu.RUnlock()
	return apiKeyGet(id)
}

func apiKeyGet(id int) (model.APIKey, error) {
	apiKey, ok := apiKeyCache.Get(id)
	if !ok {
		return model.APIKey{}, fmt.Errorf("API key not found")
	}
	return cloneAPIKey(apiKey), nil
}

func APIKeyGetByAPIKey(apiKey string, ctx context.Context) (model.APIKey, error) {
	apiKeyMu.RLock()
	defer apiKeyMu.RUnlock()
	id, ok := apiKeyIDMap.Get(apiKey)
	if !ok {
		return model.APIKey{}, fmt.Errorf("API key not found")
	}
	key, err := apiKeyGet(id)
	if err != nil || key.APIKey != apiKey {
		return model.APIKey{}, fmt.Errorf("API key not found")
	}
	return key, nil
}

func APIKeyDelete(id int, ctx context.Context) error {
	apiKeyMu.Lock()
	defer apiKeyMu.Unlock()
	key, ok := apiKeyCache.Get(id)
	if !ok {
		return fmt.Errorf("API key not found")
	}
	statsSaveLock.Lock()
	defer statsSaveLock.Unlock()
	statsAPIKeyCacheNeedUpdateLock.Lock()
	defer statsAPIKeyCacheNeedUpdateLock.Unlock()
	if err := db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.StatsAPIKey{}, id).Error; err != nil {
			return err
		}
		result := tx.Delete(&model.APIKey{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("API key not found")
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	apiKeyCache.Del(id)
	apiKeyIDMap.Del(key.APIKey)
	statsAPIKeyCache.Del(id)
	delete(statsAPIKeyCacheNeedUpdate, id)
	return nil
}

func apiKeyRefreshCache(ctx context.Context) error {
	apiKeyMu.Lock()
	defer apiKeyMu.Unlock()
	apiKeys := []model.APIKey{}
	if err := db.GetDB().WithContext(ctx).Find(&apiKeys).Error; err != nil {
		return err
	}
	apiKeyCache.Clear()
	apiKeyIDMap.Clear()
	for _, apiKey := range apiKeys {
		apiKeyCache.Set(apiKey.ID, apiKey)
		apiKeyIDMap.Set(apiKey.APIKey, apiKey.ID)
	}
	return nil
}

func cloneAPIKey(key model.APIKey) model.APIKey {
	key.SupportedModels = slices.Clone(key.SupportedModels)
	return key
}
