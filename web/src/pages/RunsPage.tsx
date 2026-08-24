import { Card } from 'animal-island-ui';
import { CircleAlert, ShieldCheck } from 'lucide-react';

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
              {run.error && (
                <div className="run-error-detail">
                  <CircleAlert size={18} />
                  <div><strong>{t('failureDetails')}</strong><p>{run.error}</p></div>
                </div>
              )}
              {run.object_name && <code>{run.object_name}</code>}
              {targets.length > 0 && <div className="target-results">
                {targets.map((target) => (
                  <div key={target.target_name} className={`target-result ${target.status}`}>
                    <span className="target-result-summary">
                      {target.status === 'failed' ? <CircleAlert size={14} /> : <ShieldCheck size={14} />}
                      {target.target_name}: {statusLabel(target.status, t)}
                      {target.retention_deleted ? ` · ${t('deleted')} ${target.retention_deleted}` : ''}
                    </span>
                    {target.error && <small className="target-error">{target.error}</small>}
                  </div>
                ))}
              </div>}
            </Card>
          );
        })}
      </div>
    </Panel>
  );
}
