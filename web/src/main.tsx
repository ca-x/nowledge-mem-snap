import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Button, Card, Checkbox, Input, Modal, Radio, Select, Switch, Tabs, Title } from 'animal-island-ui';
import 'animal-island-ui/style';
import {
  CheckCircle2,
  DatabaseBackup,
  FolderArchive,
  LogOut,
  Pencil,
  Play,
  Plus,
  ServerCog,
  Settings,
  ShieldCheck,
  ShipWheel,
  Trash2,
  UserRound,
  XCircle
} from 'lucide-react';
import './styles.css';

type SourceType = 'nowledgemem_api' | 'directory';
type TargetType = 's3' | 'webdav';
type ScheduleType = 'daily' | 'weekly';
type ExportFlag =
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

type ExportConfig = Partial<Record<ExportFlag, boolean>>;

type Source = {
  key: string;
  name: string;
  remark?: string;
  enabled: boolean;
  type: SourceType;
  nowledge_mem?: { api_url: string; api_key_env: string };
  directory?: { path: string; root_key?: string };
};

type S3Config = {
  endpoint_url: string;
  region: string;
  path_style: boolean;
  bucket_name: string;
  root_prefix: string;
  access_key_id: string;
  secret_access_key_env: string;
};

type WebDAVConfig = {
  url: string;
  root_prefix: string;
  username: string;
  password_env: string;
};

type Target = {
  key: string;
  name: string;
  enabled: boolean;
  type: TargetType;
  s3?: S3Config;
  webdav?: WebDAVConfig;
};

type Schedule = {
  key: string;
  name: string;
  enabled: boolean;
  type: ScheduleType;
  time: string;
  weekday?: string;
};

type Task = {
  key: string;
  name: string;
  enabled: boolean;
  source_key: string;
  schedule_key: string;
  target_keys: string[];
  object_prefix: string;
  encryption: { enabled: boolean; password_env: string };
  retention: Retention;
  export?: ExportConfig;
};

type Retention = {
  mode: 'none' | 'keep_last' | 'keep_days' | 'keep_after' | 'keep_before';
  keep_last?: number;
  keep_days?: number;
  keep_after?: string;
  keep_before?: string;
};

type Config = {
  export?: ExportConfig;
  sources: Source[];
  targets: Target[];
  tasks: Task[];
  schedules: Schedule[];
  history_limit: number;
  history_retention_days: number;
};

type Profile = {
  tenant: string;
  username: string;
  display_name: string;
  avatar_url: string;
  is_admin: boolean;
};

