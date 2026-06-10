import { Card, Switch } from 'animal-island-ui';

import { scheduleLabel } from '../format';
import { useI18n } from '../i18n';
import { CardActions, Empty, Panel } from '../components/ui';
import type { Schedule } from '../types';

export function SchedulesPage({ schedules, onAdd, onEdit, onDelete }: {
  schedules: Schedule[];
  onAdd: () => void;
  onEdit: (schedule: Schedule, index: number) => void;
  onDelete: (index: number) => void;
}) {
  const { t } = useI18n();
  return (
    <Panel title={t('schedules')} actionLabel={t('addSchedule')} onAdd={onAdd}>
      {schedules.length === 0 ? <Empty text={t('noSchedulesYet')} /> : (
        <div className="grid-list">
          {schedules.map((schedule, index) => (
            <Card key={schedule.key} color="app-yellow" pattern="app-yellow" className="item">
              <div className="item-head">
                <h3>{schedule.name}</h3>
                <Switch checked={schedule.enabled} disabled />
              </div>
              <p>{scheduleLabel(schedule, t)}</p>
              <CardActions onEdit={() => onEdit(schedule, index)} onDelete={() => onDelete(index)} />
            </Card>
          ))}
        </div>
      )}
    </Panel>
  );
}
