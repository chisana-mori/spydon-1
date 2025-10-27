'use client';

import React from 'react';
import { MarkdownBlock as MarkdownBlockType } from '../../types/enrichment';

interface MarkdownBlockProps {
  block: MarkdownBlockType;
}

export function MarkdownBlock({ block }: MarkdownBlockProps) {
  return (
    <div className="prose prose-sm max-w-none dark:prose-invert">
      <div className="whitespace-pre-wrap text-sm leading-relaxed">
        {block.text}
      </div>
    </div>
  );
}
