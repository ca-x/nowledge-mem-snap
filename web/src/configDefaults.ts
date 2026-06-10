import type { Translate } from './i18n';
import type {
  BackupStrategy,
  Config,
  ExportConfig,
  ExportFlag,
  ExportOption,
  Retention,
  S3Config,
  Schedule,
  Source,
  SourceRoot,
  Target,
  Task,
  WebDAVConfig
} from './types';

export function normalizeConfig(cfg: Config, t: Translate): Config {
  const exportOptions = (cfg.export_options ?? []).length > 0
    ? cfg.export_options.map((option) => ({
      ...option,
      export: { ...defaultExportConfig(), ...(option.export ?? {}) }
    }))
    : [defaultExportOption(0, t)];
  const backupStrategies = (cfg.backup_strategies ?? []).length > 0
    ? cfg.backup_strategies.map((strategy) => ({
      ...strategy,
      retention: defaultRetention(strategy.retention)
    }))
    : [defaultBackupStrategy(0, t)];
  const defaultExportKey = exportOptions[0]?.key ?? '';
  const defaultStrategyKey = backupStrategies[0]?.key ?? '';
  return {
    sources: cfg.sources ?? [],
    targets: cfg.targets ?? [],
    tasks: (cfg.tasks ?? []).map((task) => ({
      ...task,
      target_keys: task.target_keys ?? [],
      export_option_key: task.export_option_key || defaultExportKey,
      backup_strategy_key: task.backup_strategy_key || defaultStrategyKey,
      encryption: task.encryption ?? { enabled: false, password: '' }
    })),
    schedules: cfg.schedules ?? [],
    export_options: exportOptions,
    backup_strategies: backupStrategies,
    history_limit: cfg.history_limit || 100,
    history_retention_days: cfg.history_retention_days || 180,
    runtime: cfg.runtime
  };
}

export function defaultSource(index: number, roots: SourceRoot[], t: Translate): Source {
  const safeRoots = roots ?? [];
  return {
    key: newRecordKey(),
    name: `${t('sourceDefault')} ${index + 1}`,
    enabled: true,
    type: 'nowledgemem_api',
    nowledge_mem: {
      api_url: 'http://127.0.0.1:14242',
      api_key: '',
      api_key_env: 'NMEM_API_KEY'
    },
    directory: {
      path: safeRoots[0]?.path ?? '',
      root_key: safeRoots[0]?.key ?? ''
    }
  };
}

export function defaultTarget(index: number, t: Translate): Target {
  return {
    key: newRecordKey(),
    name: `${t('targetDefault')} ${index + 1}`,
    enabled: true,
    type: 's3',
    s3: {
      endpoint_url: '',
      region: 'auto',
      path_style: true,
      bucket_name: '',
      root_prefix: '',
      access_key_id: '',
      secret_access_key: '',
      secret_access_key_env: `NMEM_SNAP_TARGET_TARGET_${index + 1}_S3_SECRET_ACCESS_KEY`
    },
    webdav: {
      url: '',
      root_prefix: '',
      username: '',
      password: '',
      password_env: `NMEM_SNAP_TARGET_TARGET_${index + 1}_WEBDAV_PASSWORD`
    }
  };
}

export function defaultSchedule(index: number, t: Translate): Schedule {
  return {
    key: newRecordKey(),
    name: `${t('scheduleDefault')} ${index + 1}`,
    enabled: true,
    type: 'daily',
    time: '03:00',
    weekday: 'sunday',
    run_at: defaultRunAt()
  };
}

export function defaultTask(cfg: Config, t: Translate): Task {
  const next = cfg.tasks.length + 1;
  return {
    key: newRecordKey(),
    name: `${t('taskDefault')} ${next}`,
    enabled: true,
    source_key: cfg.sources[0]?.key ?? '',
    schedule_key: cfg.schedules[0]?.key ?? '',
    target_keys: cfg.targets.filter((target) => target.enabled).map((target) => target.key),
    export_option_key: cfg.export_options[0]?.key ?? '',
    backup_strategy_key: cfg.backup_strategies[0]?.key ?? '',
    object_prefix: 'nowledge-mem/{task}/{timestamp}',
    encryption: {
      enabled: false,
      password: '',
      password_env: 'NMEM_SNAP_ENCRYPTION_PASSWORD'
    }
  };
}

