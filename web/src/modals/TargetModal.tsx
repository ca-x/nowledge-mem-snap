import { useEffect, useState } from 'react';
import { Button, Checkbox, Input, Modal, Radio } from 'animal-island-ui';
import { CheckCircle2, ServerCog, XCircle } from 'lucide-react';

import { api } from '../api';
import { defaultS3, defaultWebDAV } from '../configDefaults';
import { useI18n } from '../i18n';
import { Field, FormGrid, ModalFooter, SwitchField, Tip } from '../components/ui';
import type { Editor, S3Config, Target, TargetType, TestResult, WebDAVConfig } from '../types';

export function TargetModal({ editor, saving, onChange, onCancel, onSave }: {
  editor: Editor<Target> | null;
  saving: boolean;
  onChange: (next: Editor<Target> | null) => void;
  onCancel: () => void;
  onSave: (editor: Editor<Target>) => void;
}) {
  const { t } = useI18n();
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [testing, setTesting] = useState(false);
  const [uploadDuringTest, setUploadDuringTest] = useState(false);

  useEffect(() => {
    setTestResult(null);
    setTesting(false);
    setUploadDuringTest(false);
  }, [editor?.index, editor?.value.key]);

  if (!editor) return null;
  const target = editor.value;
  const setTarget = (value: Target) => onChange({ ...editor, value });
  const test = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      setTestResult(await api<TestResult>('/api/targets/test', {
        method: 'POST',
        body: JSON.stringify({ target, upload_file: uploadDuringTest })
      }));
    } catch (err) {
      setTestResult({ ok: false, message: err instanceof Error ? err.message : t('testFailed') });
    } finally {
      setTesting(false);
    }
  };
  return (
    <Modal
      open
      title={editor.index < 0 ? t('addTargetTitle') : t('editTargetTitle')}
      typewriter={false}
      width={700}
      onClose={onCancel}
      footer={<ModalFooter saving={saving} onCancel={onCancel} onSave={() => onSave(editor)} />}
    >
      <TargetForm value={target} onChange={setTarget} />
      <div className="test-strip">
        <Button type="default" icon={<ServerCog size={16} />} loading={testing} onClick={test}>{t('testTarget')}</Button>
        <Checkbox
          className="test-download-choice"
          value={uploadDuringTest ? ['upload'] : []}
          onChange={(values) => setUploadDuringTest(values.map(String).includes('upload'))}
          options={[{ label: t('uploadTestFile'), value: 'upload' }]}
        />
        <Tip content={t('uploadTestFileTip')} />
        {testResult && (
          <span className={`test-result ${testResult.ok ? 'success' : 'danger'}`}>
            {testResult.ok ? <CheckCircle2 size={16} /> : <XCircle size={16} />}
            {targetTestMessage(testResult, t)}
          </span>
        )}
      </div>
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

function targetTestMessage(result: TestResult, t: (key: string) => string) {
  if (result.code) {
    const key = `targetTest.${result.code}`;
    const template = t(key);
    if (template !== key) {
      return interpolate(template, result.details ?? {});
    }
  }
  return result.message || t('testFailed');
}

function interpolate(template: string, values: Record<string, string>) {
  return template.replace(/\{([a-zA-Z0-9_]+)\}/g, (_, key) => values[key] ?? '');
}
