import { Input, Modal, Radio } from 'animal-island-ui';

import { defaultRunAt, weekdayOptions } from '../configDefaults';
import { useI18n } from '../i18n';
import { Field, FormGrid, ModalFooter, NativeSelect, SwitchField } from '../components/ui';
import type { Editor, Schedule, ScheduleType } from '../types';

export function ScheduleModal({ editor, saving, timezoneLabel, onChange, onCancel, onSave }: {
  editor: Editor<Schedule> | null;
  saving: boolean;
  timezoneLabel: string;
  onChange: (next: Editor<Schedule> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<Schedule>) => void;
}) {
  const { t } = useI18n();
  if (!editor) return null;
  const schedule = editor.value;
  const setSchedule = (value: Schedule) => onChange({ ...editor, value });
  const set = (patch: Partial<Schedule>) => setSchedule({ ...schedule, ...patch });
  return (
    <Modal
      open
      title={editor.index < 0 ? t('addScheduleTitle') : t('editScheduleTitle')}
      typewriter={false}
      width={640}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <div className="editor-form">
        <Field label={t('name')}>
          <Input value={schedule.name} onChange={(e) => set({ name: e.target.value })} allowClear />
        </Field>
        <SwitchField label={t('enabled')} checked={schedule.enabled} onChange={(enabled) => set({ enabled })} />
        <FormGrid>
          <Field label={t('type')}>
            <Radio
              value={schedule.type}
              onChange={(type) => {
                const nextType = type as ScheduleType;
                set({ type: nextType, run_at: nextType === 'once' && !schedule.run_at ? defaultRunAt() : schedule.run_at });
              }}
              options={[
                { value: 'daily', label: t('daily') },
                { value: 'weekly', label: t('weekly') },
                { value: 'once', label: t('once') }
              ]}
            />
          </Field>
          {schedule.type === 'once' ? (
            <Field label={t('runAtTimezone').replace('{timezone}', timezoneLabel)}>
              <Input type="datetime-local" value={schedule.run_at || ''} onChange={(e) => set({ run_at: e.target.value })} />
            </Field>
          ) : (
            <Field label={t('time')}>
              <Input type="time" value={schedule.time} onChange={(e) => set({ time: e.target.value })} />
            </Field>
          )}
          {schedule.type === 'weekly' && (
            <Field label={t('weekday')}>
              <NativeSelect
                value={schedule.weekday || 'sunday'}
                onChange={(weekday) => set({ weekday })}
                options={weekdayOptions(t)}
              />
            </Field>
          )}
        </FormGrid>
      </div>
    </Modal>
  );
}
