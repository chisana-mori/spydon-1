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
    <div className="space-y-2">
      {block.table_name && (
        <div className="text-sm font-medium">{block.table_name}</div>
      )}
      <div className="border rounded-lg overflow-hidden">
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                {block.headers.map((header, idx) => (
                  <TableHead key={idx} className="whitespace-nowrap">
                    {header}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {block.rows.map((row, rowIdx) => (
                <TableRow key={rowIdx}>
                  {row.map((cell, cellIdx) => {
                    const columnName = block.headers[cellIdx];
                    return (
                      <TableCell key={cellIdx} className="max-w-md">
                        <div className="truncate" title={formatCell(cell, columnName)}>
                          {formatCell(cell, columnName)}
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
