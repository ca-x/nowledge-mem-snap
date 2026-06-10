import { Card } from 'animal-island-ui';

import { exportFlags, exportSelectedValues } from '../configDefaults';
import { useI18n } from '../i18n';
import { CardActions, Empty, Panel } from '../components/ui';
import type { ExportOption } from '../types';

export function ExportOptionsPage({ options, onAdd, onEdit, onDelete }: {
  options: ExportOption[];
  onAdd: () => void;
  onEdit: (option: ExportOption, index: number) => void;
  onDelete: (index: number) => void;
}) {
  const { t } = useI18n();
  return (
    <Panel title={t('exportOptions')} actionLabel={t('addExportOption')} onAdd={onAdd}>
      {options.length === 0 ? <Empty text={t('noExportOptionsYet')} /> : (
        <div className="grid-list">
          {options.map((option, index) => {
            const selected = exportSelectedValues(option.export);
            return (
              <Card key={option.key} color="app-teal" pattern="app-teal" className="item">
                <div className="item-head">
                  <h3>{option.name}</h3>
                </div>
                <p>{t('exportContents')}: {selected.length}/{exportFlags.length}</p>
                <div className="tag-list">
                  {selected.slice(0, 5).map((flag) => (
                    <span key={flag}>{t(`export.${flag}`)}</span>
                  ))}
                  {selected.length > 5 && <span>{t('moreItems').replace('{count}', String(selected.length - 5))}</span>}
                </div>
                <CardActions onEdit={() => onEdit(option, index)} onDelete={() => onDelete(index)} />
              </Card>
            );
          })}
        </div>
      )}
    </Panel>
  );
}
