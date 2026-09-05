import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'use-intl';
import { Shuffle, HelpCircle } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { useSettingList, useSetSetting, SettingKey } from '@/api/setting';
import { toast } from 'sonner';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

// 各字段保存时按整数解析并套用下限, 与后端校验一致。
const FIELDS = [
    { key: SettingKey.FailoverMaxAttempts, stateKey: 'maxAttempts', min: 1 },
    { key: SettingKey.FailoverRetryInterval, stateKey: 'retryInterval', min: 1 },
    { key: SettingKey.FailoverCooldownBase, stateKey: 'cooldownBase', min: 1 },
    { key: SettingKey.FailoverCooldownMax, stateKey: 'cooldownMax', min: 1 },
    { key: SettingKey.FailoverAffinity, stateKey: 'affinity', min: 0 },
] as const;

type FailoverState = Record<(typeof FIELDS)[number]['stateKey'], string>;

// SettingFailover 渲染全局故障转移策略配置: 所有故障转移分组共用这一份参数。
export function SettingFailover() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [values, setValues] = useState<FailoverState>({
        maxAttempts: '',
        retryInterval: '',
        cooldownBase: '',
        cooldownMax: '',
        affinity: '',
    });
    const initialRef = useRef<FailoverState>({ ...values });

    useEffect(() => {
        if (!settings) return;
        const next = { ...values };
        for (const field of FIELDS) {
            const setting = settings.find((s) => s.key === field.key);
            if (setting) next[field.stateKey] = setting.value;
        }
        initialRef.current = { ...next };
        setValues(next);
        // 仅在设置列表加载或刷新时同步一次本地编辑值, 本地未保存的输入不参与依赖。
    }, [settings]);

    const handleSave = (field: (typeof FIELDS)[number], raw: string) => {
        const value = Number.parseInt(raw, 10);
        if (!Number.isFinite(value) || value < field.min) return;
        const normalized = String(value);
        if (normalized === initialRef.current[field.stateKey]) return;

        setSetting.mutate({ key: field.key, value: normalized }, {
            onSuccess: () => {
                toast.success(t('saved'));
                initialRef.current[field.stateKey] = normalized;
            },
        });
    };

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
                    <Input
                        type="number"
                        inputMode="numeric"
                        min={field.min}
                        step={1}
                        value={values[field.stateKey]}
                        onChange={(e) => setValues((prev) => ({ ...prev, [field.stateKey]: e.target.value }))}
                        onBlur={(e) => handleSave(field, e.target.value)}
                        className="w-48 rounded-xl"
                    />
                </div>
            ))}
        </div>
    );
}
