import { Card } from 'animal-island-ui';
import { CheckCircle2, DatabaseBackup } from 'lucide-react';

import { Empty } from '../components/ui';
import { useI18n } from '../i18n';
import type { RestoreDraft } from './types';
import type { Target } from '../types';

export function RestoreTargetStep({ draft, targets, onDraft }: {
  draft: RestoreDraft;
  targets: Target[];
  onDraft: (patch: Partial<RestoreDraft>) => void;
}) {
  const { t } = useI18n();
  if (targets.length === 0) {
    return <Empty text={t('restoreNoTargets')} />;
  }
  return (
    <div className="restore-choice-list">
      {targets.map((target) => (
        <button
          key={target.key}
          type="button"
          className={`restore-choice ${draft.targetKey === target.key ? 'selected' : ''}`}
          aria-pressed={draft.targetKey === target.key}
          onClick={() => onDraft({ targetKey: target.key })}
        >
          <Card color="app-green" pattern="app-green" className="restore-choice-card">
            <div className="restore-choice-icon"><DatabaseBackup size={22} /></div>
            <div>
              <h3>{target.name}</h3>
              <p>{target.type.toUpperCase()}</p>
              <code>{target.type === 's3' ? target.s3?.root_prefix || target.s3?.bucket_name || target.key : target.webdav?.root_prefix || target.webdav?.url || target.key}</code>
            </div>
            {draft.targetKey === target.key && <CheckCircle2 size={20} />}
          </Card>
        </button>
      ))}
    </div>
  );
}
