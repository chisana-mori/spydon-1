'use client';

import React from 'react';
import { ListBlock as ListBlockType } from '../../types/enrichment';

interface ListBlockProps {
  block: ListBlockType;
}

export function ListBlock({ block }: ListBlockProps) {
  const ListTag = block.ordered ? 'ol' : 'ul';

  return (
    <ListTag className={`space-y-1 text-sm ${block.ordered ? 'list-decimal' : 'list-disc'} list-inside`}>
      {block.items.map((item, idx) => (
        <li key={idx}>
          {typeof item === 'object' ? JSON.stringify(item) : String(item)}
        </li>
      ))}
    </ListTag>
  );
}
