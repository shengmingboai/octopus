'use client';

import { useCallback, useMemo, useState } from 'react';
import { useLogs, type LogFilters } from '@/api/endpoints/log';
import { useGroupList } from '@/api/endpoints/group';
import { useChannelList } from '@/api/endpoints/channel';
import { useAPIKeyList } from '@/api/endpoints/apikey';
import { LogCard } from './Item';
import { Loader2, ListFilter, X } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

function LogFilterBar({ filters, onFiltersChange }: {
    filters: LogFilters;
    onFiltersChange: (filters: LogFilters) => void;
}) {
    const t = useTranslations('log.filter');
    const { data: groups } = useGroupList();
    const { data: channels } = useChannelList();
    const { data: apiKeys } = useAPIKeyList();

    const [selectedGroupId, setSelectedGroupId] = useState<string>('');

    const selectedGroup = useMemo(
        () => groups?.find(g => String(g.id) === selectedGroupId),
        [groups, selectedGroupId]
    );

    const groupChannels = useMemo(() => {
        if (!selectedGroup?.items || !channels) return [];
        const channelIds = new Set(selectedGroup.items.map(item => item.channel_id));
        return channels.filter(c => channelIds.has(c.raw.id));
    }, [selectedGroup, channels]);

    const handleGroupChange = (groupId: string) => {
        setSelectedGroupId(groupId);
        if (groupId === '__all__') {
            onFiltersChange({ ...filters, channel_ids: undefined });
        } else {
            const group = groups?.find(g => String(g.id) === groupId);
            if (group?.items) {
                const ids = group.items.map(item => item.channel_id);
                onFiltersChange({ ...filters, channel_ids: ids });
            }
        }
    };

    const handleChannelChange = (channelId: string) => {
        if (channelId === '__all__') {
            if (selectedGroup?.items) {
                const ids = selectedGroup.items.map(item => item.channel_id);
                onFiltersChange({ ...filters, channel_ids: ids });
            }
        } else {
            onFiltersChange({ ...filters, channel_ids: [Number(channelId)] });
        }
    };

    const handleModelNameChange = (value: string) => {
        onFiltersChange({ ...filters, request_model_name: value || undefined });
    };

    const handleAPIKeyChange = (value: string) => {
        onFiltersChange({ ...filters, request_api_key_name: value === '__all__' ? undefined : value });
    };

    const hasFilters = (filters.channel_ids?.length ?? 0) > 0 || !!filters.request_model_name || !!filters.request_api_key_name;

    const handleClear = () => {
        setSelectedGroupId('');
        onFiltersChange({});
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-4">
            <div className="flex items-center gap-2 mb-3">
                <ListFilter className="size-4 text-muted-foreground" />
                <span className="text-sm font-medium text-card-foreground">{t('title')}</span>
                {hasFilters && (
                    <Button
                        variant="ghost"
                        size="sm"
                        className="ml-auto h-7 gap-1 text-xs text-muted-foreground"
                        onClick={handleClear}
                    >
                        <X className="size-3" />
                        {t('clear')}
                    </Button>
                )}
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                <Select value={selectedGroupId} onValueChange={handleGroupChange}>
                    <SelectTrigger className="rounded-xl h-9 w-full text-sm">
                        <SelectValue placeholder={t('group.placeholder')} />
                    </SelectTrigger>
                    <SelectContent className="rounded-xl">
                        <SelectItem className="rounded-xl" value="__all__">{t('group.all')}</SelectItem>
                        {groups?.map(group => (
                            <SelectItem key={group.id} className="rounded-xl" value={String(group.id)}>
                                {group.name}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>

                <Select
                    value={filters.channel_ids?.length === 1 ? String(filters.channel_ids[0]) : '__all__'}
                    onValueChange={handleChannelChange}
                    disabled={!selectedGroupId || selectedGroupId === '__all__'}
                >
                    <SelectTrigger className="rounded-xl h-9 w-full text-sm">
                        <SelectValue placeholder={t('channel.placeholder')} />
                    </SelectTrigger>
                    <SelectContent className="rounded-xl">
                        <SelectItem className="rounded-xl" value="__all__">{t('channel.all')}</SelectItem>
                        {groupChannels.map(ch => (
                            <SelectItem key={ch.raw.id} className="rounded-xl" value={String(ch.raw.id)}>
                                {ch.raw.name}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>

                <Input
                    value={filters.request_model_name ?? ''}
                    onChange={(e) => handleModelNameChange(e.target.value)}
                    placeholder={t('model.placeholder')}
                    className="rounded-xl h-9 text-sm"
                />

                <Select
                    value={filters.request_api_key_name ?? '__all__'}
                    onValueChange={handleAPIKeyChange}
                >
                    <SelectTrigger className="rounded-xl h-9 w-full text-sm">
                        <SelectValue placeholder={t('apikey.placeholder')} />
                    </SelectTrigger>
                    <SelectContent className="rounded-xl">
                        <SelectItem className="rounded-xl" value="__all__">{t('apikey.all')}</SelectItem>
                        {apiKeys?.map(key => (
                            <SelectItem key={key.id} className="rounded-xl" value={key.name}>
                                {key.name}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>
        </div>
    );
}

export function Log() {
    const t = useTranslations('log');
    const [filters, setFilters] = useState<LogFilters>({});
    const { logs, hasMore, isLoading, isLoadingMore, loadMore } = useLogs({ pageSize: 10, filters });

    const canLoadMore = hasMore && !isLoading && !isLoadingMore && logs.length > 0;
    const handleReachEnd = useCallback(() => {
        if (!canLoadMore) return;
        void loadMore();
    }, [canLoadMore, loadMore]);

    const footer = useMemo(() => {
        if (hasMore && (isLoading || isLoadingMore)) {
            return (
                <div className="flex justify-center py-4">
                    <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
            );
        }
        if (!hasMore && logs.length > 0) {
            return (
                <div className="flex justify-center py-4">
                    <span className="text-sm text-muted-foreground">{t('list.noMore')}</span>
                </div>
            );
        }
        return null;
    }, [hasMore, isLoading, isLoadingMore, logs.length, t]);

    return (
        <div className="flex flex-col gap-4 h-full min-h-0">
            <LogFilterBar filters={filters} onFiltersChange={setFilters} />
            <VirtualizedGrid
                items={logs}
                layout="list"
                columns={{ default: 1 }}
                estimateItemHeight={80}
                overscan={8}
                getItemKey={(log) => `log-${log.id}`}
                renderItem={(log) => <LogCard log={log} />}
                footer={footer}
                onReachEnd={handleReachEnd}
                reachEndEnabled={canLoadMore}
                reachEndOffset={2}
            />
        </div>
    );
}
