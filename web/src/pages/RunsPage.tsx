import { Card } from 'animal-island-ui';
import { ShieldCheck } from 'lucide-react';

import { formatBytes, statusLabel } from '../format';
import { useI18n } from '../i18n';
import { Empty, Panel } from '../components/ui';
import type { Run } from '../types';

export function RunsPage({ runs, locale }: { runs: Run[]; locale: string }) {
  const { t } = useI18n();
  return (
    <Panel title={t('runs')}>
      <div className="runs">
        {runs.length === 0 ? <Empty text={t('noBackupsYet')} /> : runs.map((run) => {
          const targets = run.targets ?? [];
          return (
            <Card key={run.id} color="purple" pattern="purple" className="run">
              <div className="item-head"><h3>{run.task_name}</h3><span className={`badge ${run.status}`}>{statusLabel(run.status, t)}</span></div>
              <p>{new Date(run.started_at).toLocaleString(locale)} · {formatBytes(run.size_bytes)} · {run.encrypted ? t('encrypted') : t('zip')}</p>
              <code>{run.object_name}</code>
              <div className="target-results">
                {targets.map((target) => (
                  <span key={target.target_name}>
                    <ShieldCheck size={14} /> {target.target_name}: {statusLabel(target.status, t)}
                    {target.retention_deleted ? ` · ${t('deleted')} ${target.retention_deleted}` : ''}
                  </span>
                ))}
              </div>
            </Card>
          );
        })}
      </div>
    </Panel>
  );
}
