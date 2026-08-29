import { useEffect, useMemo, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { TFunction } from 'i18next';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { ChannelModelConfig, configurableChannelEndpointApiFormats } from '../data/schema';
import { apiFormatLabel } from '../utils/api-format';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  model: string;
  config?: ChannelModelConfig | null;
  onSave: (config: ChannelModelConfig | null) => void;
}

// Sentinels: radix Select rejects empty-string item values.
const FOLLOW_CHANNEL = '__follow_channel__';
const EFFORT_UNSET = '__unset__';
const EFFORT_MAP_DISABLED = '__disabled__';
// Efforts admins can override per-model. Kept in sync with
// modelConfigReasoningEfforts in internal/server/biz/channel.go.
const EFFORT_MAP_KEYS = ['low', 'medium', 'high', 'xhigh', 'max'] as const;
const EFFORT_MAP_TARGETS = ['low', 'medium', 'high', 'xhigh', 'max'] as const;

// Keep in sync with modelConfigReasoningEfforts in internal/server/biz/channel.go.
const EFFORT_OPTIONS = ['low', 'medium', 'high', 'xhigh', 'max', 'none'] as const;

// Short human-readable summary of a saved config, shown on the model badge tooltip.
export function summarizeChannelModelConfig(config: ChannelModelConfig, t: TFunction): string {
  const parts: string[] = [];
  if (config.apiFormat) {
    parts.push(t('channels.dialogs.modelConfig.summary.apiFormat', { value: apiFormatLabel(t, config.apiFormat) }));
  }
  if (config.path) {
    parts.push(t('channels.dialogs.modelConfig.summary.path', { value: config.path }));
  }
  if (config.reasoning?.enabled === false) {
    parts.push(t('channels.dialogs.modelConfig.summary.reasoningOff'));
  } else {
    if (config.reasoning?.defaultEffort) {
      parts.push(t('channels.dialogs.modelConfig.summary.defaultEffort', { value: config.reasoning.defaultEffort }));
    }
    if (config.reasoning?.defaultBudget) {
      parts.push(t('channels.dialogs.modelConfig.summary.defaultBudget', { value: config.reasoning.defaultBudget }));
    }
    const mapEntries = Object.entries(config.reasoning?.effortMap ?? {});
    if (mapEntries.length) {
      parts.push(
        t('channels.dialogs.modelConfig.summary.effortMap', { value: mapEntries.map(([k, v]) => `${k}→${v ?? 'off'}`).join(', ') }),
      );
    }
  }
  return parts.join(' · ');
}

interface DraftState {
  apiFormat: string;
  path: string;
  forceDisable: boolean;
  defaultEffort: string;
  defaultBudget: string;
  effortMap: Record<string, string>;
}

function draftFromConfig(config?: ChannelModelConfig | null): DraftState {
  const effortMap: Record<string, string> = {};
  for (const key of EFFORT_MAP_KEYS) {
    const value = config?.reasoning?.effortMap?.[key];
    effortMap[key] = value === null ? EFFORT_MAP_DISABLED : (value ?? '');
  }
  return {
    apiFormat: config?.apiFormat || '',
    path: config?.path || '',
    forceDisable: config?.reasoning?.enabled === false,
    defaultEffort: config?.reasoning?.defaultEffort || '',
    defaultBudget: config?.reasoning?.defaultBudget ? String(config.reasoning.defaultBudget) : '',
    effortMap,
  };
}

