import { useEffect, useState } from 'react';
import { Button, Checkbox, Input, Modal, Radio } from 'animal-island-ui';
import { CheckCircle2, ServerCog, XCircle } from 'lucide-react';

import { api } from '../api';
import { defaultGCS, defaultS3, defaultSFTP, defaultWebDAV } from '../configDefaults';
import { useI18n } from '../i18n';
import { Field, FormGrid, ModalFooter, SwitchField, Tip } from '../components/ui';
import type { Editor, GCSConfig, S3Config, SFTPConfig, Target, TargetType, TestResult, WebDAVConfig } from '../types';

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
  const gcs = defaultGCS(value);
  const sftp = defaultSFTP(value);
  const set = (patch: Partial<Target>) => onChange({ ...value, ...patch });
  const setS3 = (patch: Partial<S3Config>) => set({ s3: { ...s3, ...patch } });
  const setWebDAV = (patch: Partial<WebDAVConfig>) => set({ webdav: { ...webdav, ...patch } });
  const setGCS = (patch: Partial<GCSConfig>) => set({ gcs: { ...gcs, ...patch } });
  const setSFTP = (patch: Partial<SFTPConfig>) => set({ sftp: { ...sftp, ...patch } });
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
            { label: 'WebDAV', value: 'webdav' },
            { label: 'Google Cloud Storage', value: 'gcs' },
            { label: 'SFTP', value: 'sftp' }
          ]}
        />
      </Field>
      {value.type === 's3' && (
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
      )}
      {value.type === 'webdav' && (
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
      {value.type === 'gcs' && (
        <FormGrid>
          <Field label={t('bucket')}>
            <Input value={gcs.bucket_name} onChange={(e) => setGCS({ bucket_name: e.target.value })} allowClear />
          </Field>
          <Field label={t('rootPrefix')}>
            <Input value={gcs.root_prefix} onChange={(e) => setGCS({ root_prefix: e.target.value })} allowClear />
          </Field>
          <div className="form-grid-span-2">
            <Field label={t('gcsCredentialsJson')} help={t('secretPreserveHelp')}>
              <textarea
                className="field-textarea secret-textarea"
                value={gcs.credentials_json ?? ''}
                onChange={(e) => setGCS({ credentials_json: e.target.value })}
                rows={5}
                spellCheck={false}
                autoComplete="off"
              />
            </Field>
          </div>
        </FormGrid>
      )}
      {value.type === 'sftp' && (
        <>
          <FormGrid>
            <Field label={t('sftpHost')}>
              <Input value={sftp.host} onChange={(e) => setSFTP({ host: e.target.value })} allowClear />
            </Field>
            <Field label={t('sftpPort')}>
              <Input type="number" min={1} max={65535} value={String(sftp.port || 22)} onChange={(e) => setSFTP({ port: Math.max(1, Number(e.target.value) || 22) })} />
            </Field>
            <Field label={t('rootPrefix')}>
              <Input value={sftp.root_prefix} onChange={(e) => setSFTP({ root_prefix: e.target.value })} allowClear />
            </Field>
            <Field label={t('sftpUsername')}>
              <Input value={sftp.username} onChange={(e) => setSFTP({ username: e.target.value })} allowClear />
            </Field>
            <Field label={t('sftpPassword')} help={t('secretPreserveHelp')}>
              <Input type="password" value={sftp.password ?? ''} onChange={(e) => setSFTP({ password: e.target.value })} allowClear />
            </Field>
            <div className="form-grid-span-2">
              <Field label={t('sftpPrivateKey')} help={t('secretPreserveHelp')}>
                <textarea
                  className="field-textarea secret-textarea"
                  value={sftp.private_key ?? ''}
                  onChange={(e) => setSFTP({ private_key: e.target.value })}
                  rows={5}
                  spellCheck={false}
                  autoComplete="off"
                />
              </Field>
            </div>
            <Field label={t('sftpPrivateKeyPassphrase')} help={t('secretPreserveHelp')}>
              <Input type="password" value={sftp.private_key_passphrase ?? ''} onChange={(e) => setSFTP({ private_key_passphrase: e.target.value })} allowClear />
            </Field>
            <Field label={t('sftpHostKeySha256')} help={t('sftpHostKeySha256Tip')}>
              <Input value={sftp.host_key_sha256 ?? ''} onChange={(e) => setSFTP({ host_key_sha256: e.target.value })} allowClear />
            </Field>
          </FormGrid>
          <SwitchField label={t('sftpInsecureIgnoreHostKey')} checked={Boolean(sftp.insecure_ignore_host_key)} onChange={(insecure) => setSFTP({ insecure_ignore_host_key: insecure })} />
        </>
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
