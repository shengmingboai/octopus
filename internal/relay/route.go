package relay

import (
	"maps"
	"sync"
	"time"

	"github.com/shengmingboai/octopus/internal/model"
)

// RouteState 是一个分组的进程内路由状态; 跨该分组的全部请求共享。
// 同时作为路由流的消息形状与分组读取响应中的 runtime 字段: 冷却, 探测与亲和都是本包路由算法的概念,
// 故状态形状由本包定义, 分组的持久化配置不含它; 内部标志未导出, 不会随消息出到 JSON。
// 两种模式共用 CurrentItemID: 手动模式下即人工指定的成员, 故障转移模式下由路由决定,
// 前端由此只读这一个字段即可知道当前承载请求的成员, 无需再按模式分支。
type RouteState struct {
	GroupID       int           `json:"group_id"`        // 状态所属的分组 ID, 供状态流按分组定位。
	CurrentItemID int           `json:"current_item_id"` // 当前承载请求的成员 ID, 0 表示尚未建立路由或未人工指定。
	ProbeItemID   int           `json:"probe_item_id"`   // 当前占用恢复探测的成员 ID, 同一分组同时只允许一个成员被探测; 手动模式恒为 0。
	AffinityUntil int64         `json:"affinity_until"`  // 当前路由的亲和截止 Unix 毫秒时间, 0 表示无亲和; 手动模式恒为 0。
	Cooldowns     map[int]int64 `json:"cooldowns"`       // 失败成员 ID 对应的冷却截止 Unix 毫秒时间, 已到期的条目由前端按当前时间忽略。

	affinityArmed bool          // 当前路由下一次成功后是否开始亲和, 仅故障切换后为真。
	trips         map[int]int   // 成员累计进入冷却的次数, 冷却时长按它指数退避, 成功后重置; 不随状态流出 JSON。
}

const routeStreamBuffer = 16 // 单个路由流连接的非阻塞消息缓冲容量。

var (
	routeMu      sync.Mutex                           // routeMu 保护全部分组路由状态。
	routes       = make(map[int]*RouteState)          // routes 按分组 ID 保存路由状态。
	routeStreams = make(map[chan RouteState]struct{}) // 全部路由 SSE 连接。
)

// RouteStateOf 返回分组当前的实时路由状态, 供读取接口随分组一并返回。
// 手动模式没有进程内路由: 当前成员即人工指定的成员, 冷却与亲和均不适用, 故直接由分组配置得出。
func RouteStateOf(group model.Group) RouteState {
	if group.Mode == model.GroupModeManual {
		return RouteState{
			GroupID:       group.ID,
			CurrentItemID: group.ActiveItemID,
			Cooldowns:     map[int]int64{},
		}
	}

	routeMu.Lock()
	defer routeMu.Unlock()

	route := routes[group.ID]
	if route == nil {
		return RouteState{GroupID: group.ID, Cooldowns: map[int]int64{}}
	}
	state := *route
	state.Cooldowns = maps.Clone(route.Cooldowns)
	return state
}

// ResetRouteState 丢弃分组的进程内路由状态, 用于分组切换选择模式或被删除。
// 不丢弃的话冷却与亲和会在 failover 切到 manual 再切回来之后复活并继续影响选路, 分组删除后其状态也会永久残留。
func ResetRouteState(groupID int) {
	routeMu.Lock()
	defer routeMu.Unlock()

	delete(routes, groupID)
}