export function ChannelsModelConfigDialog({ open, onOpenChange, model, config, onSave }: Props) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState<DraftState>(() => draftFromConfig(config));

  useEffect(() => {
    if (open) {
      setDraft(draftFromConfig(config));
    }
  }, [open, config, model]);

  const budgetValue = draft.defaultBudget.trim() === '' ? null : Number(draft.defaultBudget.trim());
  const budgetInvalid = draft.defaultBudget.trim() !== '' && (!Number.isInteger(budgetValue) || (budgetValue ?? 0) <= 0);

  const buildConfig = useCallback((): ChannelModelConfig | null => {
    const effortMapEntries = EFFORT_MAP_KEYS.filter((key) => draft.effortMap[key] !== '');
    const isEmpty =
      draft.apiFormat === '' &&
      draft.path.trim() === '' &&
      !draft.forceDisable &&
      draft.defaultEffort === '' &&
      draft.defaultBudget.trim() === '' &&
      effortMapEntries.length === 0;
    if (isEmpty) {
      return null;
    }
    const hasReasoning = draft.forceDisable || draft.defaultEffort !== '' || budgetValue !== null || effortMapEntries.length > 0;
    return {
      model,
      ...(draft.apiFormat ? { apiFormat: draft.apiFormat } : {}),
      ...(draft.path.trim() ? { path: draft.path.trim() } : {}),
      ...(hasReasoning
        ? {
            reasoning: {
              ...(draft.forceDisable ? { enabled: false } : {}),
              ...(!draft.forceDisable && draft.defaultEffort ? { defaultEffort: draft.defaultEffort } : {}),
              ...(!draft.forceDisable && budgetValue !== null ? { defaultBudget: budgetValue } : {}),
              ...(!draft.forceDisable && effortMapEntries.length > 0
                ? {
                    effortMap: Object.fromEntries(
                      effortMapEntries.map((key) => [key, draft.effortMap[key] === EFFORT_MAP_DISABLED ? null : draft.effortMap[key]]),
                    ),
                  }
                : {}),
            },
          }
        : {}),
    };
  }, [draft, model, budgetValue]);

  // Compare drafts (not built configs) so null fields from the backend don't
  // count as modifications.
  const dirty = useMemo(() => {
    const initial = draftFromConfig(config && (config.apiFormat || config.path || config.reasoning) ? config : null);
    return JSON.stringify(draft) !== JSON.stringify(initial);
  }, [draft, config]);

  const summary = useMemo(() => {
    const built = buildConfig();
    return built ? summarizeChannelModelConfig(built, t) : '';
  }, [buildConfig, t]);

  const hasExistingConfig = !!(config && (config.apiFormat || config.path || config.reasoning));

  const patch = (partial: Partial<DraftState>) => setDraft((prev) => ({ ...prev, ...partial }));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-[480px]'>
        <DialogHeader className='text-left'>
          <DialogTitle className='flex items-center gap-2'>
            {t('channels.dialogs.modelConfig.title')}
            <Badge variant='secondary' className='font-mono text-xs'>
              {model}
            </Badge>
          </DialogTitle>
          <DialogDescription>{t('channels.dialogs.modelConfig.description')}</DialogDescription>
        </DialogHeader>

        <div className='max-h-[60vh] space-y-4 overflow-y-auto pr-1'>
          <Card>
            <CardHeader>
              <CardTitle className='text-base'>{t('channels.dialogs.modelConfig.protocol.title')}</CardTitle>
              <CardDescription>{t('channels.dialogs.modelConfig.protocol.description')}</CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <div className='space-y-1.5'>
                <Label>{t('channels.dialogs.modelConfig.protocol.apiFormat.label')}</Label>
                <Select
                  value={draft.apiFormat || FOLLOW_CHANNEL}
                  onValueChange={(value) => patch({ apiFormat: value === FOLLOW_CHANNEL ? '' : value })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={FOLLOW_CHANNEL}>{t('channels.dialogs.modelConfig.protocol.apiFormat.followChannel')}</SelectItem>
                    {configurableChannelEndpointApiFormats.map((format) => (
                      <SelectItem key={format} value={format}>
                        {apiFormatLabel(t, format)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className='space-y-1.5'>
                <Label>{t('channels.dialogs.modelConfig.protocol.path.label')}</Label>
                <Input
                  value={draft.path}
                  onChange={(e) => patch({ path: e.target.value })}
                  placeholder={t('channels.dialogs.modelConfig.protocol.path.placeholder')}
                  disabled={!draft.apiFormat}
                  className='font-mono text-sm'
                />
                <p className='text-muted-foreground text-xs'>{t('channels.dialogs.modelConfig.protocol.path.hint')}</p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className='text-base'>{t('channels.dialogs.modelConfig.reasoning.title')}</CardTitle>
              <CardDescription>{t('channels.dialogs.modelConfig.reasoning.description')}</CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <div className='flex items-center gap-2'>
                <Checkbox
                  id='model-config-force-disable'
                  checked={draft.forceDisable}
                  onCheckedChange={(checked) => patch({ forceDisable: checked === true })}
                />
                <div className='space-y-0.5'>
                  <Label htmlFor='model-config-force-disable' className='cursor-pointer text-sm font-normal'>
                    {t('channels.dialogs.modelConfig.reasoning.forceDisable.label')}
                  </Label>
                  <p className='text-muted-foreground text-xs'>{t('channels.dialogs.modelConfig.reasoning.forceDisable.description')}</p>
                </div>
              </div>

              <div className='space-y-1.5'>
                <Label>{t('channels.dialogs.modelConfig.reasoning.defaultEffort.label')}</Label>
                <Select
                  value={draft.defaultEffort || EFFORT_UNSET}
                  onValueChange={(value) => patch({ defaultEffort: value === EFFORT_UNSET ? '' : value })}
                  disabled={draft.forceDisable}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={EFFORT_UNSET}>{t('channels.dialogs.modelConfig.reasoning.notSet')}</SelectItem>
                    {EFFORT_OPTIONS.map((effort) => (
                      <SelectItem key={effort} value={effort}>
                        {effort}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className='text-muted-foreground text-xs'>{t('channels.dialogs.modelConfig.reasoning.defaultEffort.description')}</p>
              </div>

              <div className='space-y-1.5'>
                <Label>{t('channels.dialogs.modelConfig.reasoning.defaultBudget.label')}</Label>
                <Input
                  type='number'
                  min={1}
                  step={1}
                  value={draft.defaultBudget}
                  onChange={(e) => patch({ defaultBudget: e.target.value })}
                  placeholder={t('channels.dialogs.modelConfig.reasoning.notSet')}
                  disabled={draft.forceDisable}
                />
                {budgetInvalid ? (
                  <p className='text-destructive text-xs'>{t('channels.dialogs.modelConfig.reasoning.defaultBudget.invalid')}</p>
                ) : (
                  <p className='text-muted-foreground text-xs'>{t('channels.dialogs.modelConfig.reasoning.defaultBudget.description')}</p>
                )}
              </div>

              <div className='space-y-2'>
                <div>
                  <Label>{t('channels.dialogs.modelConfig.reasoning.effortMap.label')}</Label>
                  <p className='text-muted-foreground text-xs'>{t('channels.dialogs.modelConfig.reasoning.effortMap.description')}</p>
                </div>
                <div className='space-y-1.5'>
                  {EFFORT_MAP_KEYS.map((key) => (
                    <div key={key} className='flex items-center gap-2'>
                      <span className='w-16 shrink-0 font-mono text-sm'>{key}</span>
                      <Select
                        value={draft.effortMap[key] || EFFORT_UNSET}
                        onValueChange={(value) =>
                          patch({ effortMap: { ...draft.effortMap, [key]: value === EFFORT_UNSET ? '' : value } })
                        }
                        disabled={draft.forceDisable}
                      >
                        <SelectTrigger className='h-8 flex-1'>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value={EFFORT_UNSET}>{t('channels.dialogs.modelConfig.reasoning.effortMap.follow')}</SelectItem>
                          {EFFORT_MAP_TARGETS.map((target) => (
                            <SelectItem key={target} value={target}>
                              {target}
                            </SelectItem>
                          ))}
                          <SelectItem value={EFFORT_MAP_DISABLED}>{t('channels.dialogs.modelConfig.reasoning.effortMap.disabled')}</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <DialogFooter className='items-center gap-2 sm:justify-between'>
          <div className='flex min-w-0 flex-1 items-center gap-2'>
            {hasExistingConfig && (
              <Button type='button' variant='ghost' className='text-destructive hover:text-destructive' onClick={() => onSave(null)}>
                {t('channels.dialogs.modelConfig.buttons.clear')}
              </Button>
            )}
            {summary && <span className='text-muted-foreground truncate text-xs'>{summary}</span>}
          </div>
          <div className='flex gap-2'>
            <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
              {t('common.buttons.cancel')}
            </Button>
            <Button type='button' onClick={() => onSave(buildConfig())} disabled={!dirty || budgetInvalid}>
              {t('common.buttons.save')}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
