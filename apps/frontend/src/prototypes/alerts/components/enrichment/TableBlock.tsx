'use client';

import React from 'react';
import { TableBlock as TableBlockType } from '../../types/enrichment';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface TableBlockProps {
  block: TableBlockType;
}

export function TableBlock({ block }: TableBlockProps) {
  // 获取 column_renderers 配置
  const columnRenderers = (block.raw as any)?.column_renderers || {};

  // 格式化单元格内容
  const formatCell = (cell: any, columnName: string): string => {
    if (cell === null || cell === undefined) {
      return '';
    }

    // 检查是否有特殊渲染器
    const renderer = columnRenderers[columnName];

    if (renderer === 'DATETIME') {
      // 处理时间戳（毫秒）
      if (typeof cell === 'number') {
        try {
          const date = new Date(cell);
          return date.toLocaleString();
        } catch (e) {
          return String(cell);
        }
      }
    }

    // 默认处理
    if (typeof cell === 'object') {
      return JSON.stringify(cell);
    }

    return String(cell);
  };

  return (
    <div className="space-y-3">
      {block.table_name && (
        <div className="flex items-center justify-between">
          <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">
            {block.table_name}
          </div>
          <div className="text-xs text-muted-foreground">
            {block.rows.length} 行 · {block.headers.length} 列
          </div>
        </div>
      )}
      <div className="border-2 border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden shadow-sm">
        <div className="max-h-[480px] overflow-auto">
          <Table className="min-w-full">
            <TableHeader className="sticky top-0 z-10 bg-muted">
              <TableRow className="bg-muted/80 backdrop-blur supports-[backdrop-filter]:bg-muted/60">
                {block.headers.map((header, idx) => (
                  <TableHead
                    key={idx}
                    className="whitespace-nowrap text-xs font-semibold uppercase tracking-wide text-muted-foreground border-r border-border/40 last:border-r-0"
                  >
                    {header}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {block.rows.map((row, rowIdx) => (
                <TableRow
                  key={rowIdx}
                  className={rowIdx % 2 === 0 ? 'bg-muted/10' : 'bg-background'}
                >
                  {row.map((cell, cellIdx) => {
                    const columnName = block.headers[cellIdx];
                    const formatted = formatCell(cell, columnName);
                    const isLongText = formatted.length > 80 || formatted.includes('\n');
                    return (
                      <TableCell
                        key={cellIdx}
                        className="align-top border-r border-border/20 last:border-r-0"
                      >
                        <div
                          className={`text-xs text-foreground/90 ${
                            isLongText
                              ? 'whitespace-pre-wrap break-words font-mono'
                              : 'truncate'
                          }`}
                          title={formatted}
                        >
                          {formatted || '—'}
                        </div>
                      </TableCell>
                    );
                  })}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}
