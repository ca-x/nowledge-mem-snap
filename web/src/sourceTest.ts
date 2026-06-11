import type { Translate } from './i18n';
import { formatBytes } from './format';
import { apiPath } from './paths';
import type { Source, TestResult } from './types';

export async function downloadSourceTest(source: Source, t: Translate): Promise<TestResult> {
  const res = await fetch(apiPath('/api/sources/test/download'), {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ source })
  });
  const contentType = res.headers.get('content-type') ?? '';
  if (!res.ok) {
    if (contentType.includes('application/json')) {
      const body = await res.json();
      if (typeof body?.ok === 'boolean' && typeof body?.message === 'string') {
        return body as TestResult;
      }
      throw new Error(body?.error ?? t('testFailed'));
    }
    const text = await res.text();
    throw new Error(text || t('testFailed'));
  }
  const blob = await res.blob();
  const filename = filenameFromContentDisposition(res.headers.get('content-disposition'))
    || `source-test-${new Date().toISOString().replace(/[:.]/g, '-')}.zip`;
  triggerBlobDownload(blob, filename);
  return {
    ok: true,
    code: 'download_started',
    message: 'download started',
    details: { bytes: formatBytes(blob.size) }
  };
}

export function sourceTestMessage(result: TestResult, t: Translate) {
  if (result.code) {
    const key = `sourceTest.${result.code}`;
    const template = t(key);
    if (template !== key) {
      return interpolate(template, result.details ?? {});
    }
  }
  return result.message || t('testFailed');
}

function filenameFromContentDisposition(value: string | null) {
  if (!value) return '';
  const match = /filename="([^"]+)"/i.exec(value) || /filename=([^;]+)/i.exec(value);
  return match?.[1]?.trim() ?? '';
}

function triggerBlobDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function interpolate(template: string, values: Record<string, string>) {
  return template.replace(/\{([a-zA-Z0-9_]+)\}/g, (_, key) => values[key] ?? '');
}
