import { Input, Modal, Radio } from 'animal-island-ui';

import { defaultS3, defaultWebDAV } from '../configDefaults';
import { useI18n } from '../i18n';
import { Field, FormGrid, ModalFooter, SwitchField } from '../components/ui';
import type { Editor, S3Config, Target, TargetType, WebDAVConfig } from '../types';

export function TargetModal({ editor, saving, onChange, onCancel, onSave }: {
  editor: Editor<Target> | null;
  saving: boolean;
  onChange: (next: Editor<Target> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<Target>) => void;
}) {
  const { t } = useI18n();
  if (!editor) return null;
  const setTarget = (value: Target) => onChange({ ...editor, value });
  return (
    <Modal
      open
      title={editor.index < 0 ? t('addTargetTitle') : t('editTargetTitle')}
      typewriter={false}
      width={700}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <TargetForm value={editor.value} onChange={setTarget} />
    </Modal>
  );
}

function TargetForm({ value, onChange }: { value: Target; onChange: (value: Target) => void }) {
  const { t } = useI18n();
  const s3 = defaultS3(value);
  const webdav = defaultWebDAV(value);
  const set = (patch: Partial<Target>) => onChange({ ...value, ...patch });
  const setS3 = (patch: Partial<S3Config>) => set({ s3: { ...s3, ...patch } });
  const setWebDAV = (patch: Partial<WebDAVConfig>) => set({ webdav: { ...webdav, ...patch } });
  return (
    <div className="editor-form">
      <Field label={t('name')}>
        <Input value={value.name} onChange={(e) => set({ name: e.target.value })} allowClear />
      </Field>
      <SwitchField label={t('enabled')} checked={value.enabled} onChange={(enabled) => set({ enabled })} />
      <Field label={t('type')}>
        <Radio
          value={value.type}
          onChange={(next) => set({ type: String(next) as TargetType })}
          options={[
            { label: 'S3 / R2', value: 's3' },
            { label: 'WebDAV', value: 'webdav' }
          ]}
        />
      </Field>
      {value.type === 's3' ? (
        <>
          <FormGrid>
            <Field label={t('endpointUrl')}>
              <Input value={s3.endpoint_url} onChange={(e) => setS3({ endpoint_url: e.target.value })} allowClear />
            </Field>
            <Field label={t('region')}>
              <Input value={s3.region} onChange={(e) => setS3({ region: e.target.value })} allowClear />
            </Field>
            <Field label={t('bucket')}>
              <Input value={s3.bucket_name} onChange={(e) => setS3({ bucket_name: e.target.value })} allowClear />
            </Field>
            <Field label={t('rootPrefix')}>
              <Input value={s3.root_prefix} onChange={(e) => setS3({ root_prefix: e.target.value })} allowClear />
            </Field>
            <Field label={t('accessKeyId')}>
              <Input value={s3.access_key_id} onChange={(e) => setS3({ access_key_id: e.target.value })} allowClear />
            </Field>
            <Field label={t('secretAccessKey')} help={t('secretPreserveHelp')}>
              <Input type="password" value={s3.secret_access_key ?? ''} onChange={(e) => setS3({ secret_access_key: e.target.value })} allowClear />
            </Field>
          </FormGrid>
          <SwitchField label={t('pathStyle')} checked={s3.path_style} onChange={(pathStyle) => setS3({ path_style: pathStyle })} />
        </>
      ) : (
        <FormGrid>
          <Field label={t('webdavUrl')}>
            <Input value={webdav.url} onChange={(e) => setWebDAV({ url: e.target.value })} allowClear />
          </Field>
          <Field label={t('rootPrefix')}>
            <Input value={webdav.root_prefix} onChange={(e) => setWebDAV({ root_prefix: e.target.value })} allowClear />
          </Field>
          <Field label={t('webdavUsername')}>
            <Input value={webdav.username} onChange={(e) => setWebDAV({ username: e.target.value })} allowClear />
          </Field>
          <Field label={t('webdavPassword')} help={t('secretPreserveHelp')}>
            <Input type="password" value={webdav.password ?? ''} onChange={(e) => setWebDAV({ password: e.target.value })} allowClear />
          </Field>
        </FormGrid>
      )}
    </div>
  );
}
