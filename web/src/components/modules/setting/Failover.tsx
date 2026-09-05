import { useTranslations } from 'use-intl';
import { Shuffle, HelpCircle } from 'lucide-react';
import { SettingKey } from '@/api/setting';
import { SettingInput } from './SettingInput';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

// 数字输入的步长和下限与后端校验一致。
const FIELDS = [
    { key: SettingKey.FailoverMaxAttempts, stateKey: 'maxAttempts', min: 1 },
    { key: SettingKey.FailoverRetryInterval, stateKey: 'retryInterval', min: 1 },
    { key: SettingKey.FailoverCooldownBase, stateKey: 'cooldownBase', min: 1 },
    { key: SettingKey.FailoverCooldownMax, stateKey: 'cooldownMax', min: 1 },
    { key: SettingKey.FailoverAffinity, stateKey: 'affinity', min: 0 },
] as const;

// SettingFailover 渲染全局故障转移策略配置: 所有故障转移分组共用这一份参数。
export function SettingFailover() {
    const t = useTranslations('setting');

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Shuffle className="h-5 w-5" />
                {t('failover.title')}
            </h2>

            {FIELDS.map((field) => (
                <div key={field.key} className="flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3 min-w-0">
                        <span className="text-sm font-medium">{t(`failover.${field.stateKey}.label`)}</span>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <HelpCircle className="size-4 text-muted-foreground cursor-help shrink-0" />
                            </TooltipTrigger>
                            <TooltipContent side="top" sideOffset={10} align="center">
                                {t(`failover.${field.stateKey}.hint`)}
                            </TooltipContent>
                        </Tooltip>
                    </div>
                    <SettingInput
                        settingKey={field.key}
                        aria-label={t(`failover.${field.stateKey}.label`)}
                        type="number"
                        inputMode="numeric"
                        min={field.min}
                        step={1}
                        className="w-40 max-w-[45%] shrink-0 rounded-xl"
                    />
                </div>
            ))}
        </div>
    );
}
