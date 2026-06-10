import type { Translate } from './i18n';
import { defaultRetention } from './configDefaults';
import type { Retention, Schedule } from './types';

export function retentionLabel(retention: Retention | undefined, t: Translate) {
  const value = defaultRetention(retention);
  switch (value.mode) {
    case 'keep_last':
      return `${t('retentionLabelLatest')} ${value.keep_last || 7}`;
    case 'keep_days':
      return `${t('retentionLabelDays')} ${value.keep_days || 30}`;
    case 'keep_after':
      return `${t('retentionLabelAfter')} ${value.keep_after || t('date')}`;
    case 'keep_before':
      return `${t('retentionLabelBefore')} ${value.keep_before || t('date')}`;
    default:
      return t('retentionLabelNone');
  }
}

export function scheduleLabel(schedule: Schedule, t: Translate) {
  switch (schedule.type) {
    case 'weekly':
      return `${t('scheduleWeeklyAt')} ${weekdayLabel(schedule.weekday || 'sunday', t)} ${t('at')} ${schedule.time}`;
    case 'once':
      return `${t('scheduleOnceAt')} ${schedule.run_at || t('notSet')}`;
    default:
      return `${t('scheduleDailyAt')} ${schedule.time}`;
  }
}

export function weekdayLabel(weekday: string, t: Translate) {
  return t(`weekday.${weekday || 'sunday'}`);
}

export function statusLabel(status: string, t: Translate) {
  const key = `status${status.charAt(0).toUpperCase()}${status.slice(1)}`;
  const label = t(key);
  return label === key ? status : label;
}

export function formatBytes(value: number) {
  if (!value) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit++;
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}