// pickGroupItem 按分组模式选择本轮目标成员, 没有可用成员时返回零值; group.Items 已按 Priority 升序排列。
// skip 为本请求内已耗尽尝试的成员集合: 不冷却成员不写入跨请求冷却表, 只能由此在本请求内跳过。
// 不可用成员(渠道或凭据被停用, 两侧被删除)不参与选路, 也不会被登记为当前路由;
// 可用性取自分组快照的实时口径, 调用方仍保留权威校验以兜底两次读取之间的状态变化。
func pickGroupItem(group model.Group, skip map[int]bool) model.GroupItem {
	if group.Mode == model.GroupModeManual {
		for _, item := range group.Items {
			if item.ID == group.ActiveItemID {
				return item
			}
		}
		return model.GroupItem{}
	}

	routeMu.Lock()
	defer routeMu.Unlock()

	route := groupRouteLocked(group)
	now := time.Now().UnixMilli()
	if route.AffinityUntil <= now {
		route.AffinityUntil = 0
	}

	// 亲和期内沿用当前成员, 不提前探测已恢复的高优先级成员; 当前成员被本请求跳过时亲和让位。
	if route.CurrentItemID != 0 && route.AffinityUntil > now && !skip[route.CurrentItemID] {
		return itemOf(group, route.CurrentItemID)
	}

	for _, item := range group.Items {
		// 遍历到当前成员说明比它优先级更高的成员都不可选, 沿用当前成员; 当前成员被本请求跳过时继续看更低优先级成员。
		if item.ID == route.CurrentItemID && !skip[item.ID] {
			break
		}
		if skip[item.ID] || !item.Available {
			continue
		}
		deadline, cooling := route.Cooldowns[item.ID]
		if cooling && deadline > now {
			continue
		}
		// 冷却已到期的成员只放行一个探测请求, 避免全部请求同时涌向尚未恢复的成员。
		if cooling {
			if route.ProbeItemID != 0 {
				continue
			}
			route.ProbeItemID = item.ID
			publishRouteLocked(route)
			return item
		}
		route.CurrentItemID = item.ID
		publishRouteLocked(route)
		return item
	}
	// 全部成员都不可选时沿用当前成员兜底; 它被本请求跳过时放弃, 交由调用方等待。
	if route.CurrentItemID != 0 && !skip[route.CurrentItemID] {
		return itemOf(group, route.CurrentItemID)
	}
	return model.GroupItem{}
}

// recordRouteSuccess 上报一轮成功: 结束该成员的冷却与探测占用, 并在故障切换后按配置开始亲和。
func recordRouteSuccess(group model.Group, itemID int) {
	if group.Mode == model.GroupModeManual {
		return
	}

	routeMu.Lock()
	defer routeMu.Unlock()

	route := routes[group.ID]
	if route == nil {
		return
	}
	now := time.Now().UnixMilli()
	changed := false

	// 探测成功说明该成员已恢复, 解除冷却并重置退避计数; 若当前路由不在亲和期内则立即切回该成员。
	if route.ProbeItemID == itemID {
		route.ProbeItemID = 0
		delete(route.Cooldowns, itemID)
		delete(route.trips, itemID)
		if route.CurrentItemID == 0 || route.AffinityUntil <= now {
			route.CurrentItemID = itemID
			route.AffinityUntil = 0
		}
		changed = true
	}
	// 亲和只在故障切换后的首次成功时开始, 使请求在一段时间内稳定留在备用成员上。
	if route.CurrentItemID == itemID && route.affinityArmed {
		route.affinityArmed = false
		if seconds := failoverAffinitySeconds(); seconds > 0 {
			route.AffinityUntil = now + int64(seconds)*1000
			changed = true
		}
	}
	if changed {
		publishRouteLocked(route)
	}
}

// recordRouteFailure 上报一轮失败: 达到配置的总尝试次数后将该成员打入冷却并让出当前路由, 返回是否应换成员。
// failures 为该成员在本请求内包含首次请求的连续失败次数, 由调用方累计。
// noCooldown 渠道不写入跨请求冷却表: 只让出当前路由, 由调用方在本请求内跳过该成员, 后续请求继续选它。
func recordRouteFailure(group model.Group, itemID, failures int, noCooldown bool) bool {
	if group.Mode == model.GroupModeManual {
		return false
	}

	routeMu.Lock()
	defer routeMu.Unlock()

	route := routes[group.ID]
	if route == nil {
		return false
	}
	// 探测请求只有一次机会, 常规成员达到全局策略的总尝试次数后进入冷却。
	if route.ProbeItemID != itemID && failures < failoverMaxAttempts() {
		return false
	}

	now := time.Now().UnixMilli()
	if noCooldown {
		if route.ProbeItemID == itemID {
			route.ProbeItemID = 0
		}
		if route.CurrentItemID == itemID {
			route.CurrentItemID = 0
			route.AffinityUntil = 0
			route.affinityArmed = true
		}
		publishRouteLocked(route)
		return true
	}
	// 冷却按连续熔断次数指数退避: 每次耗尽尝试翻倍, 封顶全局上限, 成功一次即重置回基准。
	route.trips[itemID]++
	route.Cooldowns[itemID] = now + int64(failoverCooldownSeconds(route.trips[itemID]))*1000
	if route.ProbeItemID == itemID {
		route.ProbeItemID = 0
	}
	// 当前路由失败才需要下一个成员开始亲和; 独立探测失败不影响当前路由。
	if route.CurrentItemID == itemID {
		route.CurrentItemID = 0
		route.AffinityUntil = 0
		route.affinityArmed = true
	}
	publishRouteLocked(route)
	return true
}

