import { Card, Switch } from 'animal-island-ui';

import { useI18n } from '../i18n';
import { CardActions, Empty, Panel } from '../components/ui';
import type { Target } from '../types';

export function TargetsPage({ targets, onAdd, onEdit, onDelete }: {
  targets: Target[];
  onAdd: () => void;
  onEdit: (target: Target, index: number) => void;
  onDelete: (index: number) => void;
}) {
  const { t } = useI18n();
  return (
    <Panel title={t('targets')} actionLabel={t('addTarget')} onAdd={onAdd}>
      {targets.length === 0 ? <Empty text={t('noTargetsYet')} /> : (
        <div className="grid-list">
          {targets.map((target, index) => (
            <Card key={target.key} color="app-green" pattern="app-green" className="item">
              <div className="item-head">
                <h3>{target.name}</h3>
                <Switch checked={target.enabled} disabled />
              </div>
              <p>{target.type.toUpperCase()}</p>
              <code>{target.type === 's3' ? target.s3?.bucket_name : target.webdav?.url}</code>
              <CardActions onEdit={() => onEdit(target, index)} onDelete={() => onDelete(index)} />
            </Card>
          ))}
        </div>
      )}
    </Panel>
  );
}
