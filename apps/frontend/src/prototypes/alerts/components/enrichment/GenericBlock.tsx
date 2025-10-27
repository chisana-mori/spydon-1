'use client';

import React, { useState } from 'react';
import { GenericBlock as GenericBlockType } from '../../types/enrichment';
import { Button } from '@/components/ui/button';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { AlertCircle } from 'lucide-react';

interface GenericBlockProps {
  block: GenericBlockType;
}

export function GenericBlock({ block }: GenericBlockProps) {
  const [expanded, setExpanded] = useState(false);
  const jsonString = JSON.stringify(block.raw, null, 2);

  return (
    <div className="space-y-2">
      <Alert>
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>
          Unknown block type. Showing raw data.
        </AlertDescription>
      </Alert>
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
            Raw Data
          </button>
        </div>
        {expanded && (
          <div className="bg-slate-950 text-slate-50">
            <pre className="p-3 overflow-auto text-xs max-h-96">
              <code>{jsonString}</code>
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}
