import { Button, Modal } from 'animal-island-ui';
import { Play } from 'lucide-react';

import { Tip } from '../components/ui';
import { useI18n } from '../i18n';
import type { Target, Task } from '../types';

export type RunTaskEditor = {
  task: Task;
  selectedTargetKeys: string[];
};

export function defaultRunTargetKeys(task: Task, targets: Target[]) {
  const enabledKeys = new Set(targets.filter((target) => target.enabled).map((target) => target.key));
  const seen = new Set<string>();
  return task.target_keys.filter((key) => {
    if (!enabledKeys.has(key) || seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

export function RunTaskModal({ editor, targets, running, onChange, onCancel, onRun }: {
  editor: RunTaskEditor | null;
  targets: Target[];
  running: boolean;
  onChange: (next: RunTaskEditor | null) => void;
  onCancel: () => void;
  onRun: (taskKey: string, targetKeys: string[]) => void;
}) {
  const { t } = useI18n();
  if (!editor) return null;

  const task = editor.task;
  const targetsByKey = new Map(targets.map((target) => [target.key, target]));
  const options = task.target_keys.map((key) => ({ key, target: targetsByKey.get(key) }));
  const runnableKeys = new Set(options.filter((option) => option.target?.enabled).map((option) => option.key));
  const selectedKeys = editor.selectedTargetKeys.filter((key) => runnableKeys.has(key));
  const canRun = selectedKeys.length > 0;

  const toggleTarget = (key: string, checked: boolean) => {
    const selected = new Set(editor.selectedTargetKeys);
    if (checked) selected.add(key);
    else selected.delete(key);
    onChange({ ...editor, selectedTargetKeys: [...selected] });
  };

  const submit = () => {
    if (!canRun) return;
    onRun(task.key, selectedKeys);
  };

  return (
    <Modal
      open
      title={t('runBackupTitle')}
      typewriter={false}
      width={640}
      className="run-target-modal"
      onClose={onCancel}
      footer={(
        <div className="modal-footer">
          <Button type="default" onClick={onCancel}>{t('cancel')}</Button>
          <Button type="primary" icon={<Play size={16} />} loading={running} disabled={!canRun} onClick={submit}>
            {t('runSelectedTargets')}
          </Button>
        </div>
      )}
    >
      <div className="run-target-form">
        <div className="run-target-summary">
          <span>{task.name}</span>
          <strong>{selectedKeys.length}/{options.length}</strong>
        </div>

        <div className="run-target-field">
          <span className="label-with-tip">{t('runBackupTargets')}<Tip content={t('runBackupTargetsTip')} /></span>
          {options.length === 0 ? (
            <p className="muted">{t('runBackupTargetsEmpty')}</p>
          ) : (
            <div className="run-target-list" role="group" aria-label={t('runBackupTargets')}>
              {options.map(({ key, target }) => {
                const runnable = Boolean(target?.enabled);
                const checked = runnable && editor.selectedTargetKeys.includes(key);
                const status = !target
                  ? t('runBackupTargetMissing')
                  : target.enabled ? target.type.toUpperCase() : t('runBackupTargetDisabled');
                return (
                  <label key={key} className={`run-target-option ${runnable ? '' : 'disabled'}`}>
                    <input
                      type="checkbox"
                      checked={checked}
                      disabled={!runnable || running}
                      onChange={(event) => toggleTarget(key, event.target.checked)}
                    />
                    <span className="run-target-option-body">
                      <strong>{target?.name ?? key}</strong>
                      <span>{status}</span>
                    </span>
                  </label>
                );
              })}
            </div>
          )}
          {!canRun && <p className="field-error">{t('selectAtLeastOneTarget')}</p>}
        </div>
      </div>
    </Modal>
  );
}
