import { Card } from 'animal-island-ui';
import { CheckCircle2, ServerCog } from 'lucide-react';

import { Empty } from '../components/ui';
import { useI18n } from '../i18n';
import type { RestoreDraft } from './types';
import type { Source } from '../types';

export function RestoreDestinationStep({ draft, sources, onDraft }: {
  draft: RestoreDraft;
  sources: Source[];
  onDraft: (patch: Partial<RestoreDraft>) => void;
}) {
  const { t } = useI18n();
  if (sources.length === 0) {
    return <Empty text={t('restoreNoSources')} />;
  }
  return (
    <div className="restore-choice-list">
      {sources.map((source) => (
        <button
          key={source.key}
          type="button"
          className={`restore-choice ${draft.destinationSourceKey === source.key ? 'selected' : ''}`}
          aria-pressed={draft.destinationSourceKey === source.key}
          onClick={() => onDraft({ destinationSourceKey: source.key })}
        >
          <Card color="app-blue" pattern="app-blue" className="restore-choice-card">
            <div className="restore-choice-icon"><ServerCog size={22} /></div>
            <div>
              <h3>{source.name}</h3>
              <p>{t('nowledgeMemApi')}</p>
              <code>{source.nowledge_mem?.api_url ?? source.key}</code>
            </div>
            {draft.destinationSourceKey === source.key && <CheckCircle2 size={20} />}
          </Card>
        </button>
      ))}
    </div>
  );
}
