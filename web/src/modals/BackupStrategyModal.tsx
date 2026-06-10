import { Input, Modal, Radio } from 'animal-island-ui';

import { defaultRetention } from '../configDefaults';
import { useI18n } from '../i18n';
import { Field, ModalFooter } from '../components/ui';
import type { BackupStrategy, Editor, Retention } from '../types';

export function BackupStrategyModal({ editor, saving, timezoneLabel, onChange, onCancel, onSave }: {
  editor: Editor<BackupStrategy> | null;
  saving: boolean;
  timezoneLabel: string;
  onChange: (next: Editor<BackupStrategy> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<BackupStrategy>) => void;
}) {
  const { t } = useI18n();
  if (!editor) return null;
  const strategy = editor.value;
  const setStrategy = (value: BackupStrategy) => onChange({ ...editor, value });
  const set = (patch: Partial<BackupStrategy>) => setStrategy({ ...strategy, ...patch });
  return (
    <Modal
      open
      title={editor.index < 0 ? t('addBackupStrategyTitle') : t('editBackupStrategyTitle')}
      typewriter={false}
      width={760}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <div className="editor-form">
        <Field label={t('name')}>
          <Input value={strategy.name} onChange={(e) => set({ name: e.target.value })} allowClear />
        </Field>
        <RetentionFields
          retention={defaultRetention(strategy.retention)}
          timezoneLabel={timezoneLabel}
          onChange={(patch) => set({ retention: { ...defaultRetention(strategy.retention), ...patch } })}
        />
      </div>
    </Modal>
  );
}

function RetentionFields({ retention, timezoneLabel, onChange }: {
  retention: Retention;
  timezoneLabel: string;
  onChange: (patch: Partial<Retention>) => void;
}) {
  const { t } = useI18n();
  return (
    <div className="retention-box">
      <Field label={t('backupStrategyRule')} help={t('retentionScopeNote').replace('{timezone}', timezoneLabel)}>
        <Radio
          className="choice-grid retention-choice-grid"
          value={retention.mode}
          onChange={(mode) => onChange({ mode: mode as Retention['mode'] })}
          options={[
            { value: 'none', label: t('retentionNone') },
            { value: 'keep_last', label: t('retentionKeepLast') },
            { value: 'keep_days', label: t('retentionKeepDays') },
            { value: 'keep_after', label: t('retentionKeepAfter') },
            { value: 'keep_before', label: t('retentionKeepBefore') }
          ]}
        />
      </Field>
      {retention.mode === 'keep_last' && (
        <Field label={t('backupsToKeep')}>
          <Input type="number" min={1} value={String(retention.keep_last || 7)} onChange={(e) => onChange({ keep_last: Number(e.target.value) || 1 })} />
        </Field>
      )}
      {retention.mode === 'keep_days' && (
        <Field label={t('daysToKeep')}>
          <Input type="number" min={1} value={String(retention.keep_days || 30)} onChange={(e) => onChange({ keep_days: Number(e.target.value) || 1 })} />
        </Field>
      )}
      {retention.mode === 'keep_after' && (
        <Field label={t('keepAfter')}>
          <Input type="date" value={retention.keep_after || ''} onChange={(e) => onChange({ keep_after: e.target.value })} />
        </Field>
      )}
      {retention.mode === 'keep_before' && (
        <Field label={t('keepBefore')}>
          <Input type="date" value={retention.keep_before || ''} onChange={(e) => onChange({ keep_before: e.target.value })} />
        </Field>
      )}
    </div>
  );
}
