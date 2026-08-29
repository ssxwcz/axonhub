import type { TFunction } from 'i18next';

/**
 * Returns the localized display label for an API format, falling back to the
 * raw format identifier when no translation exists (e.g. formats added on the
 * backend before their i18n keys land).
 */
export function apiFormatLabel(t: TFunction, format: string): string {
  const key = `channels.dialogs.fields.apiFormat.formats.${format}`;
  const label = t(key);
  return label === key ? format : label;
}
