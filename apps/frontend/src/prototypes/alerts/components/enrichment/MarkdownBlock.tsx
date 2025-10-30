'use client';

import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import oneDark from 'react-syntax-highlighter/dist/esm/styles/prism/one-dark';
import { MarkdownBlock as MarkdownBlockType } from '../../types/enrichment';

interface MarkdownBlockProps {
  block: MarkdownBlockType;
}

export function MarkdownBlock({ block }: MarkdownBlockProps) {
  return (
    <div className="prose prose-sm max-w-none dark:prose-invert">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath]}
        components={{
          code({ node, inline, className, children, ...props }) {
            const match = /language-(\w+)/.exec(className || '');
            if (inline) {
              return (
                <code
                  className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono text-primary"
                  {...props}
                >
                  {children}
                </code>
              );
            }
            return (
              <SyntaxHighlighter
                PreTag="div"
                language={match?.[1] || 'plaintext'}
                style={oneDark}
                customStyle={{
                  margin: '12px 0',
                  borderRadius: '8px',
                  fontSize: '13px',
                }}
                {...props}
              >
                {String(children).replace(/\n$/, '')}
              </SyntaxHighlighter>
            );
          },
          table({ children }) {
            return (
              <div className="overflow-x-auto rounded-lg border border-border/50">
                <table className="w-full border-collapse">{children}</table>
              </div>
            );
          },
          th({ children }) {
            return (
              <th className="border-b border-border/60 bg-muted/70 px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide">
                {children}
              </th>
            );
          },
          td({ children }) {
            return (
              <td className="border-b border-border/40 px-3 py-2 text-sm">
                {children}
              </td>
            );
          },
        }}
      >
        {block.text}
      </ReactMarkdown>
    </div>
  );
}
