package probe

// 上游模型探测与渠道模型同步的编排。
// 独立成包是因为探测要构建 HTTP 客户端(rhttp), 而同步落库要用渠道缓存(op):
// rhttp 依赖 op, op 不得反向依赖本包, 故探测与编排放本包, 落库由 op 提供。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dlclark/regexp2"
	"github.com/shengmingboai/octopus/internal/model"
	"github.com/shengmingboai/octopus/internal/op"
	"github.com/shengmingboai/octopus/internal/price"
	"github.com/shengmingboai/octopus/internal/rhttp"
)

// HTTPClient 按渠道配置构建探测用的 HTTP 客户端。
// 渠道专用代理的客户端不共享, 返回的清理函数负责关掉其空闲连接; 共享客户端的清理函数为空。
func HTTPClient(config model.ChannelConfig) (*http.Client, func(), error) {
	switch {
	case !config.Proxy:
		httpClient, err := rhttp.Direct()
		return httpClient, nil, err
	case config.ChannelProxy == "":
		httpClient, err := rhttp.Proxy()
		return httpClient, nil, err
	default:
		httpClient, err := rhttp.New(config.ChannelProxy)
		if httpClient != nil {
			return httpClient, httpClient.CloseIdleConnections, err
		}
		return nil, nil, err
	}
}

// Models 按渠道配置与凭据探测上游模型列表, 并按过滤表达式筛选后返回。
// 同时探测 OpenAI 与 Anthropic 两侧, 谁返回了哪些模型, 就给对应协议位打勾: 协议支持由探测结果决定, 无需用户声明。
// OpenAI 侧记为 Responses 而不是 Chat: Chat Completions 已被官方标记弃用, 新渠道应默认走 Responses,
// 仍需 Chat 的渠道由用户在界面上手动勾选。两侧的 /models 地址与认证形态不同, 故必须分别探测:
// 单协议上游只有一侧会成功, "哪侧成功" 本身就是协议支持的证据。
// 只有两侧都失败才算失败; 一侧失败属正常情况, 单协议上游本就只有一侧讲得通, 按成功那侧的结果返回。
func Models(ctx context.Context, httpClient *http.Client, config model.ChannelConfig, key string, re *regexp2.Regexp) ([]model.ChannelFetchModel, error) {
	var openaiModels, anthropicModels []string
	var openaiErr, anthropicErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		openaiModels, openaiErr = fetchOpenAIModels(httpClient, ctx, config, key, modelsURL(config.BaseURL, config.OpenAIResponsePath))
	}()
	go func() {
		defer wg.Done()
		anthropicModels, anthropicErr = fetchAnthropicModels(httpClient, ctx, config, key, modelsURL(config.BaseURL, config.AnthropicMessagePath))
	}()
	wg.Wait()

	if openaiErr != nil && anthropicErr != nil {
		// 上游鉴权失败或地址不通属于调用方配置问题, 带上两侧的上游原文便于在界面上直接看到原因。
		return nil, fmt.Errorf("openai: %v; anthropic: %v", openaiErr, anthropicErr)
	}

	// 两侧结果按名称合并成一份有序集合: 同名模型在两侧都出现时, 协议位取并集。
	// 保持首次出现的顺序, 界面上模型的排列才与上游返回的一致;
	// 先并入 OpenAI 再并入 Anthropic, 顺序写死而不用 map 遍历, 否则界面上的模型排列会随每次刷新变化。
	protocolsByModel := make(map[string]model.Protocol, len(openaiModels)+len(anthropicModels))
	order := make([]string, 0, len(openaiModels)+len(anthropicModels))
	for _, name := range openaiModels {
		if re != nil {
			matched, err := re.MatchString(name)
			if err != nil {
				return nil, err
			}
			if !matched {
				continue
			}
		}
		if _, ok := protocolsByModel[name]; !ok {
			order = append(order, name)
		}
		protocolsByModel[name] |= model.ProtocolOpenAIResponse
	}
	for _, name := range anthropicModels {
		if re != nil {
			matched, err := re.MatchString(name)
			if err != nil {
				return nil, err
			}
			if !matched {
				continue
			}
		}
		if _, ok := protocolsByModel[name]; !ok {
			order = append(order, name)
		}
		protocolsByModel[name] |= model.ProtocolAnthropicMessage
	}

	models := make([]model.ChannelFetchModel, 0, len(order))
	for _, name := range order {
		models = append(models, model.ChannelFetchModel{Name: name, Protocols: protocolsByModel[name]})
	}
	return models, nil
}

