'use client';

import { useMemo, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Database, Download, Upload, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { Progress } from '@/components/ui/progress';
import { toast } from '@/components/common/Toast';
import { useExportDB, useImportDB } from '@/api/endpoints/setting';

export function SettingBackup() {
    const t = useTranslations('setting');

    const { status, progress, startExport, cancelExport, reset } = useExportDB();
    const importDB = useImportDB();

    const [includeLogs, setIncludeLogs] = useState(false);
    const [includeStats, setIncludeStats] = useState(false);

    const [file, setFile] = useState<File | null>(null);
    const fileInputRef = useRef<HTMLInputElement | null>(null);

    const rowsAffected = importDB.data?.rows_affected ?? null;
    const rowsAffectedList = useMemo(() => {
        if (!rowsAffected) return [];
        return Object.entries(rowsAffected)
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([k, v]) => ({ table: k, count: v }));
    }, [rowsAffected]);

    const onPickFile = (f: File | null) => {
        setFile(f);
    };

    const onImport = async () => {
        if (!file) {
            toast.error(t('backup.import.noFile'));
            return;
        }
        try {
            await importDB.mutateAsync(file);
            toast.success(t('backup.import.success'));
            if (fileInputRef.current) fileInputRef.current.value = '';
            setFile(null);
        } catch (e) {
            toast.error(e instanceof Error ? e.message : t('backup.import.failed'));
        }
    };

    const onExport = async () => {
        try {
            toast.success(t('backup.export.exporting'));
            await startExport({ include_logs: includeLogs, include_stats: includeStats });
        } catch (e) {
            if (e instanceof Error && e.message === '__CANCELLED__') {
                toast.error(t('backup.export.cancelled'));
                return;
            }
            toast.error(e instanceof Error ? e.message : t('backup.export.failed'));
        }
    };

    const isExporting = status === 'starting' || status === 'exporting';
    const progressPercent = progress && progress.total > 0
        ? Math.round((progress.current / progress.total) * 100)
        : 0;

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Database className="h-5 w-5" />
                {t('backup.title')}
            </h2>

            {/* 导出 */}
            <div className="space-y-3">
                <div className="text-sm font-semibold text-card-foreground">{t('backup.export.title')}</div>

                <div className="flex items-center justify-between gap-4">
                    <div className="text-sm text-muted-foreground">{t('backup.export.includeLogs')}</div>
                    <Switch checked={includeLogs} onCheckedChange={setIncludeLogs} disabled={isExporting} />
                </div>

                <div className="flex items-center justify-between gap-4">
                    <div className="text-sm text-muted-foreground">{t('backup.export.includeStats')}</div>
                    <Switch checked={includeStats} onCheckedChange={setIncludeStats} disabled={isExporting} />
                </div>

                {isExporting ? (
                    <div className="space-y-2">
                        <Progress value={progressPercent} className="h-2 rounded-full" />
                        <div className="flex items-center justify-between text-xs text-muted-foreground">
                            <span>
                                {progress
                                    ? t('backup.export.progress', {
                                        table: progress.table,
                                        current: progress.current,
                                        total: progress.total,
                                    })
                                    : t('backup.export.exporting')}
                            </span>
                            {progress?.rows !== undefined && progress.rows > 0 && (
                                <span className="tabular-nums">{progress.rows.toLocaleString()} rows</span>
                            )}
                        </div>
                        <Button
                            type="button"
                            variant="outline"
                            className="w-full rounded-xl"
                            onClick={cancelExport}
                        >
                            <X className="size-4" />
                            {t('backup.export.cancel')}
                        </Button>
                    </div>
                ) : (
                    <Button
                        type="button"
                        variant="outline"
                        className="w-full rounded-xl"
                        onClick={status === 'done' || status === 'error' ? reset : onExport}
                    >
                        <Download className="size-4" />
                        {status === 'done' ? t('backup.export.completed')
                            : status === 'error' ? t('backup.export.failed')
                            : t('backup.export.button')}
                    </Button>
                )}
            </div>

            <div className="h-px bg-border" />

            {/* 导入 */}
            <div className="space-y-3">
                <div className="text-sm font-semibold text-card-foreground">{t('backup.import.title')}</div>

                <Input
                    ref={fileInputRef}
                    type="file"
                    accept="application/json,.json"
                    onChange={(e) => onPickFile(e.target.files?.[0] ?? null)}
                    className="rounded-xl"
                />

                <Button
                    type="button"
                    variant="destructive"
                    className="w-full rounded-xl"
                    onClick={onImport}
                    disabled={importDB.isPending}
                >
                    <Upload className="size-4" />
                    {importDB.isPending ? t('backup.import.importing') : t('backup.import.button')}
                </Button>

                {rowsAffectedList.length > 0 && (
                    <div className="mt-2 space-y-1">
                        <div className="text-xs font-semibold text-card-foreground">{t('backup.import.result')}</div>
                        <div className="grid grid-cols-2 gap-1 text-xs text-muted-foreground">
                            {rowsAffectedList.map((it) => (
                                <div key={it.table} className="flex justify-between gap-2">
                                    <span className="truncate">{it.table}</span>
                                    <span className="tabular-nums">{it.count}</span>
                                </div>
                            ))}
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
