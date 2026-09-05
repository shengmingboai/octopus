import { Fragment } from 'react';
import { useTranslations } from 'use-intl';
import type { MetricKey } from './store';

const METRIC_KEYS = ['cost', 'count', 'tokens'] as const;

// MetricTabs 渲染以 / 分隔的统计维度切换, 供趋势图和排行榜共用。
// 选中高亮与渠道详情的排序一致: 选中项加重, 未选中项弱化并在悬停时加深。
export function MetricTabs({ value, onChange }: { value: MetricKey; onChange: (value: MetricKey) => void }) {
    const t = useTranslations('home.metric');

    return (
        <div className="flex shrink-0 items-center text-sm">
            {METRIC_KEYS.map((key, index) => (
                <Fragment key={key}>
                    {index > 0 && (
                        <span aria-hidden="true" className="mx-1 text-muted-foreground/40">/</span>
                    )}
                    <button
                        type="button"
                        onClick={() => onChange(key)}
                        aria-pressed={value === key}
                        className={`transition-colors ${value === key
                            ? 'font-medium text-foreground'
                            : 'text-muted-foreground/50 hover:text-muted-foreground'}`}
                    >
                        {t(key)}
                    </button>
                </Fragment>
            ))}
        </div>
    );
}
