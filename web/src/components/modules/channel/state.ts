import type { ChannelDetail } from '@/api/channel';

// ChannelFormState 是渠道表单的全部可编辑内容。
// 全按名称组织而不存主键: 后端凭据与模型都按名称匹配增删改, 而新建渠道和新加模型时主键尚不存在。
export type ChannelFormState = {
    name: string;
    dialect: ChannelDetail['dialect'];
    base_url: string;
    enabled: boolean;
    proxy: boolean;
    openai_chat_completion_path: string;
    openai_response_path: string;
    anthropic_message_path: string;
    keys: { name: string; key: string; enabled: boolean }[];
    models: string[];
    grants: Map<string, number>; // 键为 grantKey(模型名, 凭据名), 值为 Protocol 位掩码。
    missingGrants: Set<string>; // 已从上游消失的授权。
    custom_header: ChannelDetail['custom_header'];
    channel_proxy: string;
    param_override: string;
    match_regex: string;
    auto_sync_models: boolean;
    auto_group: boolean;
    no_cooldown: boolean;
};

// grantKey 生成授权在状态里的键; 分隔符取 \0, 模型名与凭据名都不会含它。
export function grantKey(modelName: string, keyName: string) {
    return `${modelName}\0${keyName}`;
}

export const emptyFormState: ChannelFormState = {
    name: '',
    dialect: 'generic',
    base_url: '',
    enabled: true,
    proxy: false,
    openai_chat_completion_path: '/v1/chat/completions',
    openai_response_path: '/v1/responses',
    anthropic_message_path: '/v1/messages',
    keys: [],
    models: [],
    grants: new Map(),
    missingGrants: new Set(),
    custom_header: [],
    channel_proxy: '',
    param_override: '',
    match_regex: '',
    auto_sync_models: false,
    auto_group: false,
    no_cooldown: false,
};

// fromChannel 把渠道完整配置还原为表单状态; 授权读写都按名称, 直接建索引即可。
export function fromChannel(channel: ChannelDetail): ChannelFormState {
    return {
        name: channel.name,
        dialect: channel.dialect,
        base_url: channel.base_url,
        enabled: channel.enabled,
        proxy: channel.proxy,
        openai_chat_completion_path: channel.openai_chat_completion_path,
        openai_response_path: channel.openai_response_path,
        anthropic_message_path: channel.anthropic_message_path,
        keys: channel.keys.map(({ name, key, enabled }) => ({ name, key, enabled })),
        models: [...channel.models],
        grants: new Map(channel.grants.map((g) => [grantKey(g.model_name, g.key_name), g.protocols])),
        missingGrants: new Set(channel.grants.filter((g) => g.missing).map((g) => grantKey(g.model_name, g.key_name))),
        custom_header: channel.custom_header,
        channel_proxy: channel.channel_proxy,
        param_override: channel.param_override,
        match_regex: channel.match_regex,
        auto_sync_models: channel.auto_sync_models,
        auto_group: channel.auto_group,
        no_cooldown: channel.no_cooldown,
    };
}

// withGrants 返回替换授权表后的状态; 消失标记只在授权仍存在时保留, 协议位清空即授权删除。
// 授权的增删改有多处入口, 消失标记的清理集中在此, 各入口不会漏删残留的键。
export function withGrants(state: ChannelFormState, grants: Map<string, number>): ChannelFormState {
    return {
        ...state,
        grants,
        missingGrants: new Set([...state.missingGrants].filter((mapKey) => grants.has(mapKey))),
    };
}

// toChannelConfig 生成渠道自身的配置字段, 提交与探测共用。
// 探测只用得上其中的地址, 路径, 代理与过滤表达式, 但必须与保存后生效的完全一致, 故由同一处给出。
export function toChannelConfig(state: ChannelFormState) {
    return {
        name: state.name.trim(),
        dialect: state.dialect,
        enabled: state.enabled,
        base_url: state.base_url.trim(),
        openai_chat_completion_path: state.openai_chat_completion_path.trim(),
        openai_response_path: state.openai_response_path.trim(),
        anthropic_message_path: state.anthropic_message_path.trim(),
        proxy: state.proxy,
        custom_header: state.custom_header.filter((h) => h.header_key.trim() && h.header_value !== ''),
        channel_proxy: state.channel_proxy.trim(),
        param_override: state.param_override.trim(),
        match_regex: state.match_regex.trim(),
        auto_sync_models: state.auto_sync_models,
        auto_group: state.auto_group,
        no_cooldown: state.no_cooldown,
    };
}

// toChannelDetail 把表单状态还原为提交用的完整配置; 创建时 id 取 0, 由后端分配。
// 读写同构, 提交即全量: 无需与原渠道逐字段比对, 表单本就一次给出完整配置。
// 协议位为空的条目不是授权, 在此丢弃; 消失的授权协议位仍在, 原样带回才不会被当作已删除。
export function toChannelDetail(state: ChannelFormState, id: number): ChannelDetail {
    return {
        ...toChannelConfig(state),
        id,
        keys: state.keys.map(({ name, key, enabled }) => ({ name: name.trim(), key: key.trim(), enabled })),
        models: [...state.models],
        grants: [...state.grants]
            .filter(([, protocols]) => protocols !== 0)
            .map(([mapKey, protocols]) => {
                const [model_name, key_name] = mapKey.split('\0');
                return { model_name, key_name, protocols, missing: state.missingGrants.has(mapKey) };
            }),
    };
}
