import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Button, Card, Cursor, Input, Tabs, Tooltip } from 'animal-island-ui';
import 'animal-island-ui/style';
import {
  ArchiveRestore,
  CalendarClock,
  ChevronDown,
  CheckCircle2,
  DatabaseBackup,
  Ellipsis,
  FileDown,
  FolderArchive,
  History,
  Languages,
  LogOut,
  Recycle,
  Settings,
  ShipWheel,
  UserCog,
  UserRound,
  X,
  XCircle
} from 'lucide-react';
import {
  I18nContext,
  initialLang,
  localeForLang,
  makeTranslator,
  persistLang,
  useI18n
} from './i18n';
import type { Lang } from './i18n';
import { api } from './api';
import { Avatar, Field } from './components/ui';
import {
  cloneBackupStrategy,
  cloneExportOption,
  cloneSource,
  cloneTarget,
  cloneTask,
  defaultBackupStrategy,
  defaultExportOption,
  defaultSchedule,
  defaultSource,
  defaultTarget,
  defaultTask,
  normalizeConfig
} from './configDefaults';
import { BackupStrategyModal } from './modals/BackupStrategyModal';
import { ExportOptionModal } from './modals/ExportOptionModal';
import { ProfileModal } from './modals/ProfileModal';
import { ScheduleModal } from './modals/ScheduleModal';
import { SourceModal } from './modals/SourceModal';
import { TargetModal } from './modals/TargetModal';
import { TaskModal } from './modals/TaskModal';
import { BackupStrategiesPage } from './pages/BackupStrategiesPage';
import { ExportOptionsPage } from './pages/ExportOptionsPage';
import { RunsPage } from './pages/RunsPage';
import { SchedulesPage } from './pages/SchedulesPage';
import { SettingsPage } from './pages/SettingsPage';
import { SiteAdminPage } from './pages/SiteAdminPage';
import { SourcesPage } from './pages/SourcesPage';
import { TargetsPage } from './pages/TargetsPage';
import { TasksPage } from './pages/TasksPage';
import { RestorePage } from './pages/RestorePage';
import { appPath, assetPath, currentAppPath, nextFromSearch, routePath } from './paths';
import type {
  BackupStrategy,
  Config,
  Editor,
  ExportOption,
  Profile,
  Run,
  Schedule,
  Source,
  SourceRoot,
  Task,
  Target,
  VersionInfo
} from './types';
import './styles.css';

function Root() {
  const [setupRequired, setSetupRequired] = useState<boolean | null>(null);
  const [lang, setLangState] = useState<Lang>(initialLang);
  const path = routePath();
  const t = useMemo(() => makeTranslator(lang), [lang]);
  const setLang = (next: Lang) => {
    setLangState(next);
    persistLang(next);
  };

  useEffect(() => {
    api<{ setup_required: boolean }>('/api/setup/status')
      .then((v) => setSetupRequired(v.setup_required))
      .catch(() => setSetupRequired(false));
  }, []);

  return (
    <I18nContext.Provider value={{ lang, setLang, t }}>
      <Cursor>
        {setupRequired === null ? <Splash /> : setupRequired ? <SetupPage /> : path === '/login' ? <LoginPage /> : <Dashboard />}
      </Cursor>
    </I18nContext.Provider>
  );
}

function Splash() {
  return (
    <div className="page center">
      <img src={assetPath('/logo.png')} className="logo xl" alt="Nowledge Mem Snap" />
    </div>
  );
}

function SetupPage() {
  const { t } = useI18n();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const submit = async () => {
    setError('');
    try {
      await api('/api/setup', { method: 'POST', body: JSON.stringify({ username, password }) });
      window.location.href = appPath('/');
    } catch (err) {
      setError(err instanceof Error ? err.message : t('setupFailed'));
    }
  };
  return (
    <AuthShell title={t('setupTitle')} subtitle={t('setupSubtitle')}>
      <Field label={t('adminUsername')}>
        <Input size="large" value={username} placeholder={t('adminUsernamePlaceholder')} onChange={(e) => setUsername(e.target.value)} shadow />
      </Field>
      <Field label={t('password')}>
        <Input size="large" type="password" value={password} onChange={(e) => setPassword(e.target.value)} shadow />
      </Field>
      {error && <p className="error">{error}</p>}
      <Button type="primary" size="large" block onClick={submit}>{t('createAdmin')}</Button>
    </AuthShell>
  );
}

