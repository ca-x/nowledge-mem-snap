import { Button, Card } from 'animal-island-ui';
import { CheckCircle2, LoaderCircle, RotateCcw, UploadCloud, XCircle } from 'lucide-react';

import { formatBytes } from '../format';
import { useI18n } from '../i18n';
import type { RestoreJob } from '../types';

export function RestoreProgressStep({ job, jobError, starting, onStart, onReset }: {
  job: RestoreJob | null;
  jobError: string;
  starting: boolean;
  onStart: () => void;
  onReset: () => void;
}) {
  const { t } = useI18n();
  const terminal = job && ['completed', 'failed', 'cancelled'].includes(job.state);
  const failed = job?.state === 'failed' || jobError;
  const progress = Math.round((job?.progress ?? 0) * 100);
  return (
    <Card color={failed ? 'app-pink' : 'app-green'} pattern={failed ? 'app-pink' : 'app-green'} className="restore-progress-card">
      <div className="restore-progress-head">
        {job?.state === 'completed' ? <CheckCircle2 size={26} /> : failed ? <XCircle size={26} /> : <LoaderCircle size={26} className="spin" />}
        <div aria-live="polite">
          <h3>{job ? t(`restoreState.${job.state}`) : t('restoreReadyToStart')}</h3>
          <p role={failed ? 'alert' : undefined}>{job?.message || job?.error || jobError || t('restoreReadyMessage')}</p>
        </div>
      </div>
      <div className="restore-progress-bar" aria-label={t('restoreProgress')} aria-valuemin={0} aria-valuemax={100} aria-valuenow={progress} role="progressbar">
        <span style={{ width: `${Math.max(progress, job && !terminal ? 8 : 0)}%` }} />
      </div>
      <div className="restore-progress-grid">
        <Metric label={t('restoreProgress')} value={`${progress}%`} />
        <Metric label={t('restoreObjectSize')} value={formatBytes(job?.size_bytes ?? 0)} />
        <Metric label={t('restoreImported')} value={String(job?.imported ?? 0)} />
        <Metric label={t('restoreSkipped')} value={String(job?.skipped ?? 0)} />
        <Metric label={t('restoreFailedItems')} value={String(job?.failed ?? 0)} />
        <Metric label={t('restoreMemJob')} value={job?.mem_job_id || t('notSet')} />
      </div>
      <div className="card-actions">
        {!job && <Button type="primary" icon={<UploadCloud size={16} />} loading={starting} disabled={starting} onClick={onStart}>{t('restoreStart')}</Button>}
        {terminal && <Button type="default" icon={<RotateCcw size={16} />} onClick={onReset}>{t('restoreStartAnother')}</Button>}
      </div>
    </Card>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="restore-metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}