// releaseRouteProbe 归还未产生成败结论的探测占用, 用于请求被人工中止或客户端断开。
func releaseRouteProbe(group model.Group, itemID int) {
	routeMu.Lock()
	defer routeMu.Unlock()

	if route := routes[group.ID]; route != nil && route.ProbeItemID == itemID {
		route.ProbeItemID = 0
		publishRouteLocked(route)
	}
}

// groupRouteLocked 取出分组路由状态并清理已删除或不可用成员的残留, 有清理时发布一次; 调用方必须持有锁。
func groupRouteLocked(group model.Group) *RouteState {
	route := routes[group.ID]
	if route == nil {
		route = &RouteState{GroupID: group.ID, Cooldowns: make(map[int]int64), trips: make(map[int]int)}
		routes[group.ID] = route
	}
	// items 的值为该成员是否可用: 冷却只清已删除的成员, 不可用成员的冷却保留, 重新启用后继续生效。
	items := make(map[int]bool, len(group.Items))
	for _, item := range group.Items {
		items[item.ID] = item.Available
	}
	changed := false
	for itemID := range route.Cooldowns {
		if _, exists := items[itemID]; !exists {
			delete(route.Cooldowns, itemID)
			delete(route.trips, itemID)
			changed = true
		}
	}
	// 当前路由与探测占用让不可用成员立即让位, 否则禁用成员会残留勾选并继续显示为当前路由。
	if route.ProbeItemID != 0 && !items[route.ProbeItemID] {
		route.ProbeItemID = 0
		changed = true
	}
	if route.CurrentItemID != 0 && !items[route.CurrentItemID] {
		route.CurrentItemID = 0
		route.AffinityUntil = 0
		route.affinityArmed = false
		changed = true
	}
	if changed {
		publishRouteLocked(route)
	}
	return route
}

// itemOf 返回分组内指定 ID 的成员, 不存在时返回零值。
func itemOf(group model.Group, itemID int) model.GroupItem {
	for _, item := range group.Items {
		if item.ID == itemID {
			return item
		}
	}
	return model.GroupItem{}
}

// publishRouteLocked 非阻塞发布路由状态, 连接拥塞时关闭它并交给客户端重连获取全量快照; 冷却表按值复制以免前端读到后续变更; 调用方必须持有锁。
func publishRouteLocked(route *RouteState) {
	message := *route
	message.Cooldowns = maps.Clone(route.Cooldowns)
	for stream := range routeStreams {
		select {
		case stream <- message:
		default:
			delete(routeStreams, stream)
			close(stream)
		}
	}
}

// OpenRouteStream 注册路由流连接, 返回后续增量通道。
// 不再返回快照: 分组读取接口已随分组带回当前路由状态, 前端由此拿到的初始值即全量, 连接只负责增量。
func OpenRouteStream() chan RouteState {
	routeMu.Lock()
	defer routeMu.Unlock()

	stream := make(chan RouteState, routeStreamBuffer)
	routeStreams[stream] = struct{}{}
	return stream
}

// CloseRouteStream 注销并关闭指定路由流连接。
func CloseRouteStream(stream chan RouteState) {
	routeMu.Lock()
	defer routeMu.Unlock()

	if _, exists := routeStreams[stream]; exists {
		delete(routeStreams, stream)
		close(stream)
	}
}
