'use client';

import React, { useState } from 'react';
import { JsonBlock as JsonBlockType } from '../../types/enrichment';
import { Button } from '@/components/ui/button';
import { ChevronDown, ChevronRight, Copy, Check } from 'lucide-react';
import { copyTextToClipboard } from '@/lib/clipboard';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import oneDark from 'react-syntax-highlighter/dist/esm/styles/prism/one-dark';

interface JsonBlockProps {
  block: JsonBlockType;
}

export function JsonBlock({ block }: JsonBlockProps) {
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);

  const jsonString = JSON.stringify(block.data, null, 2);
  const preview = jsonString.slice(0, 200) + (jsonString.length > 200 ? '...' : '');

  const handleCopy = async () => {
    const success = await copyTextToClipboard(jsonString);
    if (success) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } else {
      // 浏览器不支持复制或拒绝访问剪贴板
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
        {expanded ? (
          <SyntaxHighlighter
            language="json"
            style={oneDark}
            customStyle={{
              margin: 0,
              borderRadius: 0,
              fontSize: '13px',
              background: 'transparent',
            }}
          >
            {jsonString}
          </SyntaxHighlighter>
        ) : (
          <pre className="p-3 overflow-auto text-xs">
            <code>{preview}</code>
          </pre>
        )}
      </div>
    </div>
  );
}
