'use client';

import React, { useMemo, useState } from 'react';
import { CallbackBlock as CallbackBlockType, CallbackChoice } from '../../types/enrichment';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { toast } from 'sonner';
import { appConfig } from '@/config';
import { Loader2, ShieldAlert, Zap } from 'lucide-react';

interface CallbackBlockProps {
  block: CallbackBlockType;
}

const resolveUrl = (rawUrl?: string | null): string | null => {
  if (!rawUrl) return null;

  if (rawUrl.startsWith('http://') || rawUrl.startsWith('https://')) {
    return rawUrl;
  }

  const base = appConfig.apiBaseUrl?.replace(/\/$/, '') || '';
  const suffix = rawUrl.startsWith('/') ? rawUrl : `/${rawUrl}`;
  return `${base}${suffix}`;
};

const pickUrl = (block: CallbackBlockType): string | null => {
  const fromBlock = block.callback_url || block.webhook_url;
  const fromExtra = (block.extra?.callback_url || block.extra?.webhook_url) as string | undefined;
  return resolveUrl(fromBlock || fromExtra || block.raw?.callback_url || block.raw?.webhook_url);
};

const defaultBody = (choice: CallbackChoice, block: CallbackBlockType) => ({
  choice: choice.value,
  label: choice.label,
  description: choice.description,
  extra: block.extra,
});

export function CallbackBlock({ block }: CallbackBlockProps) {
  const [pendingChoice, setPendingChoice] = useState<string | null>(null);

  const actionUrl = useMemo(() => pickUrl(block), [block]);

  const handleExecute = async (choice: CallbackChoice) => {
    if (!actionUrl) {
      toast.error('未找到可用的回调地址，请检查配置。');
      return;
    }

    try {
      setPendingChoice(choice.value);

      const response = await fetch(actionUrl, {
        method: block.verb || 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(defaultBody(choice, block)),
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || response.statusText);
      }

      toast.success(`操作已触发：${choice.label}`);
    } catch (error) {
      console.error('[CallbackBlock] 执行回调失败', error);
      toast.error(`回调触发失败：${(error as Error).message}`);
    } finally {
      setPendingChoice(null);
    }
  };

  if (!block.choices || block.choices.length === 0) {
    return null;
  }

  return (
    <div className="border-2 border-amber-200 dark:border-amber-900 rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow">
      <div className="bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-950 dark:to-orange-950 px-4 py-3 flex items-center justify-between border-b-2 border-amber-200 dark:border-amber-900">
        <div className="flex items-center gap-3">
          <div className="p-1.5 rounded-md bg-amber-100 dark:bg-amber-900 text-amber-700 dark:text-amber-200">
            <Zap className="h-4 w-4" />
          </div>
          <div>
            <div className="text-sm font-semibold text-amber-900 dark:text-amber-100">
              {block.title || '自动化操作'}
            </div>
            <div className="text-xs text-amber-600 dark:text-amber-400">
              选择操作后将调用 Robusta 回调接口
            </div>
          </div>
        </div>
        {block.verb && (
          <Badge className="text-xs bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-200 border-0">
            {block.verb}
          </Badge>
        )}
      </div>
      <div className="space-y-3 p-4 bg-white/90 dark:bg-slate-950/70">
        {block.description && (
          <p className="text-sm text-amber-800 dark:text-amber-200 bg-amber-50/80 dark:bg-amber-900/30 border border-amber-200 dark:border-amber-900 rounded-md px-3 py-2">
            {block.description}
          </p>
        )}
        {!actionUrl && (
          <div className="flex items-center gap-2 text-sm text-amber-700 dark:text-amber-300 bg-amber-50/80 dark:bg-amber-900/30 border border-amber-200 dark:border-amber-900 rounded-md px-3 py-2">
            <ShieldAlert className="h-4 w-4" />
            <span>未提供回调地址，按钮将不会触发任何操作。</span>
          </div>
        )}
        <div className="flex flex-wrap gap-3">
          {block.choices.map((choice) => (
            <Button
              key={choice.value}
              onClick={() => handleExecute(choice)}
              disabled={pendingChoice !== null}
              variant={choice.danger ? 'destructive' : 'default'}
              className={`min-w-[120px] justify-center ${
                choice.danger
                  ? 'bg-red-600 hover:bg-red-700 text-white'
                  : 'bg-amber-500/90 hover:bg-amber-500 text-white'
              }`}
            >
              {pendingChoice === choice.value ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  执行中...
                </>
              ) : (
                choice.label
              )}
            </Button>
          ))}
        </div>
      </div>
    </div>
  );
}
