package op

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/shengmingboai/octopus/internal/db"
	"github.com/shengmingboai/octopus/internal/model"
	"github.com/charmbracelet/log"
)

// AutoGroupChannel 把渠道内命中的模型授权追加到匹配分组的底部, 已在分组中的成员不重复添加。
// 模型名与分组名都在归一化后比较: 去掉 / 前缀并转小写, 故 "z-ai/glm-5.3-flash",
// "glm-5.3-flash" 与 "GLM-5.3-Flash" 都命中 "glm-5.3-flash" 分组。
// 只处理启用中的模型: 被自动拉取禁用的模型上游已下架, 不纳入分组。
func AutoGroupChannel(detail *model.ChannelDetail, ctx context.Context) error {
	if detail == nil || !detail.AutoGroup || !detail.Enabled {
		return nil
	}

	groupByKey := make(map[string]model.Group)
	for _, group := range groupCache.GetAll() {
		key := normalizeModelName(group.Name)
		if key == "" {
			continue
		}
		groupByKey[key] = group
	}
	if len(groupByKey) == 0 {
		return nil
	}

	// 收集每个分组待追加的授权主键; 已在分组中的授权跳过。
	toAdd := make(map[int][]int)
	existingGrant := make(map[int]map[int]struct{})
	for _, group := range groupCache.GetAll() {
		set := make(map[int]struct{}, len(group.Items))
		for _, item := range group.Items {
			set[item.ChannelGrantID] = struct{}{}
		}
		existingGrant[group.ID] = set
	}

	for _, grant := range channelGrantCache.GetAll() {
		channelModel, ok := channelModelCache.Get(grant.ChannelModelID)
		if !ok || channelModel.ChannelID != detail.ID || !channelModel.Enabled {
			continue
		}
		group, ok := groupByKey[normalizeModelName(channelModel.Name)]
		if !ok {
			continue
		}
		if _, exists := existingGrant[group.ID][grant.ID]; exists {
			continue
		}
		toAdd[group.ID] = append(toAdd[group.ID], grant.ID)
	}
	if len(toAdd) == 0 {
		return nil
	}

	// 分组与授权均按主键定序, 追加顺序与日志输出不随 map 遍历变化。
	groupIDs := make([]int, 0, len(toAdd))
	for groupID := range toAdd {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Ints(groupIDs)

	conn := db.GetDB().WithContext(ctx)
	for _, groupID := range groupIDs {
		grantIDs := toAdd[groupID]
		sort.Ints(grantIDs)

		var maxPriority int
		if err := conn.Model(&model.GroupItem{}).
			Where("group_id = ?", groupID).
			Select("COALESCE(MAX(priority), 0)").
			Scan(&maxPriority).Error; err != nil {
			return fmt.Errorf("failed to load group %d max priority: %w", groupID, err)
		}
		for _, grantID := range grantIDs {
			maxPriority++
			if err := conn.Create(&model.GroupItem{
				GroupID:        groupID,
				ChannelGrantID: grantID,
				Priority:       maxPriority,
			}).Error; err != nil {
				return fmt.Errorf("failed to add grant %d to group %d: %w", grantID, groupID, err)
			}
			log.Infof("channel %s: grant %d added to group %d", detail.Name, grantID, groupID)
		}
	}

	if err := groupRefreshCache(ctx); err != nil {
		return fmt.Errorf("failed to refresh group cache: %w", err)
	}
	return nil
}

// normalizeModelName 去掉模型名前的渠道商段并转小写, 使不同写法归一化到同一键。
// 例如 "z-ai/glm-5.3-flash" 与 "GLM-5.3-Flash" 都归一化为 "glm-5.3-flash"。
func normalizeModelName(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.Index(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return strings.ToLower(strings.TrimSpace(name))
}