function LoginPage() {
  const { t } = useI18n();
  const [options, setOptions] = useState<{ password: boolean; oidc: boolean; username: string }>({ password: true, oidc: false, username: '' });
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const next = nextFromSearch();

  useEffect(() => {
    api<{ password: boolean; oidc: boolean; username: string }>('/api/auth/options').then((v) => {
      setOptions(v);
    });
  }, []);

  const login = async () => {
    setError('');
    try {
      await api('/api/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) });
      window.location.href = appPath(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('loginFailed'));
    }
  };

  return (
    <AuthShell title={t('loginTitle')} subtitle={t('loginSubtitle')}>
      {options.password && (
        <>
          <Field label={t('username')}>
            <Input size="large" value={username} placeholder={t('usernamePlaceholder')} onChange={(e) => setUsername(e.target.value)} shadow />
          </Field>
          <Field label={t('password')}>
            <Input size="large" type="password" value={password} onChange={(e) => setPassword(e.target.value)} shadow />
          </Field>
          {error && <p className="error">{error}</p>}
          <Button type="primary" size="large" block onClick={login}>{t('logIn')}</Button>
        </>
      )}
      {options.oidc && (
        <Button type="default" size="large" block onClick={() => { window.location.href = appPath(`/auth/oidc/start?next=${encodeURIComponent(next)}`); }}>
          {t('signInOidc')}
        </Button>
      )}
    </AuthShell>
  );
}

function AuthShell({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <div className="page auth">
      <Card color="app-blue" className="auth-card" pattern="app-yellow">
        <div className="auth-toolbar"><LanguageSwitch /></div>
        <img src={assetPath('/logo.png')} className="logo" alt="Nowledge Mem Snap" />
        <h1>{title}</h1>
        <p>{subtitle}</p>
        <div className="form-stack">{children}</div>
      </Card>
    </div>
  );
}

function LanguageSwitch() {
  const { lang, setLang, t } = useI18n();
  const next = lang === 'zh' ? 'en' : 'zh';
  return (
    <Button type="default" size="small" icon={<Languages size={14} />} onClick={() => setLang(next)} aria-label={t('language')}>
      {next === 'zh' ? '中' : 'EN'}
    </Button>
  );
}

function Dashboard() {
  const { lang, t } = useI18n();
  const [cfg, setCfg] = useState<Config | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [runs, setRuns] = useState<Run[]>([]);
  const [roots, setRoots] = useState<SourceRoot[]>([]);
  const [versionInfo, setVersionInfo] = useState<VersionInfo | null>(null);
  const [authOptions, setAuthOptions] = useState<{ oidc: boolean }>({ oidc: false });
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [sourceEditor, setSourceEditor] = useState<Editor<Source> | null>(null);
  const [targetEditor, setTargetEditor] = useState<Editor<Target> | null>(null);
  const [scheduleEditor, setScheduleEditor] = useState<Editor<Schedule> | null>(null);
  const [exportOptionEditor, setExportOptionEditor] = useState<Editor<ExportOption> | null>(null);
  const [backupStrategyEditor, setBackupStrategyEditor] = useState<Editor<BackupStrategy> | null>(null);
  const [taskEditor, setTaskEditor] = useState<Editor<Task> | null>(null);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [profileModalOpen, setProfileModalOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('sources');

  const load = async () => {
    const [configResp, profileResp, runsResp, rootsResp, versionResp, authOptionsResp] = await Promise.all([
      api<Config>('/api/config'),
      api<Profile>('/api/profile'),
      api<{ runs: Run[] | null }>('/api/runs'),
      api<{ roots: SourceRoot[] | null }>('/api/source-roots'),
      api<VersionInfo>('/api/version'),
      api<{ oidc: boolean }>('/api/auth/options')
    ]);
    setCfg(normalizeConfig(configResp, t));
    setProfile(profileResp);
    setRuns(runsResp.runs ?? []);
    setRoots(rootsResp.roots ?? []);
    setVersionInfo(versionResp);
    setAuthOptions({ oidc: Boolean(authOptionsResp.oidc) });
  };

  useEffect(() => {
    load().catch(() => {
      window.location.href = appPath(`/login?next=${encodeURIComponent(currentAppPath())}`);
    });
  }, []);

  useEffect(() => {
    if (!message) return;
    const timer = window.setTimeout(() => setMessage(''), 4000);
    return () => window.clearTimeout(timer);
  }, [message]);

  const persist = async (next: Config, successMessage: string) => {
    setSaving(true);
    setError('');
    setMessage('');
    try {
      const saved = await api<Config>('/api/config', { method: 'PUT', body: JSON.stringify(next) });
      setCfg(normalizeConfig(saved, t));
      setMessage(successMessage);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const upsertSource = async (editor: Editor<Source>) => {
    if (!cfg) return;
    const sources = [...cfg.sources];
    if (editor.index < 0) sources.push(editor.value);
    else sources[editor.index] = editor.value;
    await persist({ ...cfg, sources }, t('sourceSaved'));
    setSourceEditor(null);
  };

  const upsertTarget = async (editor: Editor<Target>) => {
    if (!cfg) return;
    const targets = [...cfg.targets];
    if (editor.index < 0) targets.push(editor.value);
    else targets[editor.index] = editor.value;
    await persist({ ...cfg, targets }, t('targetSaved'));
    setTargetEditor(null);
  };

  const upsertSchedule = async (editor: Editor<Schedule>) => {
    if (!cfg) return;
    const schedules = [...cfg.schedules];
    if (editor.index < 0) schedules.push(editor.value);
    else schedules[editor.index] = editor.value;
    await persist({ ...cfg, schedules }, t('scheduleSaved'));
    setScheduleEditor(null);
  };

  const upsertExportOption = async (editor: Editor<ExportOption>) => {
    if (!cfg) return;
    const export_options = [...cfg.export_options];
    if (editor.index < 0) export_options.push(editor.value);
    else export_options[editor.index] = editor.value;
    await persist({ ...cfg, export_options }, t('exportOptionSaved'));
    setExportOptionEditor(null);
  };

  const upsertBackupStrategy = async (editor: Editor<BackupStrategy>) => {
    if (!cfg) return;
    const backup_strategies = [...cfg.backup_strategies];
    if (editor.index < 0) backup_strategies.push(editor.value);
    else backup_strategies[editor.index] = editor.value;
    await persist({ ...cfg, backup_strategies }, t('backupStrategySaved'));
    setBackupStrategyEditor(null);
  };

  const upsertTask = async (editor: Editor<Task>) => {
    if (!cfg) return;
    const tasks = [...cfg.tasks];
    if (editor.index < 0) tasks.push(editor.value);
    else tasks[editor.index] = editor.value;
    await persist({ ...cfg, tasks }, t('taskSaved'));
    setTaskEditor(null);
  };

  const saveSettings = async (next: Config) => {
    await persist(next, t('settingsSaved'));
  };

  const saveProfile = async (next: Profile) => {
    setSaving(true);
    setError('');
    setMessage('');
    try {
      const saved = await api<Profile>('/api/profile', { method: 'PUT', body: JSON.stringify(next) });
      setProfile(saved);
      setMessage(t('profileSaved'));
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : t('profileSaveFailed'));
      return false;
    } finally {
      setSaving(false);
    }
  };

  const linkOIDC = () => {
    window.location.href = appPath(`/auth/oidc/start?mode=link&next=${encodeURIComponent(currentAppPath())}`);
  };

  const removeItem = async (kind: 'sources' | 'targets' | 'schedules' | 'export_options' | 'backup_strategies' | 'tasks', index: number, successMessage: string) => {
    if (!cfg) return;
    const next = { ...cfg, [kind]: cfg[kind].filter((_, i) => i !== index) };
    await persist(next, successMessage);
  };

  const runTask = async (taskKey: string) => {
    setError('');
    setMessage(t('runningBackup'));
    try {
      await api('/api/backup/run', { method: 'POST', body: JSON.stringify({ task_key: taskKey }) });
      await load();
      setMessage(t('backupFinished'));
    } catch (err) {
      setError(err instanceof Error ? err.message : t('backupFailed'));
    }
  };

  const logout = async () => {
    await api('/api/auth/logout', { method: 'POST' });
    window.location.href = appPath('/login');
  };

  const summary = useMemo(() => ({
    sources: cfg?.sources.length ?? 0,
    targets: cfg?.targets.length ?? 0,
    tasks: cfg?.tasks.length ?? 0
  }), [cfg]);

  if (!cfg || !profile) return <Splash />;

  const dashboardTabs = [
    {
      key: 'sources',
      label: t('sources'),
      children: (
        <SourcesPage
          sources={cfg.sources}
          roots={roots}
          onAdd={() => setSourceEditor({ index: -1, value: defaultSource(cfg.sources.length, roots, t) })}
          onEdit={(source, index) => setSourceEditor({ index, value: cloneSource(source) })}
          onDelete={(index) => removeItem('sources', index, t('sourceDeleted'))}
        />
      )
    },
    {
      key: 'targets',
      label: t('targets'),
      children: (
        <TargetsPage
          targets={cfg.targets}
          onAdd={() => setTargetEditor({ index: -1, value: defaultTarget(cfg.targets.length, t) })}
          onEdit={(target, index) => setTargetEditor({ index, value: cloneTarget(target) })}
          onDelete={(index) => removeItem('targets', index, t('targetDeleted'))}
        />
      )
    },
    {
      key: 'schedules',
      label: t('schedules'),
      children: (
        <SchedulesPage
          schedules={cfg.schedules}
          onAdd={() => setScheduleEditor({ index: -1, value: defaultSchedule(cfg.schedules.length, t) })}
          onEdit={(schedule, index) => setScheduleEditor({ index, value: { ...schedule } })}
          onDelete={(index) => removeItem('schedules', index, t('scheduleDeleted'))}
        />
      )
    },
    {
      key: 'export-options',
      label: t('exportOptions'),
      children: (
        <ExportOptionsPage
          options={cfg.export_options}
          onAdd={() => setExportOptionEditor({ index: -1, value: defaultExportOption(cfg.export_options.length, t) })}
          onEdit={(option, index) => setExportOptionEditor({ index, value: cloneExportOption(option) })}
          onDelete={(index) => removeItem('export_options', index, t('exportOptionDeleted'))}
        />
      )
    },
    {
      key: 'backup-strategies',
      label: t('backupStrategies'),
      children: (
        <BackupStrategiesPage
          strategies={cfg.backup_strategies}
          onAdd={() => setBackupStrategyEditor({ index: -1, value: defaultBackupStrategy(cfg.backup_strategies.length, t) })}
          onEdit={(strategy, index) => setBackupStrategyEditor({ index, value: cloneBackupStrategy(strategy) })}
          onDelete={(index) => removeItem('backup_strategies', index, t('backupStrategyDeleted'))}
        />
      )
    },
    {
      key: 'tasks',
      label: t('tasks'),
      children: (
        <TasksPage
          tasks={cfg.tasks}
          sources={cfg.sources}
          targets={cfg.targets}
          schedules={cfg.schedules}
          exportOptions={cfg.export_options}
          backupStrategies={cfg.backup_strategies}
          taskRuntime={cfg.task_runtime ?? {}}
          locale={localeForLang(lang)}
          onAdd={() => setTaskEditor({ index: -1, value: defaultTask(cfg, t) })}
          onEdit={(task, index) => setTaskEditor({ index, value: cloneTask(task) })}
          onDelete={(index) => removeItem('tasks', index, t('taskDeleted'))}
          onRun={runTask}
        />
      )
    },
    { key: 'runs', label: t('runs'), children: <RunsPage runs={runs} locale={localeForLang(lang)} /> },
    {
      key: 'restore',
      label: t('restore'),
      children: <RestorePage targets={cfg.targets} sources={cfg.sources} locale={localeForLang(lang)} />
    },
    {
      key: 'settings',
      label: t('settings'),
      children: <SettingsPage cfg={cfg} saving={saving} onSaveConfig={saveSettings} />
    }
  ];
  if (profile.is_admin) {
    dashboardTabs.push({
      key: 'site',
      label: t('siteManagement'),
      children: <SiteAdminPage currentTenant={profile.tenant} locale={localeForLang(lang)} onCurrentUserRenamed={logout} />
    });
  }

  return (
    <div className="page dashboard">
      <header className="topbar">
        <div className="brand">
          <img src={assetPath('/logo.png')} className="logo small" alt="" />
          <div>
            <h1>Nowledge Mem Snap</h1>
            <p>{t('dashboardSubtitle')}</p>
          </div>
        </div>
        <div className="topbar-actions">
          <LanguageSwitch />
          <div className="user-menu">
            <button className="user-menu-trigger" type="button" onClick={() => setUserMenuOpen((open) => !open)} aria-expanded={userMenuOpen}>
              <Avatar profile={profile} />
              <span className="user-menu-name">
                <strong>{profile.display_name || profile.username}</strong>
                <span>{profile.username}</span>
              </span>
              <ChevronDown size={16} />
            </button>
            {userMenuOpen && (
              <div className="user-menu-popover">
                <Button type="default" icon={<UserRound size={16} />} onClick={() => { setProfileModalOpen(true); setUserMenuOpen(false); }}>{t('profile')}</Button>
                <Button type="default" icon={<LogOut size={16} />} onClick={logout}>{t('logout')}</Button>
              </div>
            )}
          </div>
        </div>
      </header>

      <section className="stats">
        <Stat icon={<DatabaseBackup />} label={t('sources')} value={summary.sources} onClick={() => setActiveTab('sources')} />
        <Stat icon={<ShipWheel />} label={t('targets')} value={summary.targets} onClick={() => setActiveTab('targets')} />
        <Stat icon={<FolderArchive />} label={t('tasks')} value={summary.tasks} onClick={() => setActiveTab('tasks')} />
      </section>

      {(message || error) && (
        <div className={`notice ${error ? 'danger' : 'success'}`} role={error ? 'alert' : 'status'}>
          {error ? <XCircle size={18} /> : <CheckCircle2 size={18} />}
          <span>{error || message}</span>
        </div>
      )}

      <Tabs className="dashboard-tabs" activeKey={activeTab} onChange={setActiveTab} items={dashboardTabs} leafAnimation={false} />

      <MobileNav
        items={dashboardTabs.map(({ key, label }) => ({ key, label }))}
        activeKey={activeTab}
        onSelect={setActiveTab}
      />

      <AppFooter versionInfo={versionInfo} />

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
        timezoneLabel={cfg.runtime?.timezone_label || cfg.runtime?.timezone || t('serverTimezone')}
        onChange={setScheduleEditor}
        onCancel={() => setScheduleEditor(null)}
        onSave={upsertSchedule}
      />
      <ExportOptionModal
        editor={exportOptionEditor}
        saving={saving}
        onChange={setExportOptionEditor}
        onCancel={() => setExportOptionEditor(null)}
        onSave={upsertExportOption}
      />
      <BackupStrategyModal
        editor={backupStrategyEditor}
        saving={saving}
        timezoneLabel={cfg.runtime?.timezone_label || cfg.runtime?.timezone || t('serverTimezone')}
        onChange={setBackupStrategyEditor}
        onCancel={() => setBackupStrategyEditor(null)}
        onSave={upsertBackupStrategy}
      />
      <TaskModal
        editor={taskEditor}
        cfg={cfg}
        saving={saving}
        timezoneLabel={cfg.runtime?.timezone_label || cfg.runtime?.timezone || t('serverTimezone')}
        onChange={setTaskEditor}
        onCancel={() => setTaskEditor(null)}
        onSave={upsertTask}
      />
      <ProfileModal
        open={profileModalOpen}
        profile={profile}
        saving={saving}
        onCancel={() => setProfileModalOpen(false)}
        onSave={saveProfile}
        oidcEnabled={authOptions.oidc}
        onLinkOIDC={linkOIDC}
      />
    </div>
  );
}

function Stat({ icon, label, value, onClick }: { icon: React.ReactNode; label: string; value: number; onClick?: () => void }) {
  const card = <Card color="app-green" pattern="app-green" className="stat"><div>{icon}</div><span>{label}</span><strong>{value}</strong></Card>;
  if (!onClick) return card;
  return <button type="button" className="stat-link" onClick={onClick}>{card}</button>;
}

const sectionIcons: Record<string, React.ReactNode> = {
  sources: <DatabaseBackup />,
  targets: <ShipWheel />,
  schedules: <CalendarClock />,
  'export-options': <FileDown />,
  'backup-strategies': <Recycle />,
  tasks: <FolderArchive />,
  runs: <History />,
  restore: <ArchiveRestore />,
  settings: <Settings />,
  site: <UserCog />
};

const primarySections = ['sources', 'tasks', 'runs', 'restore'];

function MobileNav({ items, activeKey, onSelect }: {
  items: Array<{ key: string; label: React.ReactNode }>;
  activeKey: string;
  onSelect: (key: string) => void;
}) {
  const { t } = useI18n();
  const [sheetOpen, setSheetOpen] = useState(false);
  const primary = primarySections
    .map((key) => items.find((item) => item.key === key))
    .filter((item): item is { key: string; label: React.ReactNode } => Boolean(item));
  const moreActive = !primarySections.includes(activeKey);

  useEffect(() => {
    if (!sheetOpen) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setSheetOpen(false);
    };
    window.addEventListener('keydown', onKey);
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      window.removeEventListener('keydown', onKey);
      document.body.style.overflow = previousOverflow;
    };
  }, [sheetOpen]);

  const select = (key: string) => {
    onSelect(key);
    setSheetOpen(false);
    window.scrollTo({ top: 0 });
  };

  return (
    <>
      <nav className="bottom-nav" aria-label={t('sectionNavigation')}>
        {primary.map((item) => (
          <button
            key={item.key}
            type="button"
            className={`bottom-nav-item ${activeKey === item.key ? 'active' : ''}`}
            aria-current={activeKey === item.key ? 'page' : undefined}
            onClick={() => select(item.key)}
          >
            <span className="bottom-nav-icon">{sectionIcons[item.key]}</span>
            <span className="bottom-nav-label">{item.label}</span>
          </button>
        ))}
        <button
          type="button"
          className={`bottom-nav-item ${moreActive ? 'active' : ''}`}
          aria-expanded={sheetOpen}
          onClick={() => setSheetOpen((open) => !open)}
        >
          <span className="bottom-nav-icon"><Ellipsis /></span>
          <span className="bottom-nav-label">{t('more')}</span>
        </button>
      </nav>
      {sheetOpen && (
        <div className="sheet-scrim" onClick={() => setSheetOpen(false)}>
          <div
            className="section-sheet"
            role="dialog"
            aria-modal="true"
            aria-label={t('sectionNavigation')}
            onClick={(event) => event.stopPropagation()}
          >
            <div className="sheet-handle" aria-hidden="true" />
            <header className="sheet-head">
              <strong>{t('sectionNavigation')}</strong>
              <button type="button" className="sheet-close" aria-label={t('close')} onClick={() => setSheetOpen(false)}>
                <X size={18} />
              </button>
            </header>
            <div className="sheet-grid">
              {items.map((item) => (
                <button
                  key={item.key}
                  type="button"
                  className={`sheet-item ${activeKey === item.key ? 'active' : ''}`}
                  aria-current={activeKey === item.key ? 'page' : undefined}
                  onClick={() => select(item.key)}
                >
                  {sectionIcons[item.key]}
                  <span>{item.label}</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function AppFooter({ versionInfo }: { versionInfo: VersionInfo | null }) {
  const { t } = useI18n();
  const version = versionInfo?.version || 'dev';
  const versionLabel = version === 'dev' || version.startsWith('v') ? version : `v${version}`;
  const commit = versionInfo?.git_commit || 'unknown';
  const buildTime = versionInfo?.build_time || 'unknown';
  const full = versionInfo?.full || version;

  return (
    <footer className="app-footer" aria-label={t('versionInfo')}>
      <span>nowledge-mem snap</span>
      <Tooltip
        placement="top"
        title={(
          <div className="version-tooltip">
            <div>{full}</div>
            <div>{t('buildTime')}: {buildTime}</div>
            <div>{t('gitCommit')}: {commit}</div>
          </div>
        )}
      >
        <span className="footer-pill" tabIndex={0}>{versionLabel}</span>
      </Tooltip>
      <span>© 2026</span>
    </footer>
  );
}

createRoot(document.getElementById('root')!).render(<Root />);
