import { Button, Card, Switch } from 'animal-island-ui';
import { Pencil, Play, Trash2 } from 'lucide-react';

import { retentionLabel, taskRuntimeLabel } from '../format';
import { useI18n } from '../i18n';
import { Empty, Panel } from '../components/ui';
import type { BackupStrategy, ExportOption, Schedule, Source, Target, Task, TaskRuntime } from '../types';

export function TasksPage({ tasks, sources, targets, schedules, exportOptions, backupStrategies, taskRuntime, locale, onAdd, onEdit, onDelete, onRun }: {
  tasks: Task[];
  sources: Source[];
  targets: Target[];
  schedules: Schedule[];
  exportOptions: ExportOption[];
  backupStrategies: BackupStrategy[];
  taskRuntime: Record<string, TaskRuntime>;
  locale: string;
  onAdd: () => void;
  onEdit: (task: Task, index: number) => void;
  onDelete: (index: number) => void;
  onRun: (task: Task) => void;
}) {
  const { t } = useI18n();
  const exportName = (key: string) => exportOptions.find((option) => option.key === key)?.name ?? key;
  const strategy = (key: string) => backupStrategies.find((item) => item.key === key);
  return (
    <Panel title={t('tasks')} actionLabel={t('addTask')} onAdd={onAdd}>
      {tasks.length === 0 ? <Empty text={t('noTasksYet')} /> : (
        <div className="grid-list">
          {tasks.map((task, index) => {
            const selectedStrategy = strategy(task.backup_strategy_key);
            return (
              <Card key={task.key} color="app-yellow" pattern="app-yellow" className="item">
                <div className="item-head">
                  <h3>{task.name}</h3>
                  <Switch checked={task.enabled} disabled />
                </div>
                <p>{t('source')}: {sources.find((s) => s.key === task.source_key)?.name ?? task.source_key}</p>
                <p>{t('schedule')}: {schedules.find((s) => s.key === task.schedule_key)?.name ?? task.schedule_key}</p>
                <p className={`task-runtime ${taskRuntime[task.key]?.status ?? 'unknown'}`}>{taskRuntimeLabel(taskRuntime[task.key], locale, t)}</p>
                <p>{t('targets')}: {task.target_keys.map((k) => targets.find((target) => target.key === k)?.name ?? k).join(', ') || t('none')}</p>
                <p>{t('exportOption')}: {exportName(task.export_option_key)}</p>
                <p>{t('backupStrategy')}: {selectedStrategy?.name ?? task.backup_strategy_key} · {retentionLabel(selectedStrategy?.retention, t)}</p>
                <p>{task.encryption.enabled ? t('encryptedPackage') : t('plainPortableZip')}</p>
                <div className="card-actions">
                  <Button type="primary" icon={<Play size={16} />} onClick={() => onRun(task)}>{t('runNow')}</Button>
                  <Button type="default" icon={<Pencil size={16} />} onClick={() => onEdit(task, index)}>{t('edit')}</Button>
                  <Button type="default" danger icon={<Trash2 size={16} />} onClick={() => onDelete(index)}>{t('delete')}</Button>
                </div>
              </Card>
            );
          })}
        </div>
      )}
    </Panel>
  );
}
