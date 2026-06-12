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
              <code>{targetSummary(target)}</code>
              <CardActions onEdit={() => onEdit(target, index)} onDelete={() => onDelete(index)} />
            </Card>
          ))}
        </div>
      )}
    </Panel>
  );
}

function targetSummary(target: Target) {
  switch (target.type) {
    case 's3':
      return target.s3?.root_prefix || target.s3?.bucket_name || target.key;
    case 'webdav':
      return target.webdav?.root_prefix || target.webdav?.url || target.key;
    case 'gcs':
      return target.gcs?.root_prefix || target.gcs?.bucket_name || target.key;
    case 'sftp':
      return target.sftp?.root_prefix || target.sftp?.host || target.key;
    default:
      return target.key;
  }
}
