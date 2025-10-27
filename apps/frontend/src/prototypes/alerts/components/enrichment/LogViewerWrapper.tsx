'use client';

import React from 'react';
import { LazyLog } from '@melloware/react-logviewer';

interface LogViewerWrapperProps {
  text: string;
  height?: number;
  wrapLines?: boolean;
}

export function LogViewerWrapper({ text, height = 500, wrapLines = true }: LogViewerWrapperProps) {
  return (
    <div className="log-viewer-container">
      <LazyLog
        text={text}
        height={height}
        enableSearch
        selectableLines
        enableHotKeys
        enableLinks
        caseInsensitive
        wrapLines={wrapLines}
        extraLines={1}
        rowHeight={20}
        style={{
          backgroundColor: '#0f172a',
          color: '#e2e8f0',
          fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace',
          fontSize: '13px',
          lineHeight: '1.5',
          padding: '8px',
        }}
      />
    </div>
  );
}
