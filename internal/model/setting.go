package model

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type SettingKey string

const (
	SettingKeyProxyURL                SettingKey = "proxy_url"
	SettingKeyStatsSaveInterval       SettingKey = "stats_save_interval"             // 将统计信息写入数据库的周期(分钟)
	SettingKeyModelInfoUpdateInterval SettingKey = "model_info_update_interval"      // 模型信息更新间隔(小时)
	SettingKeyCORSAllowOrigins        SettingKey = "cors_allow_origins"              // 跨域白名单(逗号分隔, 如 "example.com,example2.com"). 为空不允许跨域, "*"允许所有
	SettingKeySyncModelsInterval      SettingKey = "sync_models_interval"            // 渠道模型自动同步间隔(小时), 0 表示关闭
	SettingKeySyncModelsLastSync      SettingKey = "sync_models_last_sync"           // 最近一次模型同步完成的 Unix 秒; 仅供界面展示, 不在设置页编辑
	SettingKeyFailoverMaxAttempts     SettingKey = "failover_max_attempts"           // 故障转移下单成员包含首次请求的总尝试次数
	SettingKeyFailoverRetryInterval   SettingKey = "failover_retry_interval_seconds" // 同一成员相邻两次尝试之间的等待秒数
	SettingKeyFailoverCooldownBase    SettingKey = "failover_cooldown_base_seconds"  // 成员耗尽尝试后的基准冷却秒数, 连续熔断按指数退避翻倍
	SettingKeyFailoverCooldownMax     SettingKey = "failover_cooldown_max_seconds"   // 指数退避的冷却上限秒数
	SettingKeyFailoverAffinity        SettingKey = "failover_affinity_seconds"       // 故障切换成功后继续保持备用成员的秒数, 0 表示不保持
)

// 故障转移策略各整型设置的默认值; DefaultSettings 与 relay 读取兜底共用, 改动须两处同步语义。
const (
	DefaultFailoverMaxAttempts   = 2   // 默认单个成员尝试 2 次。
	DefaultFailoverRetryInterval = 3   // 默认重试间隔 3 秒。
	DefaultFailoverCooldownBase  = 30  // 默认基准冷却 30 秒。
	DefaultFailoverCooldownMax   = 600 // 默认冷却上限 10 分钟。
	DefaultFailoverAffinity      = 300 // 默认亲和 5 分钟。
)

type Setting struct {
	Key   SettingKey `json:"key" gorm:"primaryKey"`
	Value string     `json:"value" gorm:"not null"`
}

func DefaultSettings() []Setting {
	return []Setting{
		{Key: SettingKeyProxyURL, Value: ""},
		{Key: SettingKeyStatsSaveInterval, Value: "10"},       // 默认10分钟保存一次统计信息
		{Key: SettingKeyCORSAllowOrigins, Value: ""},          // CORS 默认不允许跨域，设置为 "*" 才允许所有来源
		{Key: SettingKeyModelInfoUpdateInterval, Value: "24"}, // 默认24小时更新一次模型信息
		{Key: SettingKeySyncModelsInterval, Value: "24"},      // 默认24小时同步一次渠道模型
		{Key: SettingKeySyncModelsLastSync, Value: ""},        // 最近一次模型同步完成时间, 由同步流程写入
		{Key: SettingKeyFailoverMaxAttempts, Value: strconv.Itoa(DefaultFailoverMaxAttempts)},
		{Key: SettingKeyFailoverRetryInterval, Value: strconv.Itoa(DefaultFailoverRetryInterval)},
		{Key: SettingKeyFailoverCooldownBase, Value: strconv.Itoa(DefaultFailoverCooldownBase)},
		{Key: SettingKeyFailoverCooldownMax, Value: strconv.Itoa(DefaultFailoverCooldownMax)},
		{Key: SettingKeyFailoverAffinity, Value: strconv.Itoa(DefaultFailoverAffinity)},
	}
}

func (key SettingKey) IntervalUnit() time.Duration {
	switch key {
	case SettingKeyStatsSaveInterval:
		return time.Minute
	case SettingKeyModelInfoUpdateInterval, SettingKeySyncModelsInterval:
		return time.Hour
	default:
		return 0
	}
}

func (s *Setting) Validate() error {
	if unit := s.Key.IntervalUnit(); unit != 0 {
		value, err := strconv.ParseInt(s.Value, 10, 64)
		if err != nil || value < 0 || value > int64((1<<63-1)/unit) {
			return fmt.Errorf("%s must be a non-negative integer within the supported duration", s.Key)
		}
		return nil
	}
	switch s.Key {
	case SettingKeyFailoverMaxAttempts, SettingKeyFailoverRetryInterval, SettingKeyFailoverCooldownBase, SettingKeyFailoverCooldownMax:
		value, err := strconv.Atoi(s.Value)
		if err != nil {
			return fmt.Errorf("%s must be an integer", s.Key)
		}
		if value < 1 {
			return fmt.Errorf("%s must be at least 1", s.Key)
		}
		return nil
	case SettingKeyFailoverAffinity:
		value, err := strconv.Atoi(s.Value)
		if err != nil {
			return fmt.Errorf("failover affinity must be an integer")
		}
		if value < 0 {
			return fmt.Errorf("failover affinity must be at least 0")
		}
		return nil
	case SettingKeyProxyURL:
		if s.Value == "" {
			return nil
		}
		parsedURL, err := url.Parse(s.Value)
		if err != nil {
			return fmt.Errorf("proxy URL is invalid: %w", err)
		}
		validSchemes := map[string]bool{
			"http":    true,
			"https":   true,
			"socks5":  true,
			"socks5h": true,
		}
		if !validSchemes[parsedURL.Scheme] {
			return fmt.Errorf("proxy URL scheme must be http, https, socks5, or socks5h")
		}
		if parsedURL.Host == "" {
			return fmt.Errorf("proxy URL must have a host")
		}
		return nil
	}

	return nil
}
