import { useCallback, useEffect, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, API_BASE_URL } from '../client';
import { logger } from '@/lib/logger';
import { useAuthStore } from './user';

/**
 * Setting 数据
 */
export interface Setting {
    key: string;
    value: string;
}

export const SettingKey = {
    ProxyURL: 'proxy_url',
    StatsSaveInterval: 'stats_save_interval',
    ModelInfoUpdateInterval: 'model_info_update_interval',
    SyncLLMInterval: 'sync_llm_interval',
    RelayLogKeepEnabled: 'relay_log_keep_enabled',
    RelayLogKeepPeriod: 'relay_log_keep_period',
    CORSAllowOrigins: 'cors_allow_origins',
    CircuitBreakerThreshold: 'circuit_breaker_threshold',
    CircuitBreakerCooldown: 'circuit_breaker_cooldown',
    CircuitBreakerMaxCooldown: 'circuit_breaker_max_cooldown',
    AllChannelsFailedWait: 'all_channels_failed_wait',
} as const;

/**
 * 获取 Setting 列表 Hook
 * 
 * @example
 * const { data: settings, isLoading, error } = useSettingList();
 * 
 * if (isLoading) return <Loading />;
 * if (error) return <Error message={error.message} />;
 * 
 * settings?.forEach(setting => console.log(setting.key, setting.value));
 */
export function useSettingList() {
    return useQuery({
        queryKey: ['settings', 'list'],
        queryFn: async () => {
            return apiClient.get<Setting[]>('/api/v1/setting/list');
        },
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

/**
 * 设置 Setting Hook
 * 
 * @example
 * const setSetting = useSetSetting();
 * 
 * setSetting.mutate({
 *   key: 'theme',
 *   value: 'dark',
 * });
 */
export function useSetSetting() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: Setting) => {
            return apiClient.post<Setting>('/api/v1/setting/set', data);
        },
        onSuccess: (data) => {
            logger.log('Setting 设置成功:', data);
            queryClient.invalidateQueries({ queryKey: ['settings', 'list'] });
        },
        onError: (error) => {
            logger.error('Setting 设置失败:', error);
        },
    });
}

/**
 * 数据库导入/导出
 */
export interface DBImportResult {
    rows_affected: Record<string, number>;
}

export interface DBExportOptions {
    include_logs?: boolean;
    include_stats?: boolean;
}

type ApiResponse<T> = {
    code?: number;
    message?: string;
    data?: T;
};

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null;
}

function getMessageField(value: unknown): string | undefined {
    if (!isRecord(value)) return undefined;
    const msg = value.message;
    return typeof msg === 'string' ? msg : undefined;
}

function getDataField<T>(value: unknown): T | undefined {
    if (!isRecord(value)) return undefined;
    return (value as ApiResponse<T>).data;
}

function getAuthHeader(): string {
    const token = useAuthStore.getState().token;
    if (!token) throw new Error('Not authenticated');
    return `Bearer ${token}`;
}

async function downloadFile(taskId: string, filename: string) {
    const res = await fetch(`${API_BASE_URL}/api/v1/setting/export/download?task_id=${taskId}`, {
        headers: { Authorization: getAuthHeader() },
    });
    if (!res.ok) throw new Error('Download failed');
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    try {
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
    } finally {
        URL.revokeObjectURL(url);
    }
}

export interface ExportProgressData {
    table: string;
    current: number;
    total: number;
    rows?: number;
    status?: string;
    error?: string;
}

export type ExportStatus = 'idle' | 'starting' | 'exporting' | 'done' | 'error';

/**
 * 导出数据库（异步任务 + SSE 进度）
 */