// SyncChannel 同步指定渠道的模型列表: 逐个探测其启用凭据, 再把探测结果交由 op 落库。
// 单个凭据探测失败或返回空列表时跳过该凭据: 只有"探测成功且列表里确实没有"才算上游下架,
// 上游偶发故障不会把现有模型误标成消失。全部凭据都未取得有效探测结果时按失败返回。
// 新引入的模型在此补齐价格记录: 手动同步与定时任务共用本入口, 价格补齐是同步的一部分。
func SyncChannel(channelID int, ctx context.Context) (*model.ChannelSyncResult, error) {
	detail, err := op.ChannelDetailGet(channelID)
	if err != nil {
		return nil, err
	}

	httpClient, cleanup, err := HTTPClient(detail.ChannelConfig)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	var re *regexp2.Regexp
	if detail.MatchRegex != "" {
		if re, err = regexp2.Compile(detail.MatchRegex, regexp2.ECMAScript); err != nil {
			return nil, err
		}
	}

	probeByKeyName := make(map[string][]model.ChannelFetchModel, len(detail.Keys))
	for _, channelKey := range detail.Keys {
		if !channelKey.Enabled || channelKey.Key == "" {
			continue
		}
		models, err := Models(ctx, httpClient, detail.ChannelConfig, channelKey.Key, re)
		if err != nil {
			log.Warnf("channel %d key %q probe failed, skip: %v", channelID, channelKey.Name, err)
			continue
		}
		if len(models) == 0 {
			log.Warnf("channel %d key %q probe returned no models, skip", channelID, channelKey.Name)
			continue
		}
		probeByKeyName[channelKey.Name] = models
	}
	if len(probeByKeyName) == 0 {
		return nil, fmt.Errorf("channel %d has no successful probe result", channelID)
	}

	result, err := op.ChannelSyncApply(channelID, probeByKeyName, ctx)
	if err != nil {
		return nil, err
	}
	if err := addModelPrices(result.AddedModelName, ctx); err != nil {
		return nil, fmt.Errorf("failed to add model prices: %w", err)
	}
	// 记录最近一次同步完成时间, 供设置页展示; 写失败不影响同步结果。
	if err := op.SettingSetString(model.SettingKeySyncModelsLastSync, strconv.FormatInt(time.Now().Unix(), 10)); err != nil {
		log.Warnf("failed to record last sync time: %v", err)
	}
	return result, nil
}

// SyncAllChannels 同步全部开启了自动同步的渠道: 逐个同步, 单个渠道失败不影响其余渠道。
// 返回汇总摘要与失败渠道数, 供设置页的手动同步按钮展示。
func SyncAllChannels(ctx context.Context) (*model.ChannelSyncResult, int, error) {
	summary := &model.ChannelSyncResult{}
	failed := 0
	for _, channelID := range op.AutoSyncChannelIDs() {
		result, err := SyncChannel(channelID, ctx)
		if err != nil {
			log.Warnf("failed to sync models for channel %d: %v", channelID, err)
			failed++
			continue
		}
		summary.AddedModels += result.AddedModels
		summary.MissingGrants += result.MissingGrants
		summary.RestoredGrants += result.RestoredGrants
		summary.AddedModelName = append(summary.AddedModelName, result.AddedModelName...)
	}
	if failed > 0 && len(op.AutoSyncChannelIDs()) == failed {
		return summary, failed, fmt.Errorf("all %d channels failed to sync", failed)
	}
	return summary, failed, nil
}

