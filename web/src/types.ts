export type SourceType = 'nowledgemem_api' | 'directory';
export type TargetType = 's3' | 'webdav';
export type ScheduleType = 'daily' | 'weekly' | 'once';

export type ExportFlag =
  | 'include_memories'
  | 'include_threads'
  | 'include_messages'
  | 'include_entities'
  | 'include_labels'
  | 'include_sources'
  | 'include_communities'
  | 'include_skills'
  | 'include_edges'
  | 'include_working_memory'
  | 'include_working_memory_archive'
  | 'include_source_files';

export type ExportConfig = Partial<Record<ExportFlag, boolean>>;

export type Source = {
  key: string;
  name: string;
  remark?: string;
  enabled: boolean;
  type: SourceType;
  nowledge_mem?: { api_url: string; api_key?: string; api_key_env?: string };
  directory?: { path: string; root_key?: string };
};

export type S3Config = {
  endpoint_url: string;
  region: string;
  path_style: boolean;
  bucket_name: string;
  root_prefix: string;
  access_key_id: string;
  secret_access_key?: string;
  secret_access_key_env?: string;
};

export type WebDAVConfig = {
  url: string;
  root_prefix: string;
  username: string;
  password?: string;
  password_env?: string;
};

export type Target = {
  key: string;
  name: string;
  enabled: boolean;
  type: TargetType;
  s3?: S3Config;
  webdav?: WebDAVConfig;
};

export type Schedule = {
  key: string;
  name: string;
  enabled: boolean;
  type: ScheduleType;
  time: string;
  weekday?: string;
  run_at?: string;
};

export type Task = {
  key: string;
  name: string;
  enabled: boolean;
  source_key: string;
  schedule_key: string;
  target_keys: string[];
  export_option_key: string;
  backup_strategy_key: string;
  object_prefix: string;
  encryption: { enabled: boolean; password?: string; password_env?: string };
};

export type TaskRuntime = {
  status: 'scheduled' | 'running' | 'disabled' | 'schedule_disabled' | 'missing_schedule' | 'invalid_schedule';
  next_run_at?: string;
};

export type Retention = {
  mode: 'none' | 'keep_last' | 'keep_days' | 'keep_after' | 'keep_before';
  keep_last?: number;
  keep_days?: number;
  keep_after?: string;
  keep_before?: string;
};

export type Config = {
  sources: Source[];
  targets: Target[];
  tasks: Task[];
  schedules: Schedule[];
  export_options: ExportOption[];
  backup_strategies: BackupStrategy[];
  history_limit: number;
  history_retention_days: number;
  runtime?: { timezone?: string; timezone_label?: string };
  task_runtime?: Record<string, TaskRuntime>;
};

export type ExportOption = {
  key: string;
  name: string;
  export: ExportConfig;
};

export type BackupStrategy = {
  key: string;
  name: string;
  retention: Retention;
};

export type Profile = {
  tenant: string;
  username: string;
  display_name: string;
  avatar_url: string;
  is_admin: boolean;
};

export type Run = {
  id: string;
  task_name: string;
  source_key: string;
  status: string;
  object_name: string;
  encrypted: boolean;
  size_bytes: number;
  started_at: string;
  targets: Array<{ target_name: string; status: string; bytes: number; retention_deleted?: number; error?: string }>;
};

export type SourceRoot = {
  key: string;
  name: string;
  path: string;
};

export type VersionInfo = {
  version: string;
  build_time: string;
  git_commit: string;
  full: string;
};

export type TestResult = {
  ok: boolean;
  code?: string;
  message: string;
  details?: Record<string, string>;
};

export type Editor<T> = {
  index: number;
  value: T;
};
