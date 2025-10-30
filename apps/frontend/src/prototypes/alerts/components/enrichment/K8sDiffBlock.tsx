'use client';

import React, { useMemo, useState } from 'react';
import { K8sDiffBlock as K8sDiffBlockType } from '../../types/enrichment';
import { ChevronDown, ChevronRight, Sparkles } from 'lucide-react';
import { Badge } from '@/components/ui/badge';

interface K8sDiffBlockProps {
  block: K8sDiffBlockType;
}

export function K8sDiffBlock({ block }: K8sDiffBlockProps) {
  const [expanded, setExpanded] = useState(true);

  const getOpColor = (op?: string) => {
    switch (op) {
      case 'add':
        return 'bg-green-500/10 text-green-700 dark:text-green-400';
      case 'remove':
        return 'bg-red-500/10 text-red-700 dark:text-red-400';
      case 'replace':
        return 'bg-yellow-500/10 text-yellow-700 dark:text-yellow-400';
      default:
        return 'bg-blue-500/10 text-blue-700 dark:text-blue-400';
    }
  };

  const interestingDiffs = useMemo(() => {
    const rawInteresting = block.raw?.interesting_diffs || block.raw?.highlights;
    if (!rawInteresting) return [];
    if (Array.isArray(rawInteresting)) return rawInteresting;
    if (typeof rawInteresting === 'object') {
      return Object.values(rawInteresting);
    }
    return [];
  }, [block]);

  const renderValue = (value: unknown) => {
    if (value === undefined) {
      return '—';
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

  return (
    <div className="border-2 border-orange-200 dark:border-orange-900 rounded-lg overflow-hidden shadow-sm">
      <div className="bg-gradient-to-r from-orange-50 to-amber-50 dark:from-orange-950 dark:to-amber-950 px-4 py-3 border-b-2 border-orange-200 dark:border-orange-900">
        <button
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-2 text-sm font-semibold text-orange-900 dark:text-orange-100 hover:text-orange-600 dark:hover:text-orange-300 transition-colors"
        >
          {expanded ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
          Kubernetes 差异 · {block.diffs.length} 处变更
        </button>
      </div>
      {interestingDiffs.length > 0 && (
        <div className="px-4 py-3 bg-amber-50/70 dark:bg-amber-950/40 border-b border-orange-200/70 dark:border-orange-900/50">
          <div className="flex items-center gap-2 text-xs font-medium text-amber-700 dark:text-amber-200 uppercase tracking-wider">
            <Sparkles className="h-3.5 w-3.5" />
            关键变更
          </div>
          <ul className="mt-2 space-y-1 text-xs text-amber-700/90 dark:text-amber-200/90 list-disc list-inside">
            {interestingDiffs.map((item: any, idx: number) => (
              <li key={idx}>{typeof item === 'string' ? item : JSON.stringify(item)}</li>
            ))}
          </ul>
        </div>
      )}
      {expanded && (
        <div className="divide-y divide-orange-100 dark:divide-orange-900">
          {block.diffs.map((diff, idx) => (
            <div key={idx} className="p-4 space-y-3 bg-white/95 dark:bg-slate-950/60">
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant="outline" className={getOpColor(diff.op)}>
                  {diff.op || 'change'}
                </Badge>
                <code className="text-[11px] bg-orange-100/70 dark:bg-orange-900/40 text-orange-700 dark:text-orange-200 px-2 py-1 rounded-md font-mono">
                  {diff.path.join('.')}
                </code>
              </div>
              <div className="grid gap-3 lg:grid-cols-2 text-xs">
                {diff.other_value !== undefined && (
                  <div className="space-y-2">
                    <div className="text-[11px] font-semibold uppercase tracking-wide text-red-600 dark:text-red-300">
                      原值
                    </div>
                    <pre className="bg-red-500/10 border border-red-500/30 text-red-900 dark:text-red-200 dark:bg-red-950/40 rounded-md p-3 overflow-auto whitespace-pre-wrap">
                      {renderValue(diff.other_value)}
                    </pre>
                  </div>
                )}
                {diff.value !== undefined && (
                  <div className="space-y-2">
                    <div className="text-[11px] font-semibold uppercase tracking-wide text-green-600 dark:text-green-300">
                      现值
                    </div>
                    <pre className="bg-green-500/10 border border-green-500/30 text-green-900 dark:text-green-200 dark:bg-green-950/40 rounded-md p-3 overflow-auto whitespace-pre-wrap">
                      {renderValue(diff.value)}
                    </pre>
                  </div>
                )}
                {diff.other_value === undefined && diff.value === undefined && (
                  <div className="col-span-full text-xs text-muted-foreground italic">
                    未提供具体的旧值/新值，仅记录了变更类型。
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