// addModelPrices 为新引入的渠道模型匹配校准价格, 并批量写入尚不存在的价格记录。
func addModelPrices(modelNames []string, ctx context.Context) error {
	if len(modelNames) == 0 {
		return nil
	}
	llmInfos := make([]model.LLMInfo, 0, len(modelNames))
	for _, modelName := range modelNames {
		llmInfo := model.LLMInfo{Name: strings.ToLower(modelName)}
		if modelPrice := price.GetLLMPrice(modelName); modelPrice != nil {
			llmInfo.LLMPrice = *modelPrice
		}
		llmInfos = append(llmInfos, llmInfo)
	}
	return op.LLMBatchCreate(llmInfos, ctx)
}

// modelsURL 取协议请求路径的父级目录, 与地址拼成同级的 /models 地址。
// 例如 /v1/chat/completions 与 /v1/messages 都得到 /v1/models, /chat/completions 得到 /models。
func modelsURL(baseURL, protocolPath string) string {
	parent := path.Dir(strings.TrimRight(protocolPath, "/"))
	// Anthropic 的 /v1/messages 只有一层, 父级即 /v1; Chat 的 /v1/chat/completions 需要再上一层。
	if strings.HasSuffix(parent, "/chat") {
		parent = path.Dir(parent)
	}
	if parent == "." || parent == "/" {
		parent = ""
	}
	return strings.TrimRight(baseURL, "/") + parent + "/models"
}

// refer: https://platform.openai.com/docs/api-reference/models/list
func fetchOpenAIModels(httpClient *http.Client, ctx context.Context, target model.ChannelConfig, key, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	for _, header := range target.CustomHeader {
		if header.HeaderKey != "" {
			req.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}

	response, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	result, err := decodeModelList[model.OpenAIModelList](response)
	if err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

// refer: https://platform.claude.com/docs
func fetchAnthropicModels(httpClient *http.Client, ctx context.Context, target model.ChannelConfig, key, url string) ([]string, error) {
	var allModels []string
	var afterID string
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Api-Key", key)
		req.Header.Set("Anthropic-Version", "2023-06-01")
		for _, header := range target.CustomHeader {
			if header.HeaderKey != "" {
				req.Header.Set(header.HeaderKey, header.HeaderValue)
			}
		}
		if afterID != "" {
			q := req.URL.Query()
			q.Set("after_id", afterID)
			req.URL.RawQuery = q.Encode()
		}

		response, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		// 分页时每轮都会新建响应, 必须当轮读完即关; 用 defer 会攒到整个函数返回才释放。
		result, err := decodeModelList[model.AnthropicModelList](response)
		if err != nil {
			return nil, err
		}
		for _, m := range result.Data {
			allModels = append(allModels, m.ID)
		}
		if !result.HasMore {
			break
		}
		afterID = result.LastID
	}
	return allModels, nil
}

// decodeModelList 关闭响应并把响应体解成模型列表; 非 2xx 时按上游错误返回。
// 两侧解析流程一致, 只有目标结构不同, 故用类型参数收敛; 分页调用要求当轮读完即关, 关闭点放在此处最稳。
func decodeModelList[T any](response *http.Response) (T, error) {
	defer response.Body.Close()
	var result T
	// 上游报错时响应体常是能被正常解码的 JSON, 若不先拦下, 模型列表会解成空列表并当作成功;
	// 响应体截断到 512 字节: 部分上游在鉴权失败时返回整页 HTML, 全文带到界面上无用。
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, err := io.ReadAll(io.LimitReader(response.Body, 512))
		if err != nil {
			return result, fmt.Errorf("upstream %s", response.Status)
		}
		return result, fmt.Errorf("upstream %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return result, err
	}
	return result, nil
}
