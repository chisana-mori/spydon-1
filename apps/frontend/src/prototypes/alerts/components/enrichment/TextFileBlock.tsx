'use client';

import React, { useState, useMemo } from 'react';
import dynamic from 'next/dynamic';
import { TextFileBlock as TextFileBlockType } from '../../types/enrichment';
import { Button } from '@/components/ui/button';
import { ChevronDown, ChevronRight, Download, FileImage, WrapText, AlignLeft } from 'lucide-react';

// 动态导入 LogViewer 包装器
const LogViewerWrapper = dynamic(
  () => import('./LogViewerWrapper').then(mod => ({ default: mod.LogViewerWrapper })),
  {
    ssr: false,
    loading: () => <div className="text-sm text-muted-foreground p-4">加载日志查看器...</div>
  }
);

interface TextFileBlockProps {
  block: TextFileBlockType;
}

export function TextFileBlock({ block }: TextFileBlockProps) {
  const [expanded, setExpanded] = useState(false);
  const [wrapLines, setWrapLines] = useState(true);
  const maxPreviewLines = 10;

  // 检测内容是否为 base64 编码
  const isBase64 = (str: string): boolean => {
    if (!str) return false;
    const base64Regex = /^[A-Za-z0-9+/]+={0,2}$/;
    return base64Regex.test(str.trim()) && str.length % 4 === 0;
  };

  // 检测文件是否为图片
  const isImageFile = (filename?: string): boolean => {
    if (!filename) return false;
    const imageExtensions = ['.png', '.jpg', '.jpeg', '.gif', '.svg', '.webp', '.bmp'];
    return imageExtensions.some(ext => filename.toLowerCase().endsWith(ext));
  };

  // 解码 base64 内容
  const decodedContent = useMemo(() => {
    const content = block.contents || '';
    if (isBase64(content)) {
      try {
        return atob(content);
      } catch (e) {
        console.error('Failed to decode base64:', e);
        return content;
      }
    }
    return content;
  }, [block.contents]);

  // 规范化换行符（处理内容中为字面量"\n"的情况和 CR/LF 差异）
  const normalizedContent = useMemo(() => {
    let s = decodedContent || '';
    if (!s) return '';
    // 若存在字面量 \n 或 \r\n，则转换为真实换行
    if (s.includes('\\n') || s.includes('\\r')) {
      s = s.replace(/\\r\\n/g, '\n').replace(/\\n/g, '\n').replace(/\\r/g, '\n');
    }
    // 统一 CRLF/CR 到 LF
    s = s.replace(/\r\n?/g, '\n');
    return s;
  }, [decodedContent]);


  // 判断是否为图片
  const isImage = isImageFile(block.filename);

  // 获取图片的 MIME 类型
  const getImageMimeType = (filename?: string): string => {
    if (!filename) return 'image/png';
    const ext = filename.toLowerCase().split('.').pop();
    const mimeTypes: Record<string, string> = {
      'png': 'image/png',
      'jpg': 'image/jpeg',
      'jpeg': 'image/jpeg',
      'gif': 'image/gif',
      'svg': 'image/svg+xml',
      'webp': 'image/webp',
      'bmp': 'image/bmp'
    };
    return mimeTypes[ext || ''] || 'image/png';
  };

  // 生成图片 data URL
  const imageDataUrl = useMemo(() => {
    if (!isImage || !block.contents) return null;
    const mimeType = getImageMimeType(block.filename);
    const base64Content = isBase64(block.contents || '') ? block.contents : btoa(block.contents || '');
    return `data:${mimeType};base64,${base64Content}`;
  }, [isImage, block.contents, block.filename]);

  // 检测是否为日志文件
  const isLogFile = useMemo(() => {
    if (!block.filename) return false;
    const filename = block.filename.toLowerCase();
    return filename.endsWith('.log') ||
      filename.includes('log') ||
      filename.endsWith('.txt') && decodedContent.includes('[');
  }, [block.filename, decodedContent]);

  const lines = normalizedContent.split('\n');
  const hasMore = lines.length > maxPreviewLines;

  const handleDownload = () => {
    const content = decodedContent;
    const mimeType = isImage ? getImageMimeType(block.filename) : 'text/plain';
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = block.filename || 'file.txt';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  // 如果是图片，渲染图片
  if (isImage && imageDataUrl) {
    return (
      <div className="border-2 border-blue-200 dark:border-blue-800 rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow">
        <div className="bg-gradient-to-r from-blue-50 to-cyan-50 dark:from-blue-950 dark:to-cyan-950 px-4 py-3 flex items-center justify-between border-b-2 border-blue-200 dark:border-blue-800">
          <div className="flex items-center gap-3">
            <div className="p-1.5 rounded-md bg-blue-100 dark:bg-blue-900">
              <FileImage className="h-4 w-4 text-blue-600 dark:text-blue-300" />
            </div>
            <div>
              <div className="text-sm font-semibold text-blue-900 dark:text-blue-100">
                {block.filename || 'Image'}
              </div>
              <div className="text-xs text-blue-600 dark:text-blue-400">
                图片文件
              </div>
            </div>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={handleDownload}
            className="h-8 px-3 bg-white dark:bg-slate-900 hover:bg-blue-50 dark:hover:bg-blue-950 border-blue-200 dark:border-blue-800"
          >
            <Download className="h-3 w-3 mr-1" />
            下载
          </Button>
        </div>
        <div className="bg-gradient-to-br from-slate-50 to-blue-50 dark:from-slate-950 dark:to-blue-950 p-6 flex justify-center">
          <img
            src={imageDataUrl}
            alt={block.filename || 'Image'}
            className="max-w-full h-auto rounded-lg shadow-xl border-2 border-blue-100 dark:border-blue-900"
            style={{ maxHeight: '600px' }}
          />
        </div>
      </div>
    );
  }

  // 渲染文本文件
  return (
    <div className="border-2 border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow">
      <div className="bg-gradient-to-r from-slate-50 to-gray-50 dark:from-slate-950 dark:to-gray-950 px-4 py-3 flex items-center justify-between border-b-2 border-slate-200 dark:border-slate-800">
        <button
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-3 text-sm font-medium hover:text-primary transition-colors group"
        >
          <div className="p-1.5 rounded-md bg-slate-100 dark:bg-slate-900 group-hover:bg-slate-200 dark:group-hover:bg-slate-800 transition-colors">
            {expanded ? (
              <ChevronDown className="h-4 w-4 text-slate-600 dark:text-slate-300" />
            ) : (
              <ChevronRight className="h-4 w-4 text-slate-600 dark:text-slate-300" />
            )}
          </div>
          <div>
            <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">
              {block.filename || 'Text File'}
            </div>
            <div className="text-xs text-slate-600 dark:text-slate-400">
              {lines.length} 行 {expanded ? '(已展开)' : '(点击展开)'}
            </div>
          </div>
        </button>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setWrapLines(!wrapLines)}
            className="h-8 px-2"
            title={wrapLines ? '禁用自动换行' : '启用自动换行'}
          >
            {wrapLines ? (
              <WrapText className="h-3 w-3" />
            ) : (
              <AlignLeft className="h-3 w-3" />
            )}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleDownload}
            className="h-8 px-3 bg-white dark:bg-slate-900 hover:bg-slate-50 dark:hover:bg-slate-950 border-slate-200 dark:border-slate-800"
          >
            <Download className="h-3 w-3 mr-1" />
            下载
          </Button>
        </div>
      </div>
      <div className="bg-slate-950 dark:bg-slate-900 overflow-hidden">
        {isLogFile && expanded ? (
          // 使用专业的日志查看器
          <div className="h-[500px]">
            <LogViewerWrapper text={normalizedContent} height={500} wrapLines={wrapLines} />
          </div>
        ) : (
          // 使用简单的表格视图
          <>
            <div className="overflow-auto max-h-[500px]" style={{ scrollbarGutter: 'stable' }}>
              <table className="w-full border-collapse">
                <tbody>
                  {(expanded ? lines : lines.slice(0, maxPreviewLines)).map((line, idx) => (
                    <tr key={idx} className="hover:bg-slate-800/50 group">
                      <td className="text-slate-500 select-none text-right pr-4 pl-4 py-0.5 text-xs font-mono align-top border-r border-slate-800 sticky left-0 bg-slate-950 dark:bg-slate-900">
                        {idx + 1}
                      </td>
                      <td className={`text-slate-50 dark:text-slate-100 pl-4 pr-4 py-0.5 text-xs font-mono ${wrapLines ? 'whitespace-pre-wrap break-all' : 'whitespace-pre'}`}>
                        {line || '\u00A0'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {hasMore && !expanded && (
              <div className="px-4 py-3 text-xs text-slate-400 dark:text-slate-500 border-t border-slate-800 dark:border-slate-700 bg-slate-900 dark:bg-slate-950">
                <div className="flex items-center gap-2">
                  <div className="flex-1 border-t border-dashed border-slate-700"></div>
                  <span>还有 {lines.length - maxPreviewLines} 行未显示</span>
                  <div className="flex-1 border-t border-dashed border-slate-700"></div>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
