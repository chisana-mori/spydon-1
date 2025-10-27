'use client';

import React from 'react';
import { Enrichment } from '../../types/enrichment';
import { BlockRenderer } from './BlockRenderer';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

interface EnrichmentRendererProps {
  enrichment: Enrichment;
  index?: number;
}

export function EnrichmentRenderer({ enrichment, index }: EnrichmentRendererProps) {
  const title = enrichment.title || enrichment.enrichment_type || `Enrichment ${(index || 0) + 1}`;

  return (
    <Card className="overflow-hidden">
      <CardHeader className="pb-3 bg-muted/30">
        <div className="flex items-start justify-between gap-4">
          <CardTitle className="text-base font-semibold">
            {title}
          </CardTitle>
          {enrichment.enrichment_type && (
            <Badge variant="secondary" className="text-xs">
              {enrichment.enrichment_type}
            </Badge>
          )}
        </div>
        {enrichment.annotations && Object.keys(enrichment.annotations).length > 0 && (
          <div className="flex flex-wrap gap-2 mt-2">
            {Object.entries(enrichment.annotations).map(([key, value]) => (
              <Badge key={key} variant="outline" className="text-xs">
                {key}: {value}
              </Badge>
            ))}
          </div>
        )}
      </CardHeader>
      <CardContent className="pt-4 space-y-4">
        {enrichment.blocks.map((block) => (
          <BlockRenderer key={block.id} block={block} />
        ))}
      </CardContent>
    </Card>
  );
}
