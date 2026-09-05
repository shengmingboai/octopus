import { useTranslations } from 'use-intl';
import { RefreshCw, Clock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { SettingKey } from '@/api/setting';
import { SettingInput } from './SettingInput';
import { useLastSyncTime, useSyncAllChannels } from '@/api/channel';
import { toast } from 'sonner';

// SettingSync 提供渠道模型同步的全局配置: 同步间隔与手动同步。
// 与自动同步共用后端同一编排: 探测成功且列表里没有的授权标记为上游消失, 恢复的按原协议复活。
export function SettingSync() {
    const t = useTranslations('setting');
    const syncAllChannels = useSyncAllChannels();
    const { data: lastSyncTime } = useLastSyncTime();

    const handleManualSync = () => {
        syncAllChannels.mutate(undefined, {
            onSuccess: (result) => {
                toast.success(t('llmSync.syncSuccess', { models: result.added_models, missing: result.missing_grants, restored: result.restored_grants }));
            },
            onError: (error) => {
                toast.error(t('llmSync.syncFailed'), { description: error.message });
            }
        });
    };

    const formatLastSyncTime = (timeStr: string | undefined) => {
        if (!timeStr) return t('llmSync.neverSynced');
        const date = new Date(timeStr);
        if (isNaN(date.getTime())) return t('llmSync.neverSynced');
        return date.toLocaleString();
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <RefreshCw className="h-5 w-5" />
                {t('llmSync.title')}
            </h2>

            {/* 同步间隔 */}
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Clock className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('llmSync.syncInterval.label')}</span>
                </div>
                <SettingInput
                    settingKey={SettingKey.SyncModelsInterval}
                    aria-label={t('llmSync.syncInterval.label')}
                    type="number"
                    min={0}
                    step={1}
                    placeholder={t('llmSync.syncInterval.placeholder')}
                    className="w-40 max-w-[45%] shrink-0 rounded-xl"
                />
            </div>

            {/* 手动同步 */}
            <div className="flex items-center justify-between gap-4">
                <div className="flex flex-col gap-1">
                    <div className="flex items-center gap-3">
                        <RefreshCw className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('llmSync.manualSync.label')}</span>
                    </div>
                    <span className="text-xs text-muted-foreground ml-8">
                        {t('llmSync.lastSync')}: {formatLastSyncTime(lastSyncTime)}
                    </span>
                </div>
                <Button
                    variant="outline"
                    size="sm"
                    onClick={handleManualSync}
                    disabled={syncAllChannels.isPending}
                    className="rounded-xl"
                >
                    {syncAllChannels.isPending ? (
                        <>
                            <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                            {t('llmSync.manualSync.syncing')}
                        </>
                    ) : (
                        <>
                            <RefreshCw className="h-4 w-4 mr-2" />
                            {t('llmSync.manualSync.button')}
                        </>
                    )}
                </Button>
            </div>
        </div>
    );
}
