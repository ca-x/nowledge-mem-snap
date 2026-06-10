import { Checkbox, Input, Modal } from 'animal-island-ui';

import { retentionLabel } from '../format';
import { useI18n } from '../i18n';
import { Field, FormGrid, ModalFooter, NativeSelect, SwitchField } from '../components/ui';
import type { Config, Editor, Task } from '../types';

export function TaskModal({ editor, cfg, saving, timezoneLabel, onChange, onCancel, onSave }: {
  editor: Editor<Task> | null;
  cfg: Config;
  saving: boolean;
  timezoneLabel: string;
  onChange: (next: Editor<Task> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<Task>) => void;
}) {
  const { t } = useI18n();
  if (!editor) return null;
  const task = editor.value;
  const setTask = (value: Task) => onChange({ ...editor, value });
  const set = (patch: Partial<Task>) => setTask({ ...task, ...patch });
  const setEncryption = (patch: Partial<Task['encryption']>) => set({ encryption: { ...task.encryption, ...patch } });
  const selectedSource = cfg.sources.find((source) => source.key === task.source_key);
  const sourceTip = selectedSource?.type === 'directory' ? t('exportOptionDirectoryTip') : t('exportOptionTaskTip');
  return (
    <Modal
      open
      title={editor.index < 0 ? t('addTaskTitle') : t('editTaskTitle')}
      typewriter={false}
      width={980}
      className="task-modal"
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <div className="editor-form task-editor-form">
        <div className="task-name-row">
          <Field label={t('name')}>
            <Input value={task.name} onChange={(e) => set({ name: e.target.value })} allowClear />
          </Field>
          <SwitchField label={t('enabled')} checked={task.enabled} onChange={(enabled) => set({ enabled })} />
        </div>
        <div className="form-grid task-link-grid">
          <Field label={t('source')}>
            {cfg.sources.length === 0 ? (
              <p className="muted">{t('noSources')}</p>
            ) : (
              <NativeSelect
                value={task.source_key}
                onChange={(sourceKey) => set({ source_key: sourceKey })}
                options={cfg.sources.map((source) => ({ key: source.key, label: source.name }))}
              />
            )}
          </Field>
          <Field label={t('schedule')}>
            {cfg.schedules.length === 0 ? (
              <p className="muted">{t('noSchedules')}</p>
            ) : (
              <NativeSelect
                value={task.schedule_key}
                onChange={(scheduleKey) => set({ schedule_key: scheduleKey })}
                options={cfg.schedules.map((schedule) => ({ key: schedule.key, label: schedule.name }))}
              />
            )}
          </Field>
          <Field label={t('exportOption')} help={sourceTip}>
            {cfg.export_options.length === 0 ? (
              <p className="muted">{t('noExportOptions')}</p>
            ) : (
              <NativeSelect
                value={task.export_option_key}
                onChange={(exportOptionKey) => set({ export_option_key: exportOptionKey })}
                options={cfg.export_options.map((option) => ({ key: option.key, label: option.name }))}
              />
            )}
          </Field>
          <Field label={t('backupStrategy')} help={t('backupStrategyTaskTip').replace('{timezone}', timezoneLabel)}>
            {cfg.backup_strategies.length === 0 ? (
              <p className="muted">{t('noBackupStrategies')}</p>
            ) : (
              <NativeSelect
                value={task.backup_strategy_key}
                onChange={(backupStrategyKey) => set({ backup_strategy_key: backupStrategyKey })}
                options={cfg.backup_strategies.map((strategy) => ({
                  key: strategy.key,
                  label: `${strategy.name} · ${retentionLabel(strategy.retention, t)}`
                }))}
              />
            )}
          </Field>
        </div>
        <div className="form-grid task-path-grid">
          <Field label={t('targets')}>
            {cfg.targets.length === 0 ? (
              <p className="muted">{t('noTargets')}</p>
            ) : (
              <Checkbox
                className="choice-grid target-choice-grid"
                value={task.target_keys}
                onChange={(values) => set({ target_keys: values.map(String) })}
                options={cfg.targets.map((target) => ({ label: target.name, value: target.key }))}
              />
            )}
          </Field>
          <Field label={t('objectPrefix')} help={t('objectPrefixTip')}>
            <Input value={task.object_prefix} onChange={(e) => set({ object_prefix: e.target.value })} allowClear />
          </Field>
        </div>
        <SwitchField label={t('encryptPackage')} checked={task.encryption.enabled} onChange={(enabled) => setEncryption({ enabled })} />
        {task.encryption.enabled && (
          <Field label={t('encryptionPassword')} help={t('secretPreserveHelp')}>
            <Input type="password" value={task.encryption.password ?? ''} onChange={(e) => setEncryption({ password: e.target.value })} allowClear />
          </Field>
        )}
      </div>
    </Modal>
  );
}
