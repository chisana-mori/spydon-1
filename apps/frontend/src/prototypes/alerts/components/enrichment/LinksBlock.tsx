'use client';

import React from 'react';
import { LinksBlock as LinksBlockType } from '../../types/enrichment';
import { ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface LinksBlockProps {
  block: LinksBlockType;
}

export function LinksBlock({ block }: LinksBlockProps) {
  return (
    <div className="space-y-2">
      <div className="text-sm font-medium">Related Links</div>
      <div className="flex flex-wrap gap-2">
        {block.links.map((link, idx) => (
          <Button
            key={idx}
            variant="outline"
            size="sm"
            asChild
          >
            <a
              href={link.url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2"
            >
              <ExternalLink className="h-3 w-3" />
              {link.name || link.url}
            </a>
          </Button>
        ))}
      </div>
    </div>
  );
}