type Run = {
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

type SourceRoot = {
  key: string;
  name: string;
  path: string;
};

type TestResult = {
  ok: boolean;
  message: string;
  details?: Record<string, string>;
};

type Editor<T> = {
  index: number;
  value: T;
};

const api = async <T,>(path: string, init?: RequestInit): Promise<T> => {
  const res = await fetch(path, {
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
    ...init
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? res.statusText);
  }
  return res.json();
};

function Root() {
  const [setupRequired, setSetupRequired] = useState<boolean | null>(null);
  const path = window.location.pathname;

  useEffect(() => {
    api<{ setup_required: boolean }>('/api/setup/status')
      .then((v) => setSetupRequired(v.setup_required))
      .catch(() => setSetupRequired(false));
  }, []);

  if (setupRequired === null) return <Splash />;
  if (setupRequired) return <SetupPage />;
  if (path === '/login') return <LoginPage />;
  return <Dashboard />;
}

function Splash() {
  return (
    <div className="page center">
      <img src="/logo.png" className="logo xl" alt="Nowledge Mem Snap" />
    </div>
  );
}

function SetupPage() {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const submit = async () => {
    setError('');
    try {
      await api('/api/setup', { method: 'POST', body: JSON.stringify({ username, password }) });
      window.location.href = '/';
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Setup failed');
    }
  };
  return (
    <AuthShell title="Set up your island" subtitle="Create the first administrator.">
      <Field label="Admin username">
        <Input size="large" value={username} onChange={(e) => setUsername(e.target.value)} shadow />
      </Field>
      <Field label="Password">
        <Input size="large" type="password" value={password} onChange={(e) => setPassword(e.target.value)} shadow />
      </Field>
      {error && <p className="error">{error}</p>}
      <Button type="primary" size="large" block onClick={submit}>Create admin</Button>
    </AuthShell>
  );
}

function LoginPage() {
  const [options, setOptions] = useState<{ password: boolean; oidc: boolean; username: string }>({ password: true, oidc: false, username: 'admin' });
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const next = new URLSearchParams(window.location.search).get('next') || '/';

  useEffect(() => {
    api<{ password: boolean; oidc: boolean; username: string }>('/api/auth/options').then((v) => {
      setOptions(v);
      if (v.username) setUsername(v.username);
    });
  }, []);

  const login = async () => {
    setError('');
    try {
      await api('/api/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) });
      window.location.href = next;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    }
  };

  return (
    <AuthShell title="Welcome back" subtitle="Your backup island is waiting.">
      {options.password && (
        <>
          <Field label="Username">
            <Input size="large" value={username} onChange={(e) => setUsername(e.target.value)} shadow />
          </Field>
          <Field label="Password">
            <Input size="large" type="password" value={password} onChange={(e) => setPassword(e.target.value)} shadow />
          </Field>
          {error && <p className="error">{error}</p>}
          <Button type="primary" size="large" block onClick={login}>Log in</Button>
        </>
      )}
      {options.oidc && (
        <Button type="default" size="large" block onClick={() => { window.location.href = `/auth/oidc/start?next=${encodeURIComponent(next)}`; }}>
          Sign in with OIDC
        </Button>
      )}
    </AuthShell>
  );
}

function AuthShell({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <div className="page auth">
      <Card color="app-blue" className="auth-card" pattern="app-yellow">
        <img src="/logo.png" className="logo" alt="Nowledge Mem Snap" />
        <h1>{title}</h1>
        <p>{subtitle}</p>
        <div className="form-stack">{children}</div>
      </Card>
    </div>
  );
}

function Dashboard() {
  const [cfg, setCfg] = useState<Config | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [runs, setRuns] = useState<Run[]>([]);
  const [roots, setRoots] = useState<SourceRoot[]>([]);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [sourceEditor, setSourceEditor] = useState<Editor<Source> | null>(null);
  const [targetEditor, setTargetEditor] = useState<Editor<Target> | null>(null);
  const [scheduleEditor, setScheduleEditor] = useState<Editor<Schedule> | null>(null);
  const [taskEditor, setTaskEditor] = useState<Editor<Task> | null>(null);

  const load = async () => {
    const [configResp, profileResp, runsResp, rootsResp] = await Promise.all([
      api<Config>('/api/config'),
      api<Profile>('/api/profile'),
      api<{ runs: Run[] }>('/api/runs'),
      api<{ roots: SourceRoot[] }>('/api/source-roots')
    ]);
    setCfg(normalizeConfig(configResp));
    setProfile(profileResp);
    setRuns(runsResp.runs);
    setRoots(rootsResp.roots);
  };

  useEffect(() => {
    load().catch(() => {
      window.location.href = `/login?next=${encodeURIComponent(window.location.pathname)}`;
    });
  }, []);

  const persist = async (next: Config, successMessage: string) => {
    setSaving(true);
    setError('');
    setMessage('');
    try {
      const saved = await api<Config>('/api/config', { method: 'PUT', body: JSON.stringify(next) });
      setCfg(normalizeConfig(saved));
      setMessage(successMessage);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const upsertSource = async (editor: Editor<Source>) => {
    if (!cfg) return;
    const sources = [...cfg.sources];
    if (editor.index < 0) sources.push(editor.value);
    else sources[editor.index] = editor.value;
    await persist({ ...cfg, sources }, 'Source saved');
    setSourceEditor(null);
  };

  const upsertTarget = async (editor: Editor<Target>) => {
    if (!cfg) return;
    const targets = [...cfg.targets];
    if (editor.index < 0) targets.push(editor.value);
    else targets[editor.index] = editor.value;
    await persist({ ...cfg, targets }, 'Target saved');
    setTargetEditor(null);
  };

  const upsertSchedule = async (editor: Editor<Schedule>) => {
    if (!cfg) return;
    const schedules = [...cfg.schedules];
    if (editor.index < 0) schedules.push(editor.value);
    else schedules[editor.index] = editor.value;
    await persist({ ...cfg, schedules }, 'Schedule saved');
    setScheduleEditor(null);
  };

  const upsertTask = async (editor: Editor<Task>) => {
    if (!cfg) return;
    const tasks = [...cfg.tasks];
    if (editor.index < 0) tasks.push(editor.value);
    else tasks[editor.index] = editor.value;
    await persist({ ...cfg, tasks }, 'Task saved');
    setTaskEditor(null);
  };

  const saveSettings = async (next: Config) => {
    await persist(next, 'Settings saved');
  };

  const saveProfile = async (next: Profile) => {
    setSaving(true);
    setError('');
    setMessage('');
    try {
      const saved = await api<Profile>('/api/profile', { method: 'PUT', body: JSON.stringify(next) });
      setProfile(saved);
      setMessage('Profile saved');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Profile save failed');
    } finally {
      setSaving(false);
    }
  };

  const removeItem = async (kind: 'sources' | 'targets' | 'schedules' | 'tasks', index: number, successMessage: string) => {
    if (!cfg) return;
    const next = { ...cfg, [kind]: cfg[kind].filter((_, i) => i !== index) };
    await persist(next, successMessage);
  };

  const runTask = async (taskKey: string) => {
    setError('');
    setMessage('Running backup...');
    try {
      await api('/api/backup/run', { method: 'POST', body: JSON.stringify({ task_key: taskKey }) });
      await load();
      setMessage('Backup finished');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Backup failed');
    }
  };

  const logout = async () => {
    await api('/api/auth/logout', { method: 'POST' });
    window.location.href = '/login';
  };

  const summary = useMemo(() => ({
    sources: cfg?.sources.length ?? 0,
    targets: cfg?.targets.length ?? 0,
    tasks: cfg?.tasks.length ?? 0
  }), [cfg]);

  if (!cfg || !profile) return <Splash />;

  return (
    <div className="page dashboard">
      <header className="topbar">
        <div className="brand">
          <img src="/logo.png" className="logo small" alt="" />
          <div>
            <h1>Nowledge Mem Snap</h1>
            <p>Private backup islands for every user</p>
          </div>
        </div>
        <div className="account-box">
          <Avatar profile={profile} />
          <div>
            <strong>{profile.display_name || profile.username}</strong>
            <span>{profile.username}</span>
          </div>
          <Button type="default" icon={<LogOut size={16} />} onClick={logout}>Logout</Button>
        </div>
      </header>

      <section className="stats">
        <Stat icon={<DatabaseBackup />} label="Sources" value={summary.sources} />
        <Stat icon={<ShipWheel />} label="Targets" value={summary.targets} />
        <Stat icon={<FolderArchive />} label="Tasks" value={summary.tasks} />
      </section>

      {(message || error) && (
        <div className={`notice ${error ? 'danger' : 'success'}`}>
          {error ? <XCircle size={18} /> : <CheckCircle2 size={18} />}
          <span>{error || message}</span>
        </div>
      )}

      <Tabs
        defaultActiveKey="tasks"
        items={[
          {
            key: 'tasks',
            label: 'Tasks',
            children: (
              <Tasks
                tasks={cfg.tasks}
                sources={cfg.sources}
                targets={cfg.targets}
                schedules={cfg.schedules}
                onAdd={() => setTaskEditor({ index: -1, value: defaultTask(cfg) })}
                onEdit={(task, index) => setTaskEditor({ index, value: cloneTask(task) })}
                onDelete={(index) => removeItem('tasks', index, 'Task deleted')}
                onRun={runTask}
              />
            )
          },
          {
            key: 'sources',
            label: 'Sources',
            children: (
              <Sources
                sources={cfg.sources}
                roots={roots}
                onAdd={() => setSourceEditor({ index: -1, value: defaultSource(cfg.sources.length, roots) })}
                onEdit={(source, index) => setSourceEditor({ index, value: cloneSource(source) })}
                onDelete={(index) => removeItem('sources', index, 'Source deleted')}
              />
            )
          },
          {
            key: 'targets',
            label: 'Targets',
            children: (
              <Targets
                targets={cfg.targets}
                onAdd={() => setTargetEditor({ index: -1, value: defaultTarget(cfg.targets.length) })}
                onEdit={(target, index) => setTargetEditor({ index, value: cloneTarget(target) })}
                onDelete={(index) => removeItem('targets', index, 'Target deleted')}
              />
            )
          },
          {
            key: 'schedules',
            label: 'Schedules',
            children: (
              <Schedules
                schedules={cfg.schedules}
                onAdd={() => setScheduleEditor({ index: -1, value: defaultSchedule(cfg.schedules.length) })}
                onEdit={(schedule, index) => setScheduleEditor({ index, value: { ...schedule } })}
                onDelete={(index) => removeItem('schedules', index, 'Schedule deleted')}
              />
            )
          },
          { key: 'runs', label: 'Runs', children: <Runs runs={runs} /> },
          {
            key: 'settings',
            label: 'Settings',
            children: <SettingsPanel profile={profile} cfg={cfg} saving={saving} onSaveProfile={saveProfile} onSaveConfig={saveSettings} />
          }
        ]}
      />

      <SourceModal
        editor={sourceEditor}
        roots={roots}
        saving={saving}
        onChange={setSourceEditor}
        onCancel={() => setSourceEditor(null)}
        onSave={upsertSource}
      />
      <TargetModal
        editor={targetEditor}
        saving={saving}
        onChange={setTargetEditor}
        onCancel={() => setTargetEditor(null)}
        onSave={upsertTarget}
      />
      <ScheduleModal
        editor={scheduleEditor}
        saving={saving}
        onChange={setScheduleEditor}
        onCancel={() => setScheduleEditor(null)}
        onSave={upsertSchedule}
      />
      <TaskModal
        editor={taskEditor}
        cfg={cfg}
        saving={saving}
        onChange={setTaskEditor}
        onCancel={() => setTaskEditor(null)}
        onSave={upsertTask}
      />
    </div>
  );
}

function Stat({ icon, label, value }: { icon: React.ReactNode; label: string; value: number }) {
  return <Card color="app-green" className="stat"><div>{icon}</div><span>{label}</span><strong>{value}</strong></Card>;
}

function Avatar({ profile }: { profile: Pick<Profile, 'display_name' | 'username' | 'avatar_url'> }) {
  const label = (profile.display_name || profile.username || 'U').trim();
  if (profile.avatar_url) {
    return <img src={profile.avatar_url} className="avatar" alt={label} />;
  }
  return <div className="avatar fallback">{label.slice(0, 1).toUpperCase()}</div>;
}

function Sources({ sources, roots, onAdd, onEdit, onDelete }: {
  sources: Source[];
  roots: SourceRoot[];
  onAdd: () => void;
  onEdit: (source: Source, index: number) => void;
  onDelete: (index: number) => void;
}) {
  return (
    <Panel title="Sources" actionLabel="Add source" onAdd={onAdd}>
      {sources.length === 0 ? <Empty text="No sources yet." /> : (
        <div className="grid-list">
          {sources.map((source, index) => (
            <Card key={source.key} color="app-blue" className="item">
              <div className="item-head">
                <h3>{source.name}</h3>
                <Switch checked={source.enabled} disabled />
              </div>
              <p>{source.type === 'directory' ? 'Directory source' : 'Nowledge Mem API'}</p>
              <code>{source.type === 'directory' ? source.directory?.path : source.nowledge_mem?.api_url}</code>
              {source.type === 'directory' && roots.length === 0 && <p className="muted">Directory roots are not enabled.</p>}
              {source.remark && <p>{source.remark}</p>}
              <CardActions onEdit={() => onEdit(source, index)} onDelete={() => onDelete(index)} />
            </Card>
          ))}
        </div>
      )}
    </Panel>
  );
}

function Targets({ targets, onAdd, onEdit, onDelete }: {
  targets: Target[];
  onAdd: () => void;
  onEdit: (target: Target, index: number) => void;
  onDelete: (index: number) => void;
}) {
  return (
    <Panel title="Targets" actionLabel="Add target" onAdd={onAdd}>
      {targets.length === 0 ? <Empty text="No targets yet." /> : (
        <div className="grid-list">
          {targets.map((target, index) => (
            <Card key={target.key} color="app-green" className="item">
              <div className="item-head">
                <h3>{target.name}</h3>
                <Switch checked={target.enabled} disabled />
              </div>
              <p>{target.type.toUpperCase()}</p>
              <code>{target.type === 's3' ? target.s3?.bucket_name : target.webdav?.url}</code>
              <CardActions onEdit={() => onEdit(target, index)} onDelete={() => onDelete(index)} />
            </Card>
          ))}
        </div>
      )}
    </Panel>
  );
}

function Schedules({ schedules, onAdd, onEdit, onDelete }: {
  schedules: Schedule[];
  onAdd: () => void;
  onEdit: (schedule: Schedule, index: number) => void;
  onDelete: (index: number) => void;
}) {
  return (
    <Panel title="Schedules" actionLabel="Add schedule" onAdd={onAdd}>
      {schedules.length === 0 ? <Empty text="No schedules yet." /> : (
        <div className="grid-list">
          {schedules.map((schedule, index) => (
            <Card key={schedule.key} color="app-yellow" className="item">
              <div className="item-head">
                <h3>{schedule.name}</h3>
                <Switch checked={schedule.enabled} disabled />
              </div>
              <p>{schedule.type === 'weekly' ? `Weekly ${schedule.weekday}` : 'Daily'} at {schedule.time}</p>
              <code>{schedule.key}</code>
              <CardActions onEdit={() => onEdit(schedule, index)} onDelete={() => onDelete(index)} />
            </Card>
          ))}
        </div>
      )}
    </Panel>
  );
}

function Tasks({ tasks, sources, targets, schedules, onAdd, onEdit, onDelete, onRun }: {
  tasks: Task[];
  sources: Source[];
  targets: Target[];
  schedules: Schedule[];
  onAdd: () => void;
  onEdit: (task: Task, index: number) => void;
  onDelete: (index: number) => void;
  onRun: (key: string) => void;
}) {
  return (
    <Panel title="Tasks" actionLabel="Add task" onAdd={onAdd}>
      {tasks.length === 0 ? <Empty text="No tasks yet." /> : (
        <div className="grid-list">
          {tasks.map((task, index) => (
            <Card key={task.key} color="app-yellow" className="item">
              <div className="item-head">
                <h3>{task.name}</h3>
                <Switch checked={task.enabled} disabled />
              </div>
              <p>Source: {sources.find((s) => s.key === task.source_key)?.name ?? task.source_key}</p>
              <p>Schedule: {schedules.find((s) => s.key === task.schedule_key)?.name ?? task.schedule_key}</p>
              <p>Targets: {task.target_keys.map((k) => targets.find((t) => t.key === k)?.name ?? k).join(', ') || 'None'}</p>
              <p>{task.encryption.enabled ? 'Encrypted package' : 'Plain portable zip'}</p>
              <p>Retention: {retentionLabel(task.retention)}</p>
              <div className="card-actions">
                <Button type="primary" icon={<Play size={16} />} onClick={() => onRun(task.key)}>Run now</Button>
                <Button type="default" icon={<Pencil size={16} />} onClick={() => onEdit(task, index)}>Edit</Button>
                <Button type="default" danger icon={<Trash2 size={16} />} onClick={() => onDelete(index)}>Delete</Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </Panel>
  );
}

function Runs({ runs }: { runs: Run[] }) {
  return (
    <Panel title="Runs">
      <div className="runs">{runs.length === 0 ? <Empty text="No backups yet." /> : runs.map((run) => (
        <Card key={run.id} color="purple" className="run">
          <div className="item-head"><h3>{run.task_name}</h3><span className={`badge ${run.status}`}>{run.status}</span></div>
          <p>{new Date(run.started_at).toLocaleString()} · {formatBytes(run.size_bytes)} · {run.encrypted ? 'encrypted' : 'zip'}</p>
          <code>{run.object_name}</code>
          <div className="target-results">{run.targets.map((target) => <span key={target.target_name}><ShieldCheck size={14} /> {target.target_name}: {target.status}{target.retention_deleted ? ` · deleted ${target.retention_deleted}` : ''}</span>)}</div>
        </Card>
      ))}</div>
    </Panel>
  );
}

function SettingsPanel({ profile, cfg, saving, onSaveProfile, onSaveConfig }: {
  profile: Profile;
  cfg: Config;
  saving: boolean;
  onSaveProfile: (profile: Profile) => void;
  onSaveConfig: (cfg: Config) => void;
}) {
  const [draftProfile, setDraftProfile] = useState(profile);
  const [historyLimit, setHistoryLimit] = useState(String(cfg.history_limit));
  const [historyDays, setHistoryDays] = useState(String(cfg.history_retention_days));
  const [exportDraft, setExportDraft] = useState<ExportConfig>(resolvedGlobalExport(cfg));

  useEffect(() => {
    setDraftProfile(profile);
  }, [profile]);

  useEffect(() => {
    setHistoryLimit(String(cfg.history_limit));
    setHistoryDays(String(cfg.history_retention_days));
    setExportDraft(resolvedGlobalExport(cfg));
  }, [cfg]);

  const uploadAvatar = async (file?: File) => {
    if (!file) return;
    if (!file.type.startsWith('image/')) return;
    const dataURL = await readFileAsDataURL(file);
    setDraftProfile({ ...draftProfile, avatar_url: dataURL });
  };

  return (
    <Panel title="Settings">
      <div className="settings-grid">
        <Card color="app-blue" className="settings-card">
          <div className="settings-title"><UserRound size={20} /> Profile</div>
          <div className="avatar-edit">
            <Avatar profile={draftProfile} />
            <div>
              <input id="avatar-upload" className="file-input" type="file" accept="image/*" onChange={(event) => uploadAvatar(event.target.files?.[0])} />
              <Button type="default" onClick={() => document.getElementById('avatar-upload')?.click()}>Upload image</Button>
            </div>
          </div>
          <div className="editor-form">
            <Field label="Nickname">
              <Input value={draftProfile.display_name} onChange={(e) => setDraftProfile({ ...draftProfile, display_name: e.target.value })} allowClear />
            </Field>
            <Field label="Avatar URL or base64">
              <Input value={draftProfile.avatar_url} onChange={(e) => setDraftProfile({ ...draftProfile, avatar_url: e.target.value })} allowClear />
            </Field>
            <Button type="primary" loading={saving} onClick={() => onSaveProfile(draftProfile)}>Save profile</Button>
          </div>
        </Card>

        <Card color="app-green" className="settings-card">
          <div className="settings-title"><Settings size={20} /> History</div>
          <div className="editor-form">
            <Field label="Keep latest runs">
              <Input type="number" min={1} value={historyLimit} onChange={(e) => setHistoryLimit(e.target.value)} />
            </Field>
            <Field label="Keep run history days">
              <Input type="number" min={1} value={historyDays} onChange={(e) => setHistoryDays(e.target.value)} />
            </Field>
            <Button
              type="primary"
              loading={saving}
              onClick={() => onSaveConfig({
                ...cfg,
                history_limit: Math.max(1, Number(historyLimit) || 100),
                history_retention_days: Math.max(1, Number(historyDays) || 180)
              })}
            >
              Save history settings
            </Button>
          </div>
        </Card>

        <Card color="app-yellow" className="settings-card wide">
          <div className="settings-title"><FolderArchive size={20} /> Nowledge Mem export</div>
          <div className="editor-form">
            <ExportFields
              value={exportDraft}
              overridden
              onChange={setExportDraft}
              onReset={() => setExportDraft(defaultExportConfig())}
            />
            <Button
              type="primary"
              loading={saving}
              onClick={() => onSaveConfig({ ...cfg, export: exportDraft })}
            >
              Save export defaults
            </Button>
          </div>
        </Card>
      </div>
    </Panel>
  );
}

function Panel({ title, actionLabel, onAdd, children }: { title: string; actionLabel?: string; onAdd?: () => void; children: React.ReactNode }) {
  return (
    <section className="panel">
      <div className="panel-head">
        <Title size="small" color="app-teal">{title}</Title>
        {actionLabel && <Button type="primary" icon={<Plus size={16} />} onClick={onAdd}>{actionLabel}</Button>}
      </div>
      {children}
    </section>
  );
}

function CardActions({ onEdit, onDelete }: { onEdit: () => void; onDelete: () => void }) {
  return (
    <div className="card-actions">
      <Button type="default" icon={<Pencil size={16} />} onClick={onEdit}>Edit</Button>
      <Button type="default" danger icon={<Trash2 size={16} />} onClick={onDelete}>Delete</Button>
    </div>
  );
}

function SourceModal({ editor, roots, saving, onChange, onCancel, onSave }: {
  editor: Editor<Source> | null;
  roots: SourceRoot[];
  saving: boolean;
  onChange: (next: Editor<Source> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<Source>) => void;
}) {
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [testing, setTesting] = useState(false);

  useEffect(() => {
    setTestResult(null);
    setTesting(false);
  }, [editor?.index]);

  if (!editor) return null;
  const source = editor.value;
  const setSource = (value: Source) => onChange({ ...editor, value });
  const test = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      const result = await api<TestResult>('/api/sources/test', { method: 'POST', body: JSON.stringify(source) });
      setTestResult(result);
    } catch (err) {
      setTestResult({ ok: false, message: err instanceof Error ? err.message : 'Test failed' });
    } finally {
      setTesting(false);
    }
  };
  return (
    <Modal
      open
      title={editor.index < 0 ? 'Add source' : 'Edit source'}
      typewriter={false}
      width={720}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <SourceForm value={source} roots={roots} onChange={setSource} />
      <div className="test-strip">
        <Button type="default" icon={<ServerCog size={16} />} loading={testing} onClick={test}>Test source</Button>
        {testResult && (
          <span className={`test-result ${testResult.ok ? 'success' : 'danger'}`}>
            {testResult.ok ? <CheckCircle2 size={16} /> : <XCircle size={16} />}
            {testResult.message}
          </span>
        )}
      </div>
    </Modal>
  );
}

function SourceForm({ value, roots, onChange }: { value: Source; roots: SourceRoot[]; onChange: (value: Source) => void }) {
  const nowledge = defaultNowledge(value);
  const directory = defaultDirectory(value, roots);
  const set = (patch: Partial<Source>) => onChange({ ...value, ...patch });
  const setNowledge = (patch: Partial<NonNullable<Source['nowledge_mem']>>) => set({ nowledge_mem: { ...nowledge, ...patch } });
  const setDirectory = (patch: Partial<NonNullable<Source['directory']>>) => set({ directory: { ...directory, ...patch } });

  return (
    <div className="editor-form">
      <FormGrid>
        <Field label="Key">
          <Input value={value.key} onChange={(e) => set({ key: keyify(e.target.value) })} allowClear />
        </Field>
        <Field label="Name">
          <Input value={value.name} onChange={(e) => set({ name: e.target.value })} allowClear />
        </Field>
      </FormGrid>
      <SwitchField label="Enabled" checked={value.enabled} onChange={(enabled) => set({ enabled })} />
      <Field label="Type">
        <Radio
          value={value.type}
          onChange={(next) => set({ type: String(next) as SourceType })}
          options={[
            { label: 'Nowledge Mem API', value: 'nowledgemem_api' },
            { label: 'Directory', value: 'directory' }
          ]}
        />
      </Field>
      {value.type === 'nowledgemem_api' ? (
        <FormGrid>
          <Field label="API URL">
            <Input value={nowledge.api_url} onChange={(e) => setNowledge({ api_url: e.target.value })} allowClear />
          </Field>
          <Field label="API key env">
            <Input value={nowledge.api_key_env} onChange={(e) => setNowledge({ api_key_env: e.target.value })} allowClear />
          </Field>
        </FormGrid>
      ) : (
        <FormGrid>
          <Field label="Allowed root">
            <Select
              value={directory.root_key || roots[0]?.key || ''}
              onChange={(rootKey) => {
                const root = roots.find((item) => item.key === rootKey);
                setDirectory({ root_key: rootKey, path: root?.path ?? directory.path });
              }}
              options={roots.map((root) => ({ key: root.key, label: `${root.name} · ${root.path}` }))}
              placeholder="No allowed roots"
              disabled={roots.length === 0}
            />
          </Field>
          <Field label="Directory path">
            <Input value={directory.path} onChange={(e) => setDirectory({ path: e.target.value })} allowClear />
          </Field>
        </FormGrid>
      )}
      <Field label="Remark">
        <Input value={value.remark ?? ''} onChange={(e) => set({ remark: e.target.value })} allowClear />
      </Field>
    </div>
  );
}

function TargetModal({ editor, saving, onChange, onCancel, onSave }: {
  editor: Editor<Target> | null;
  saving: boolean;
  onChange: (next: Editor<Target> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<Target>) => void;
}) {
  if (!editor) return null;
  const setTarget = (value: Target) => onChange({ ...editor, value });
  return (
    <Modal
      open
      title={editor.index < 0 ? 'Add target' : 'Edit target'}
      typewriter={false}
      width={760}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <TargetForm value={editor.value} onChange={setTarget} />
    </Modal>
  );
}

function TargetForm({ value, onChange }: { value: Target; onChange: (value: Target) => void }) {
  const s3 = defaultS3(value);
  const webdav = defaultWebDAV(value);
  const set = (patch: Partial<Target>) => onChange({ ...value, ...patch });
  const setS3 = (patch: Partial<S3Config>) => set({ s3: { ...s3, ...patch } });
  const setWebDAV = (patch: Partial<WebDAVConfig>) => set({ webdav: { ...webdav, ...patch } });
  return (
    <div className="editor-form">
      <FormGrid>
        <Field label="Key">
          <Input value={value.key} onChange={(e) => set({ key: keyify(e.target.value) })} allowClear />
        </Field>
        <Field label="Name">
          <Input value={value.name} onChange={(e) => set({ name: e.target.value })} allowClear />
        </Field>
      </FormGrid>
      <SwitchField label="Enabled" checked={value.enabled} onChange={(enabled) => set({ enabled })} />
      <Field label="Type">
        <Radio
          value={value.type}
          onChange={(next) => set({ type: String(next) as TargetType })}
          options={[
            { label: 'S3 / R2', value: 's3' },
            { label: 'WebDAV', value: 'webdav' }
          ]}
        />
      </Field>
      {value.type === 's3' ? (
        <>
          <FormGrid>
            <Field label="Endpoint URL">
              <Input value={s3.endpoint_url} onChange={(e) => setS3({ endpoint_url: e.target.value })} allowClear />
            </Field>
            <Field label="Region">
              <Input value={s3.region} onChange={(e) => setS3({ region: e.target.value })} allowClear />
            </Field>
            <Field label="Bucket">
              <Input value={s3.bucket_name} onChange={(e) => setS3({ bucket_name: e.target.value })} allowClear />
            </Field>
            <Field label="Root prefix">
              <Input value={s3.root_prefix} onChange={(e) => setS3({ root_prefix: e.target.value })} allowClear />
            </Field>
            <Field label="Access key id">
              <Input value={s3.access_key_id} onChange={(e) => setS3({ access_key_id: e.target.value })} allowClear />
            </Field>
            <Field label="Secret key env">
              <Input value={s3.secret_access_key_env} onChange={(e) => setS3({ secret_access_key_env: e.target.value })} allowClear />
            </Field>
          </FormGrid>
          <SwitchField label="Path style" checked={s3.path_style} onChange={(pathStyle) => setS3({ path_style: pathStyle })} />
        </>
      ) : (
        <FormGrid>
          <Field label="WebDAV URL">
            <Input value={webdav.url} onChange={(e) => setWebDAV({ url: e.target.value })} allowClear />
          </Field>
          <Field label="Root prefix">
            <Input value={webdav.root_prefix} onChange={(e) => setWebDAV({ root_prefix: e.target.value })} allowClear />
          </Field>
          <Field label="Username">
            <Input value={webdav.username} onChange={(e) => setWebDAV({ username: e.target.value })} allowClear />
          </Field>
          <Field label="Password env">
            <Input value={webdav.password_env} onChange={(e) => setWebDAV({ password_env: e.target.value })} allowClear />
          </Field>
        </FormGrid>
      )}
    </div>
  );
}

function ScheduleModal({ editor, saving, onChange, onCancel, onSave }: {
  editor: Editor<Schedule> | null;
  saving: boolean;
  onChange: (next: Editor<Schedule> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<Schedule>) => void;
}) {
  if (!editor) return null;
  const schedule = editor.value;
  const setSchedule = (value: Schedule) => onChange({ ...editor, value });
  const set = (patch: Partial<Schedule>) => setSchedule({ ...schedule, ...patch });
  return (
    <Modal
      open
      title={editor.index < 0 ? 'Add schedule' : 'Edit schedule'}
      typewriter={false}
      width={640}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <div className="editor-form">
        <FormGrid>
          <Field label="Key">
            <Input value={schedule.key} onChange={(e) => set({ key: keyify(e.target.value) })} allowClear />
          </Field>
          <Field label="Name">
            <Input value={schedule.name} onChange={(e) => set({ name: e.target.value })} allowClear />
          </Field>
        </FormGrid>
        <SwitchField label="Enabled" checked={schedule.enabled} onChange={(enabled) => set({ enabled })} />
        <FormGrid>
          <Field label="Type">
            <Select
              value={schedule.type}
              onChange={(type) => set({ type: type as ScheduleType })}
              options={[
                { key: 'daily', label: 'Daily' },
                { key: 'weekly', label: 'Weekly' }
              ]}
            />
          </Field>
          <Field label="Time">
            <Input type="time" value={schedule.time} onChange={(e) => set({ time: e.target.value })} />
          </Field>
          {schedule.type === 'weekly' && (
            <Field label="Weekday">
              <Select
                value={schedule.weekday || 'sunday'}
                onChange={(weekday) => set({ weekday })}
                options={weekdayOptions}
              />
            </Field>
          )}
        </FormGrid>
      </div>
    </Modal>
  );
}

function TaskModal({ editor, cfg, saving, onChange, onCancel, onSave }: {
  editor: Editor<Task> | null;
  cfg: Config;
  saving: boolean;
  onChange: (next: Editor<Task> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<Task>) => void;
}) {
  if (!editor) return null;
  const task = editor.value;
  const setTask = (value: Task) => onChange({ ...editor, value });
  const set = (patch: Partial<Task>) => setTask({ ...task, ...patch });
  const setEncryption = (patch: Partial<Task['encryption']>) => set({ encryption: { ...task.encryption, ...patch } });
  const setRetention = (patch: Partial<Retention>) => set({ retention: { ...defaultRetention(task.retention), ...patch } });
  const setExport = (value: ExportConfig) => set({ export: value });
  const selectedSource = cfg.sources.find((source) => source.key === task.source_key);
  const isNowledgeMemSource = selectedSource?.type === 'nowledgemem_api';
  return (
    <Modal
      open
      title={editor.index < 0 ? 'Add task' : 'Edit task'}
      typewriter={false}
      width={780}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <div className="editor-form">
        <FormGrid>
          <Field label="Key">
            <Input value={task.key} onChange={(e) => set({ key: keyify(e.target.value) })} allowClear />
          </Field>
          <Field label="Name">
            <Input value={task.name} onChange={(e) => set({ name: e.target.value })} allowClear />
          </Field>
        </FormGrid>
        <SwitchField label="Enabled" checked={task.enabled} onChange={(enabled) => set({ enabled })} />
        <FormGrid>
          <Field label="Source">
            <Select
              value={task.source_key}
              onChange={(sourceKey) => set({ source_key: sourceKey })}
              options={cfg.sources.map((source) => ({ key: source.key, label: source.name }))}
              disabled={cfg.sources.length === 0}
              placeholder="No sources"
            />
          </Field>
          <Field label="Schedule">
            <Select
              value={task.schedule_key}
              onChange={(scheduleKey) => set({ schedule_key: scheduleKey })}
              options={cfg.schedules.map((schedule) => ({ key: schedule.key, label: schedule.name }))}
              disabled={cfg.schedules.length === 0}
              placeholder="No schedules"
            />
          </Field>
        </FormGrid>
        <Field label="Targets">
          {cfg.targets.length === 0 ? (
            <p className="muted">No targets.</p>
          ) : (
            <Checkbox
              value={task.target_keys}
              onChange={(values) => set({ target_keys: values.map(String) })}
              options={cfg.targets.map((target) => ({ label: target.name, value: target.key }))}
              direction="vertical"
            />
          )}
        </Field>
        <Field label="Object prefix">
          <Input value={task.object_prefix} onChange={(e) => set({ object_prefix: e.target.value })} allowClear />
        </Field>
        {isNowledgeMemSource && (
          <ExportFields
            value={resolvedTaskExport(task, cfg)}
            overridden={hasExportOverride(task)}
            onChange={setExport}
            onReset={() => set({ export: {} })}
          />
        )}
        <RetentionFields retention={defaultRetention(task.retention)} onChange={setRetention} />
        <SwitchField label="Encrypt package" checked={task.encryption.enabled} onChange={(enabled) => setEncryption({ enabled })} />
        {task.encryption.enabled && (
          <Field label="Password env">
            <Input value={task.encryption.password_env} onChange={(e) => setEncryption({ password_env: e.target.value })} allowClear />
          </Field>
        )}
      </div>
    </Modal>
  );
}

function RetentionFields({ retention, onChange }: { retention: Retention; onChange: (patch: Partial<Retention>) => void }) {
  return (
    <div className="retention-box">
      <Field label="Remote backup retention">
        <Select
          value={retention.mode}
          onChange={(mode) => onChange({ mode: mode as Retention['mode'] })}
          options={[
            { key: 'none', label: 'Do not clean remote backups' },
            { key: 'keep_last', label: 'Keep latest N backups' },
            { key: 'keep_days', label: 'Keep recent N days' },
            { key: 'keep_after', label: 'Keep backups after date' },
            { key: 'keep_before', label: 'Keep backups before date' }
          ]}
        />
      </Field>
      {retention.mode === 'keep_last' && (
        <Field label="Backups to keep">
          <Input type="number" min={1} value={String(retention.keep_last || 7)} onChange={(e) => onChange({ keep_last: Number(e.target.value) || 1 })} />
        </Field>
      )}
      {retention.mode === 'keep_days' && (
        <Field label="Days to keep">
          <Input type="number" min={1} value={String(retention.keep_days || 30)} onChange={(e) => onChange({ keep_days: Number(e.target.value) || 1 })} />
        </Field>
      )}
      {retention.mode === 'keep_after' && (
        <Field label="Keep after">
          <Input type="date" value={retention.keep_after || ''} onChange={(e) => onChange({ keep_after: e.target.value })} />
        </Field>
      )}
      {retention.mode === 'keep_before' && (
        <Field label="Keep before">
          <Input type="date" value={retention.keep_before || ''} onChange={(e) => onChange({ keep_before: e.target.value })} />
        </Field>
      )}
      <p className="muted">Cleanup is limited to this task's object prefix under each target root prefix.</p>
    </div>
  );
}

function ExportFields({ value, overridden, onChange, onReset }: {
  value: ExportConfig;
  overridden: boolean;
  onChange: (next: ExportConfig) => void;
  onReset: () => void;
}) {
  return (
    <div className="export-box">
      <div className="export-box-head">
        <span>Export contents</span>
        {overridden && <Button type="default" size="small" onClick={onReset}>Use defaults</Button>}
      </div>
      <Checkbox
        value={exportSelectedValues(value)}
        onChange={(values) => onChange(exportConfigFromSelected(values.map(String)))}
        options={exportOptions.map((option) => ({ label: option.label, value: option.key }))}
        direction="vertical"
      />
    </div>
  );
}

function ModalFooter({ saving, onCancel, onSave }: { saving: boolean; onCancel: () => void; onSave: () => void }) {
  return (
    <div className="modal-footer">
      <Button type="default" onClick={onCancel}>Cancel</Button>
      <Button type="primary" loading={saving} onClick={onSave}>Save</Button>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="field">
      <span>{label}</span>
      {children}
    </label>
  );
}

function SwitchField({ label, checked, onChange }: { label: string; checked: boolean; onChange: (checked: boolean) => void }) {
  return (
    <div className="switch-field">
      <span>{label}</span>
      <Switch checked={checked} onChange={onChange} checkedChildren="ON" unCheckedChildren="OFF" />
    </div>
  );
}

function FormGrid({ children }: { children: React.ReactNode }) {
  return <div className="form-grid">{children}</div>;
}

function Empty({ text }: { text: string }) {
  return <Card color="app-yellow" className="empty">{text}</Card>;
}

function normalizeConfig(cfg: Config): Config {
  return {
    export: cfg.export ?? {},
    sources: cfg.sources ?? [],
    targets: cfg.targets ?? [],
    tasks: (cfg.tasks ?? []).map((task) => ({ ...task, retention: defaultRetention(task.retention) })),
    schedules: cfg.schedules ?? [],
    history_limit: cfg.history_limit || 100,
    history_retention_days: cfg.history_retention_days || 180
  };
}

function defaultSource(index: number, roots: SourceRoot[]): Source {
  return {
    key: `source-${index + 1}`,
    name: `Source ${index + 1}`,
    enabled: true,
    type: 'nowledgemem_api',
    nowledge_mem: {
      api_url: 'http://127.0.0.1:14242',
      api_key_env: 'NMEM_API_KEY'
    },
    directory: {
      path: roots[0]?.path ?? '',
      root_key: roots[0]?.key ?? ''
    }
  };
}

function defaultTarget(index: number): Target {
  return {
    key: `target-${index + 1}`,
    name: `Target ${index + 1}`,
    enabled: true,
    type: 's3',
    s3: {
      endpoint_url: '',
      region: 'auto',
      path_style: true,
      bucket_name: '',
      root_prefix: '',
      access_key_id: '',
      secret_access_key_env: `NMEM_SNAP_TARGET_TARGET_${index + 1}_S3_SECRET_ACCESS_KEY`
    },
    webdav: {
      url: '',
      root_prefix: '',
      username: '',
      password_env: `NMEM_SNAP_TARGET_TARGET_${index + 1}_WEBDAV_PASSWORD`
    }
  };
}

function defaultSchedule(index: number): Schedule {
  return {
    key: `schedule-${index + 1}`,
    name: `Schedule ${index + 1}`,
    enabled: true,
    type: 'daily',
    time: '03:00',
    weekday: 'sunday'
  };
}

function defaultTask(cfg: Config): Task {
  const next = cfg.tasks.length + 1;
  return {
    key: `task-${next}`,
    name: `Task ${next}`,
    enabled: true,
    source_key: cfg.sources[0]?.key ?? '',
    schedule_key: cfg.schedules[0]?.key ?? '',
    target_keys: cfg.targets.filter((target) => target.enabled).map((target) => target.key),
    object_prefix: 'nowledge-mem/{task}/{timestamp}',
    encryption: {
      enabled: false,
      password_env: 'NMEM_SNAP_ENCRYPTION_PASSWORD'
    },
    retention: {
      mode: 'none'
    },
    export: {}
  };
}

function defaultExportConfig(): ExportConfig {
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

function resolvedTaskExport(task: Task, cfg: Config): ExportConfig {
  return {
    ...resolvedGlobalExport(cfg),
    ...(task.export ?? {})
  };
}

function resolvedGlobalExport(cfg: Config): ExportConfig {
  return {
    ...defaultExportConfig(),
    ...(cfg.export ?? {})
  };
}

function hasExportOverride(task: Task) {
  return Object.keys(task.export ?? {}).length > 0;
}

function exportSelectedValues(value: ExportConfig) {
  return exportOptions.filter((option) => value[option.key] === true).map((option) => option.key);
}

function exportConfigFromSelected(values: string[]): ExportConfig {
  const selected = new Set(values);
  return Object.fromEntries(exportOptions.map((option) => [option.key, selected.has(option.key)])) as ExportConfig;
}

function defaultNowledge(source: Source) {
  return source.nowledge_mem ?? { api_url: 'http://127.0.0.1:14242', api_key_env: 'NMEM_API_KEY' };
}

function defaultDirectory(source: Source, roots: SourceRoot[]) {
  return source.directory ?? { path: roots[0]?.path ?? '', root_key: roots[0]?.key ?? '' };
}

function defaultS3(target: Target): S3Config {
  return target.s3 ?? {
    endpoint_url: '',
    region: 'auto',
    path_style: true,
    bucket_name: '',
    root_prefix: '',
    access_key_id: '',
    secret_access_key_env: ''
  };
}

function defaultWebDAV(target: Target): WebDAVConfig {
  return target.webdav ?? { url: '', root_prefix: '', username: '', password_env: '' };
}

function cloneSource(source: Source): Source {
  return {
    ...source,
    nowledge_mem: source.nowledge_mem ? { ...source.nowledge_mem } : undefined,
    directory: source.directory ? { ...source.directory } : undefined
  };
}

function cloneTarget(target: Target): Target {
  return {
    ...target,
    s3: target.s3 ? { ...target.s3 } : undefined,
    webdav: target.webdav ? { ...target.webdav } : undefined
  };
}

function cloneTask(task: Task): Task {
  return {
    ...task,
    target_keys: [...task.target_keys],
    encryption: { ...task.encryption },
    retention: { ...defaultRetention(task.retention) },
    export: task.export ? { ...task.export } : {}
  };
}

function defaultRetention(retention?: Retention): Retention {
  return {
    mode: retention?.mode || 'none',
    keep_last: retention?.keep_last || 7,
    keep_days: retention?.keep_days || 30,
    keep_after: retention?.keep_after || '',
    keep_before: retention?.keep_before || ''
  };
}

function retentionLabel(retention?: Retention) {
  const value = defaultRetention(retention);
  switch (value.mode) {
    case 'keep_last':
      return `keep latest ${value.keep_last || 7}`;
    case 'keep_days':
      return `keep ${value.keep_days || 30} days`;
    case 'keep_after':
      return `keep after ${value.keep_after || 'date'}`;
    case 'keep_before':
      return `keep before ${value.keep_before || 'date'}`;
    default:
      return 'no cleanup';
  }
}

function readFileAsDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ''));
    reader.onerror = () => reject(reader.error ?? new Error('failed to read file'));
    reader.readAsDataURL(file);
  });
}

function keyify(value: string) {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9_-]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 64);
}

function formatBytes(value: number) {
  if (!value) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit++;
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

const weekdayOptions = [
  { key: 'sunday', label: 'Sunday' },
  { key: 'monday', label: 'Monday' },
  { key: 'tuesday', label: 'Tuesday' },
  { key: 'wednesday', label: 'Wednesday' },
  { key: 'thursday', label: 'Thursday' },
  { key: 'friday', label: 'Friday' },
  { key: 'saturday', label: 'Saturday' }
];

const exportOptions: Array<{ key: ExportFlag; label: string }> = [
  { key: 'include_memories', label: 'Memories' },
  { key: 'include_threads', label: 'Threads' },
  { key: 'include_messages', label: 'Messages' },
  { key: 'include_entities', label: 'Entities' },
  { key: 'include_labels', label: 'Labels' },
  { key: 'include_sources', label: 'Sources' },
  { key: 'include_communities', label: 'Communities' },
  { key: 'include_skills', label: 'Skills' },
  { key: 'include_edges', label: 'Graph edges' },
  { key: 'include_working_memory', label: 'Working Memory' },
  { key: 'include_working_memory_archive', label: 'Working Memory archive' },
  { key: 'include_source_files', label: 'Original source files' }
];

createRoot(document.getElementById('root')!).render(<Root />);
