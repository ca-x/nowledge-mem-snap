export type SourceType = 'nowledgemem_api' | 'directory';
export type TargetType = 's3' | 'webdav' | 'gcs' | 'sftp';
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

export type GCSConfig = {
  bucket_name: string;
  root_prefix: string;
  credentials_json?: string;
  credentials_json_env?: string;
};

export type SFTPConfig = {
  host: string;
  port: number;
  root_prefix: string;
  username: string;
  password?: string;
  password_env?: string;
  private_key?: string;
  private_key_env?: string;
  private_key_passphrase?: string;
  private_key_passphrase_env?: string;
  host_key_sha256?: string;
  insecure_ignore_host_key?: boolean;
};

export type Target = {
  key: string;
  name: string;
  enabled: boolean;
  type: TargetType;
  s3?: S3Config;
  webdav?: WebDAVConfig;
  gcs?: GCSConfig;
  sftp?: SFTPConfig;
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
  email?: string;
  display_name: string;
  avatar_url: string;
  is_admin: boolean;
  oidc: {
    linked: boolean;
    issuer?: string;
    email?: string;
  };
};

export type AdminUser = {
  tenant: string;
  username: string;
  email?: string;
  display_name: string;
  avatar_url: string;
  is_admin: boolean;
  oidc: {
    linked: boolean;
    issuer?: string;
    email?: string;
  };
  created_at: string;
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
  targets?: Array<{ target_name: string; status: string; bytes: number; retention_deleted?: number; error?: string }> | null;
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

export type RestoreObject = {
  name: string;
  size_bytes: number;
  modified_at: string;
  encrypted: boolean;
};

export type RestoreDirectory = {
  name: string;
  object_count: number;
  latest_modified_at: string;
};

export type RestoreImportOptions = ExportConfig & {
  mode?: string;
};

export type RestoreState =
  | 'queued'
  | 'downloading'
  | 'decrypting'
  | 'uploading'
  | 'importing'
  | 'completed'
  | 'failed'
  | 'cancelled';

export type RestoreJob = {
  id: string;
  state: RestoreState;
  stage: RestoreState;
  target_key: string;
  object_name: string;
  destination_source_key: string;
  encrypted: boolean;
  size_bytes: number;
  mem_job_id?: string;
  progress: number;
  imported: number;
  skipped: number;
  failed: number;
  message?: string;
  error?: string;
  started_at: string;
  finished_at?: string;
};

export type Editor<T> = {
  index: number;
  value: T;
};