export function defaultExportOption(index: number, t: Translate): ExportOption {
  return {
    key: newRecordKey(),
    name: `${t('exportOptionDefault')} ${index + 1}`,
    export: defaultExportConfig()
  };
}

export function defaultBackupStrategy(index: number, t: Translate): BackupStrategy {
  return {
    key: newRecordKey(),
    name: `${t('backupStrategyDefault')} ${index + 1}`,
    retention: {
      mode: 'none'
    }
  };
}

export function defaultExportConfig(): ExportConfig {
  return {
    include_memories: true,
    include_threads: true,
    include_messages: true,
    include_entities: true,
    include_labels: true,
    include_sources: true,
    include_communities: true,
    include_skills: true,
    include_edges: true,
    include_working_memory: true,
    include_working_memory_archive: false,
    include_source_files: false
  };
}

export function exportSelectedValues(value: ExportConfig) {
  return exportFlags.filter((key) => value[key] === true);
}

export function exportConfigFromSelected(values: string[]): ExportConfig {
  const selected = new Set(values);
  return Object.fromEntries(exportFlags.map((key) => [key, selected.has(key)])) as ExportConfig;
}

export function defaultNowledge(source: Source) {
  return source.nowledge_mem ?? { api_url: 'http://127.0.0.1:14242', api_key: '', api_key_env: 'NMEM_API_KEY' };
}

export function defaultDirectory(source: Source, roots: SourceRoot[]) {
  const safeRoots = roots ?? [];
  return source.directory ?? { path: safeRoots[0]?.path ?? '', root_key: safeRoots[0]?.key ?? '' };
}

export function defaultS3(target: Target): S3Config {
  return target.s3 ?? {
    endpoint_url: '',
    region: 'auto',
    path_style: true,
    bucket_name: '',
    root_prefix: '',
    access_key_id: '',
    secret_access_key: '',
    secret_access_key_env: ''
  };
}

export function defaultWebDAV(target: Target): WebDAVConfig {
  return target.webdav ?? { url: '', root_prefix: '', username: '', password: '', password_env: '' };
}

export function cloneSource(source: Source): Source {
  return {
    ...source,
    nowledge_mem: source.nowledge_mem ? { ...source.nowledge_mem } : undefined,
    directory: source.directory ? { ...source.directory } : undefined
  };
}

export function cloneTarget(target: Target): Target {
  return {
    ...target,
    s3: target.s3 ? { ...target.s3 } : undefined,
    webdav: target.webdav ? { ...target.webdav } : undefined
  };
}

export function cloneTask(task: Task): Task {
  return {
    ...task,
    target_keys: [...task.target_keys],
    encryption: { ...task.encryption }
  };
}

export function cloneExportOption(option: ExportOption): ExportOption {
  return {
    ...option,
    export: { ...option.export }
  };
}

export function cloneBackupStrategy(strategy: BackupStrategy): BackupStrategy {
  return {
    ...strategy,
    retention: { ...defaultRetention(strategy.retention) }
  };
}

export function defaultRetention(retention?: Retention): Retention {
  return {
    mode: retention?.mode || 'none',
    keep_last: retention?.keep_last || 7,
    keep_days: retention?.keep_days || 30,
    keep_after: retention?.keep_after || '',
    keep_before: retention?.keep_before || ''
  };
}

export function defaultRunAt() {
  const next = new Date(Date.now() + 60 * 60 * 1000);
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${next.getFullYear()}-${pad(next.getMonth() + 1)}-${pad(next.getDate())}T${pad(next.getHours())}:${pad(next.getMinutes())}`;
}

export function weekdayOptions(t: Translate) {
  return ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'].map((key) => ({
    key,
    label: t(`weekday.${key}`)
  }));
}

function newRecordKey() {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID();
  }
  return `id-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

export const exportFlags: ExportFlag[] = [
  'include_memories',
  'include_threads',
  'include_messages',
  'include_entities',
  'include_labels',
  'include_sources',
  'include_communities',
  'include_skills',
  'include_edges',
  'include_working_memory',
  'include_working_memory_archive',
  'include_source_files'
];
