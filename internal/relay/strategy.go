package relay

import (
	"github.com/shengmingboai/octopus/internal/model"
	"github.com/shengmingboai/octopus/internal/op"
)

// 故障转移策略已全局化到设置页, 不再随分组配置。设置页随时可改, 故每轮使用时现读,
// 读不到(键缺失或值非法)时按默认值兜底, 与 model.DefaultSettings 的默认语义保持一致。

// failoverMaxAttempts 返回单成员包含首次请求的总尝试次数。
func failoverMaxAttempts() int {
	if attempts, err := op.SettingGetInt(model.SettingKeyFailoverMaxAttempts); err == nil && attempts >= 1 {
		return attempts
	}
	return model.DefaultFailoverMaxAttempts
}

// failoverRetryInterval 返回同一成员相邻两次尝试之间的等待秒数。
func failoverRetryInterval() int {
	if seconds, err := op.SettingGetInt(model.SettingKeyFailoverRetryInterval); err == nil && seconds >= 1 {
		return seconds
	}
	return model.DefaultFailoverRetryInterval
}

// failoverCooldownSeconds 按成员的连续熔断次数计算冷却秒数, 指数退避 base*2^(trips-1) 后封顶。
// trips 为 0 或 1 时冷却即基准值; 上限低于基准时直接按上限, 避免配置错误放大冷却。
func failoverCooldownSeconds(trips int) int {
	base, max := model.DefaultFailoverCooldownBase, model.DefaultFailoverCooldownMax
	if value, err := op.SettingGetInt(model.SettingKeyFailoverCooldownBase); err == nil && value >= 1 {
		base = value
	}
	if value, err := op.SettingGetInt(model.SettingKeyFailoverCooldownMax); err == nil && value >= 1 {
		max = value
	}
	cooldown := base
	if trips > 1 {
		shift := trips - 1
		if shift > 20 { // 移位封顶防溢出, 20 次翻倍已远超任何合理的冷却上限。
			shift = 20
		}
		cooldown = base << shift
	}
	if cooldown > max {
		cooldown = max
	}
	return cooldown
}

// failoverAffinitySeconds 返回故障切换成功后继续保持备用成员的秒数, 0 表示不保持。
func failoverAffinitySeconds() int {
	if seconds, err := op.SettingGetInt(model.SettingKeyFailoverAffinity); err == nil && seconds >= 0 {
		return seconds
	}
	return model.DefaultFailoverAffinity
}
