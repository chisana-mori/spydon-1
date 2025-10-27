'use client';

import React, { useState } from 'react';
import { K8sDiffBlock as K8sDiffBlockType } from '../../types/enrichment';
import { ChevronDown, ChevronRight } from 'lucide-react';
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

  return (
    <div className="border rounded-lg overflow-hidden">
      <div className="bg-muted/50 px-3 py-2">
        <button
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-2 text-sm font-medium hover:text-primary"
        >
          {expanded ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
          Kubernetes Diff ({block.diffs.length} changes)
        </button>
      </div>
      {expanded && (
        <div className="divide-y">
          {block.diffs.map((diff, idx) => (
            <div key={idx} className="p-3 space-y-2">
              <div className="flex items-center gap-2">
                <Badge variant="outline" className={getOpColor(diff.op)}>
                  {diff.op || 'change'}
                </Badge>
                <code className="text-xs text-muted-foreground">
                  {diff.path.join('.')}
                </code>
              </div>
              <div className="grid grid-cols-2 gap-2 text-xs">
                {diff.other_value !== undefined && (
                  <div className="space-y-1">
                    <div className="text-muted-foreground">Old:</div>
                    <pre className="bg-red-500/5 border border-red-500/20 rounded p-2 overflow-auto">
                      {JSON.stringify(diff.other_value, null, 2)}
                    </pre>
                  </div>
                )}
                {diff.value !== undefined && (
                  <div className="space-y-1">
                    <div className="text-muted-foreground">New:</div>
                    <pre className="bg-green-500/5 border border-green-500/20 rounded p-2 overflow-auto">
                      {JSON.stringify(diff.value, null, 2)}
                    </pre>
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
