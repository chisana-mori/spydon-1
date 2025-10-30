'use client';

import React, { useMemo } from 'react';
import { KubernetesFieldsBlock as KubernetesFieldsBlockType, KubernetesFieldDescriptor } from '../../types/enrichment';
import { Badge } from '@/components/ui/badge';

interface KubernetesFieldsBlockProps {
  block: KubernetesFieldsBlockType;
}

const stringifyValue = (value: unknown): string => {
  if (value === undefined || value === null) {
    return '';
  }
  if (typeof value === 'string') {
    return value;
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
};

const getPathSegments = (path: string): Array<string | number> => {
  const segments: Array<string | number> = [];
  path
    .split('.')
    .map((segment) => segment.trim())
    .filter(Boolean)
    .forEach((segment) => {
      const regex = /([^[\]]+)|\[(\d+)\]/g;
      let match;
      while ((match = regex.exec(segment)) !== null) {
        if (match[1]) {
          segments.push(match[1]);
        } else if (match[2]) {
          segments.push(Number(match[2]));
        }
      }
    });

  return segments;
};

const resolveValueFromObject = (obj: any, path: string): unknown => {
  if (!obj || !path) return undefined;

  return getPathSegments(path).reduce<unknown>((acc, segment) => {
    if (acc === undefined || acc === null) {
      return undefined;
    }
    if (typeof segment === 'number' && Array.isArray(acc)) {
      return acc[segment];
    }
    if (typeof segment === 'string' && typeof acc === 'object') {
      return (acc as Record<string, unknown>)[segment];
    }
    return undefined;
  }, obj);
};

const normalizeField = (
  field: KubernetesFieldDescriptor,
  block: KubernetesFieldsBlockType
): KubernetesFieldDescriptor => {
  const value =
    field.value !== undefined
      ? field.value
      : resolveValueFromObject(block.k8s_obj, field.path);

  return {
    ...field,
    value,
  };
};

export function KubernetesFieldsBlock({ block }: KubernetesFieldsBlockProps) {
  const normalizedFields = useMemo(
    () =>
      (block.fields || [])
        .map((field) => normalizeField(field, block))
        .filter((field) => field.path),
    [block]
  );

  if (!normalizedFields.length) {
    return null;
  }

  return (
    <div className="border-2 border-emerald-200 dark:border-emerald-900 rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow">
      <div className="bg-gradient-to-r from-emerald-50 to-teal-50 dark:from-emerald-950 dark:to-teal-950 px-4 py-3 flex items-center justify-between border-b-2 border-emerald-200 dark:border-emerald-900">
        <div className="flex items-center gap-3">
          <div className="p-1.5 rounded-md bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-200 font-semibold text-sm">
            K8s
          </div>
          <div>
            <div className="text-sm font-semibold text-emerald-900 dark:text-emerald-100">
              {block.title || 'Kubernetes 关键信息'}
            </div>
            <div className="text-xs text-emerald-600 dark:text-emerald-400">
              展示来自 Kubernetes 对象的主要字段
            </div>
          </div>
        </div>
        <Badge className="text-xs bg-emerald-100 text-emerald-700 dark:bg-emerald-900 dark:text-emerald-200 border-0">
          {normalizedFields.length} 个字段
        </Badge>
      </div>
      <div className="bg-white dark:bg-slate-950">
        <dl className="grid gap-4 p-4 md:grid-cols-2">
          {normalizedFields.map((field) => {
            const explanation =
              block.explanations?.[field.path] ||
              block.explanations?.[field.label || ''] ||
              block.explanations?.[field.path.replace(/\.\d+/g, '')];
            const stringValue = stringifyValue(field.value);
            const isMultiLine = stringValue.includes('\n');

            return (
              <div
                key={field.path}
                className="rounded-lg border border-emerald-100/80 dark:border-emerald-900/50 bg-emerald-50/40 dark:bg-emerald-950/30 p-4 space-y-2"
              >
                <div className="flex items-center justify-between gap-2">
                  <dt className="text-sm font-semibold text-emerald-900 dark:text-emerald-100">
                    {field.label || field.path}
                  </dt>
                  <code className="text-[11px] text-emerald-700 dark:text-emerald-300 bg-emerald-100/60 dark:bg-emerald-900/60 px-2 py-0.5 rounded">
                    {field.path}
                  </code>
                </div>
                <dd
                  className={`text-sm text-emerald-900/90 dark:text-emerald-100/90 border border-emerald-200/80 dark:border-emerald-900/50 bg-white/80 dark:bg-slate-950/40 rounded-md px-3 py-2 font-mono ${
                    isMultiLine ? 'whitespace-pre-wrap break-words' : 'break-all'
                  }`}
                >
                  {stringValue || '—'}
                </dd>
                {explanation && (
                  <p className="text-xs text-emerald-700 dark:text-emerald-300 bg-emerald-100/50 dark:bg-emerald-900/40 px-3 py-2 rounded-md">
                    {explanation}
                  </p>
                )}
              </div>
            );
          })}
        </dl>
      </div>
    </div>
  );
}
