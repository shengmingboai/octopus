import { useState, type ComponentProps } from 'react';
import { useTranslations } from 'use-intl';
import { toast } from 'sonner';
import { useSettingList, useSetSetting } from '@/api/setting';
import { Input } from '@/components/ui/input';

type SettingInputProps = Omit<ComponentProps<typeof Input>, 'value' | 'defaultValue' | 'onChange' | 'onBlur'> & {
    settingKey: string;
};

// 只有未保存的草稿进入本地状态，查询刷新不会覆盖用户正在编辑的输入。
export function SettingInput({ settingKey, type = 'text', disabled, ...props }: SettingInputProps) {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const mutation = useSetSetting();
    const saved = settings?.find((setting) => setting.key === settingKey)?.value;
    const [draft, setDraft] = useState<string | null>(null);

    return (
        <Input
            {...props}
            type={type}
            required={type === 'number' || props.required}
            disabled={disabled || saved === undefined || mutation.isPending}
            value={draft ?? saved ?? ''}
            onChange={(event) => setDraft(event.target.value)}
            onBlur={(event) => {
                if (!event.currentTarget.reportValidity()) return;
                const submitted = event.currentTarget.value;
                const value = type === 'number' ? String(event.currentTarget.valueAsNumber) : submitted;
                if (value === saved) {
                    setDraft(null);
                    return;
                }
                mutation.mutate({ key: settingKey, value }, {
                    onSuccess: () => {
                        setDraft((current) => current === submitted ? null : current);
                        toast.success(t('saved'));
                    },
                    onError: (error) => toast.error(error.message),
                });
            }}
        />
    );
}
