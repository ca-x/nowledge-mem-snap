import { Button, Checkbox, Input, Modal } from 'animal-island-ui';

import { defaultExportConfig, exportConfigFromSelected, exportFlags, exportSelectedValues } from '../configDefaults';
import { useI18n } from '../i18n';
import { Field, ModalFooter, Tip } from '../components/ui';
import type { Editor, ExportConfig, ExportOption } from '../types';

export function ExportOptionModal({ editor, saving, onChange, onCancel, onSave }: {
  editor: Editor<ExportOption> | null;
  saving: boolean;
  onChange: (next: Editor<ExportOption> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<ExportOption>) => void;
}) {
  const { t } = useI18n();
  if (!editor) return null;
  const option = editor.value;
  const setOption = (value: ExportOption) => onChange({ ...editor, value });
  const set = (patch: Partial<ExportOption>) => setOption({ ...option, ...patch });
  return (
    <Modal
      open
      title={editor.index < 0 ? t('addExportOptionTitle') : t('editExportOptionTitle')}
      typewriter={false}
      width={860}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <div className="editor-form">
        <Field label={t('name')}>
          <Input value={option.name} onChange={(e) => set({ name: e.target.value })} allowClear />
        </Field>
        <ExportFields
          title={t('exportContents')}
          help={t('exportOptionTip')}
          value={option.export}
          overridden
          onChange={(value) => set({ export: value })}
          onReset={() => set({ export: defaultExportConfig() })}
          resetLabel={t('restoreRecommendedDefaults')}
        />
      </div>
    </Modal>
  );
}

function ExportFields({ title, help, value, overridden, onChange, onReset, resetLabel }: {
  title: string;
  help?: string;
  value: ExportConfig;
  overridden: boolean;
  onChange: (next: ExportConfig) => void;
  onReset: () => void;
  resetLabel: string;
}) {
  const { t } = useI18n();
  return (
    <div className="export-box">
      <div className="export-box-head">
        <span className="label-with-tip">{title}{help && <Tip content={help} />}</span>
        {overridden && <Button type="default" size="small" onClick={onReset}>{resetLabel}</Button>}
      </div>
      <Checkbox
        className="choice-grid export-choice-grid"
        value={exportSelectedValues(value)}
        onChange={(values) => onChange(exportConfigFromSelected(values.map(String)))}
        options={exportFlags.map((key) => ({ label: t(`export.${key}`), value: key }))}
      />
    </div>
  );
}
