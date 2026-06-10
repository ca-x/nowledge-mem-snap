import { Card, Switch } from 'animal-island-ui';

import { useI18n } from '../i18n';
import { CardActions, Empty, Panel } from '../components/ui';
import type { Source, SourceRoot } from '../types';

export function SourcesPage({ sources, roots, onAdd, onEdit, onDelete }: {
  sources: Source[];
  roots: SourceRoot[];
  onAdd: () => void;
  onEdit: (source: Source, index: number) => void;
  onDelete: (index: number) => void;
}) {
  const { t } = useI18n();
  return (
    <Panel title={t('sources')} actionLabel={t('addSource')} onAdd={onAdd}>
      {sources.length === 0 ? <Empty text={t('noSourcesYet')} /> : (
        <div className="grid-list">
          {sources.map((source, index) => (
            <Card key={source.key} color="app-blue" pattern="app-blue" className="item">
              <div className="item-head">
                <h3>{source.name}</h3>
                <Switch checked={source.enabled} disabled />
              </div>
              <p>{source.type === 'directory' ? t('directorySource') : t('nowledgeMemApi')}</p>
              <code>{source.type === 'directory' ? source.directory?.path : source.nowledge_mem?.api_url}</code>
              {source.type === 'directory' && roots.length === 0 && <p className="muted">{t('directoryRootsDisabled')}</p>}
              {source.remark && <p>{source.remark}</p>}
              <CardActions onEdit={() => onEdit(source, index)} onDelete={() => onDelete(index)} />
            </Card>
          ))}
        </div>
      )}
    </Panel>
  );
}
