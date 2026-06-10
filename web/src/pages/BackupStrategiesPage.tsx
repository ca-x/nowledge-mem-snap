import { Card } from 'animal-island-ui';

import { retentionLabel } from '../format';
import { useI18n } from '../i18n';
import { CardActions, Empty, Panel } from '../components/ui';
import type { BackupStrategy } from '../types';

export function BackupStrategiesPage({ strategies, onAdd, onEdit, onDelete }: {
  strategies: BackupStrategy[];
  onAdd: () => void;
  onEdit: (strategy: BackupStrategy, index: number) => void;
  onDelete: (index: number) => void;
}) {
  const { t } = useI18n();
  return (
    <Panel title={t('backupStrategies')} actionLabel={t('addBackupStrategy')} onAdd={onAdd}>
      {strategies.length === 0 ? <Empty text={t('noBackupStrategiesYet')} /> : (
        <div className="grid-list">
          {strategies.map((strategy, index) => (
            <Card key={strategy.key} color="app-orange" pattern="app-orange" className="item">
              <div className="item-head">
                <h3>{strategy.name}</h3>
              </div>
              <p>{t('backupStrategyRule')}: {retentionLabel(strategy.retention, t)}</p>
              <CardActions onEdit={() => onEdit(strategy, index)} onDelete={() => onDelete(index)} />
            </Card>
          ))}
        </div>
      )}
    </Panel>
  );
}
