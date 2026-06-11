type RuntimeConfig = {
  basePath?: string;
};

declare global {
  interface Window {
    __NMEM_SNAP_CONFIG__?: RuntimeConfig;
  }
}

export const basePath = normalizeBasePath(window.__NMEM_SNAP_CONFIG__?.basePath ?? '');

export function appPath(path: string) {
  const normalized = normalizeAppPath(path);
  if (basePath === '') {
    return normalized;
  }
  if (normalized === '/') {
    return `${basePath}/`;
  }
  return `${basePath}${normalized}`;
}

export function apiPath(path: string) {
  if (isAbsoluteURL(path)) {
    return path;
  }
  return appPath(path);
}

export function assetPath(path: string) {
  return appPath(path);
}

export function routePath(pathname = window.location.pathname) {
  if (basePath !== '') {
    if (pathname === basePath) {
      return '/';
    }
    if (pathname.startsWith(`${basePath}/`)) {
      return pathname.slice(basePath.length) || '/';
    }
  }
  return pathname || '/';
}

export function currentAppPath() {
  return `${routePath()}${window.location.search}${window.location.hash}`;
}

export function nextFromSearch() {
  return sanitizeAppPath(new URLSearchParams(window.location.search).get('next') || '/');
}

export function sanitizeAppPath(value: string) {
  const raw = value.trim();
  if (raw === '' || isAbsoluteURL(raw) || raw.startsWith('//')) {
    return '/';
  }
  if (!raw.startsWith('/')) {
    return '/';
  }
  if (basePath !== '') {
    if (raw === basePath) {
      return '/';
    }
    if (raw.startsWith(`${basePath}/`)) {
      return raw.slice(basePath.length) || '/';
    }
  }
  return raw;
}

function normalizeAppPath(path: string) {
  if (path === '') {
    return '/';
  }
  return path.startsWith('/') ? path : `/${path}`;
}

function normalizeBasePath(value: string) {
  let normalized = value.trim();
  if (normalized === '' || normalized === '/') {
    return '';
  }
  if (!normalized.startsWith('/')) {
    normalized = `/${normalized}`;
  }
  while (normalized.endsWith('/')) {
    normalized = normalized.slice(0, -1);
  }
  return normalized === '/' ? '' : normalized;
}

function isAbsoluteURL(value: string) {
  return /^[a-z][a-z0-9+.-]*:\/\//i.test(value);
}
