'use client';

import React from 'react';
import { Block } from '../../types/enrichment';
import { MarkdownBlock } from './MarkdownBlock';
import { JsonBlock } from './JsonBlock';
import { K8sDiffBlock } from './K8sDiffBlock';
import { ListBlock } from './ListBlock';
import { TableBlock } from './TableBlock';
import { LinksBlock } from './LinksBlock';
import { TextFileBlock } from './TextFileBlock';
import { GraphBlock } from './GraphBlock';
import { GenericBlock } from './GenericBlock';
import { KubernetesFieldsBlock } from './KubernetesFieldsBlock';
import { CallbackBlock } from './CallbackBlock';

interface BlockRendererProps {
  block: Block;
}

export function BlockRenderer({ block }: BlockRendererProps) {
  switch (block.type) {
    case 'markdown':
      return <MarkdownBlock block={block as any} />;
    
    case 'header':
      const headerBlock = block as any;
      const HeaderTag = `h${headerBlock.level || 3}` as keyof React.JSX.IntrinsicElements;
      return <HeaderTag className="text-lg font-semibold">{headerBlock.text}</HeaderTag>;
    
    case 'json':
      return <JsonBlock block={block as any} />;
    
    case 'k8s_diff':
    case 'diff':
      return <K8sDiffBlock block={block as any} />;
    
    case 'list':
      return <ListBlock block={block as any} />;
    
    case 'table':
      return <TableBlock block={block as any} />;
    
    case 'links':
      return <LinksBlock block={block as any} />;
    
    case 'text_file':
    case 'file':
      return <TextFileBlock block={block as any} />;
    
    case 'graph':
      return <GraphBlock block={block as any} />;

    case 'kubernetes_fields':
    case 'fields':
    case 'kubernetes_fields_block':
      return <KubernetesFieldsBlock block={block as any} />;

    case 'callback':
      return <CallbackBlock block={block as any} />;
    
    default:
      return <GenericBlock block={block as any} />;
  }
}
