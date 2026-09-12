package helper

import (
	"context"
	"strings"

	"github.com/shengmingboai/octopus/internal/model"
	"github.com/shengmingboai/octopus/internal/op"
	"github.com/shengmingboai/octopus/internal/price"
)

// LLMPricesAdd 为渠道模型匹配校准价格, 并批量写入尚不存在的价格记录, 渠道创建, 更新与自动拉取共用。
// 价格表以小写模型名为主键, 渠道模型保留大小写, 故在此统一转小写去重。
func LLMPricesAdd(channelModels []model.ChannelModelConfig, ctx context.Context) error {
	seen := make(map[string]struct{}, len(channelModels))
	llmInfos := make([]model.LLMInfo, 0, len(channelModels))
	for _, channelModel := range channelModels {
		modelName := strings.ToLower(channelModel.Name)
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		llmInfo := model.LLMInfo{Name: modelName}
		if modelPrice := price.GetLLMPrice(modelName); modelPrice != nil {
			llmInfo.LLMPrice = *modelPrice
		}
		llmInfos = append(llmInfos, llmInfo)
	}
	return op.LLMBatchCreate(llmInfos, ctx)
}
