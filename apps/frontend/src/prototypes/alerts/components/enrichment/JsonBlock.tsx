'use client';

import React, { useState } from 'react';
import { JsonBlock as JsonBlockType } from '../../types/enrichment';
import { Button } from '@/components/ui/button';
import { ChevronDown, ChevronRight, Copy, Check } from 'lucide-react';

interface JsonBlockProps {
  block: JsonBlockType;
}

export function JsonBlock({ block }: JsonBlockProps) {
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);

  const jsonString = JSON.stringify(block.data, null, 2);
  const preview = jsonString.slice(0, 200) + (jsonString.length > 200 ? '...' : '');

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(jsonString);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      // Copy failed
    }
  };

  return (
    <div className="border rounded-lg overflow-hidden">
      <div className="bg-muted/50 px-3 py-2 flex items-center justify-between">
        <button
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-2 text-sm font-medium hover:text-primary"
        >
          {expanded ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
          JSON Data
        </button>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleCopy}
          className="h-7 px-2"
        >
          {copied ? (
            <Check className="h-3 w-3" />
          ) : (
            <Copy className="h-3 w-3" />
          )}
        </Button>
      </div>
      <div className="bg-slate-950 text-slate-50">
        <pre className="p-3 overflow-auto text-xs">
          <code>{expanded ? jsonString : preview}</code>
        </pre>
      </div>
    </div>
  );
}