export function useExportDB() {
    const [status, setStatus] = useState<ExportStatus>('idle');
    const [progress, setProgress] = useState<ExportProgressData | null>(null);
    const abortRef = useRef<AbortController | null>(null);
    const taskIdRef = useRef<string | null>(null);
    const cancelledRef = useRef(false);

    const startExport = useCallback(async (options: DBExportOptions) => {
        setStatus('starting');
        setProgress(null);
        cancelledRef.current = false;

        let taskId: string;
        try {
            const startRes = await apiClient.post<{ task_id: string }>('/api/v1/setting/export/start', {
                include_logs: !!options.include_logs,
                include_stats: !!options.include_stats,
            });
            taskId = startRes.task_id;
            taskIdRef.current = taskId;
        } catch (e) {
            if (e instanceof DOMException && e.name === 'AbortError') {
                if (cancelledRef.current) throw new Error('__CANCELLED__');
                return;
            }
            throw e;
        }

        setStatus('exporting');
        abortRef.current = new AbortController();

        let sseRes: Response;
        try {
            sseRes = await fetch(`${API_BASE_URL}/api/v1/setting/export/progress?task_id=${taskId}`, {
                headers: { Authorization: getAuthHeader() },
                signal: abortRef.current.signal,
            });
        } catch (e) {
            if (e instanceof DOMException && e.name === 'AbortError') {
                if (cancelledRef.current) throw new Error('__CANCELLED__');
                return;
            }
            throw e;
        }

        if (!sseRes.ok || !sseRes.body) {
            throw new Error('Failed to connect to progress stream');
        }

        const reader = sseRes.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        try {
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });
                const parts = buffer.split('\n\n');
                buffer = parts.pop()!;

                for (const part of parts) {
                    if (!part.startsWith('data: ')) continue;
                    const data: ExportProgressData = JSON.parse(part.slice(6));
                    if (data.status === 'connecting') continue;
                    setProgress(data);

                    if (data.status === 'done') {
                        await downloadFile(taskId, `octopus-export-${new Date().toISOString().replace(/[-:T]/g, '').slice(0, 14)}.json`);
                        setStatus('idle');
                        setProgress(null);
                        return;
                    }
                    if (data.status === 'error') {
                        setStatus('error');
                        throw new Error(data.error || 'Export failed');
                    }
                    if (data.status === 'cancelled') {
                        setStatus('idle');
                        setProgress(null);
                        return;
                    }
                }
            }
        } catch (e) {
            if (e instanceof DOMException && e.name === 'AbortError') {
                if (cancelledRef.current) throw new Error('__CANCELLED__');
                return;
            }
            throw e;
        } finally {
            reader.releaseLock();
        }
    }, []);

    const cancelExport = useCallback(async () => {
        cancelledRef.current = true;
        abortRef.current?.abort();
        const taskId = taskIdRef.current;
        if (taskId) {
            try {
                await apiClient.post('/api/v1/setting/export/cancel', { task_id: taskId });
            } catch {
                // ignore
            }
        }
        setStatus('idle');
        setProgress(null);
        taskIdRef.current = null;
    }, []);

    const reset = useCallback(() => {
        setStatus('idle');
        setProgress(null);
        taskIdRef.current = null;
        abortRef.current?.abort();
    }, []);

    useEffect(() => {
        return () => { abortRef.current?.abort(); };
    }, []);

    return { status, progress, startExport, cancelExport, reset };
}

/**
 * 导入数据库（上传 JSON 文件，增量导入）
 */
export function useImportDB() {
    return useMutation({
        mutationFn: async (file: File) => {
            const form = new FormData();
            form.append('file', file);

            const res = await fetch(`${API_BASE_URL}/api/v1/setting/import`, {
                method: 'POST',
                headers: {
                    Authorization: getAuthHeader(),
                },
                body: form,
            });

            const contentType = res.headers.get('content-type') || '';
            const isJson = contentType.includes('application/json');
            const data = isJson ? await res.json() : await res.text();

            if (!res.ok) {
                const message = getMessageField(data) ?? (typeof data === 'string' ? data : res.statusText);
                throw new Error(message);
            }

            // 支持后端标准 ApiResponse：{code,message,data:{...}}
            const nested = getDataField<DBImportResult>(data);
            return nested ?? (data as DBImportResult);
        },
        onError: (error) => {
            logger.error('导入数据库失败:', error);
        },
    });
}
