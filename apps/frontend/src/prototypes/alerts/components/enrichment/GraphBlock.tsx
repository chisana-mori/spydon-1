'use client';

import React, { useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { Download, FileImage, AlertCircle } from 'lucide-react';

interface GraphBlockData {
  filename?: string;
  contents?: string;
  hidden?: boolean;
  html_class?: string | null;
}

interface GraphBlockProps {
  block: {
    type: string;
    filename?: string;
    contents?: string;
    data?: GraphBlockData;
    raw?: any;
  };
}

export function GraphBlock({ block }: GraphBlockProps) {
  // 从不同的可能位置获取数据
  const graphData = useMemo(() => {
    if (block.data) return block.data;
    if (block.filename && block.contents) {
      return {
        filename: block.filename,
        contents: block.contents,
        hidden: false,
        html_class: null
      };
    }
    return null;
  }, [block]);

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
    if (!graphData?.contents) return null;
    const mimeType = getImageMimeType(graphData.filename);
    const base64Content = isBase64(graphData.contents) ? graphData.contents : btoa(graphData.contents);
    return `data:${mimeType};base64,${base64Content}`;
  }, [graphData]);

  const handleDownload = () => {
    if (!graphData?.contents) return;

    const content = isBase64(graphData.contents) ? atob(graphData.contents) : graphData.contents;
    const mimeType = getImageMimeType(graphData.filename);
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = graphData.filename || 'graph.png';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  if (!graphData || graphData.hidden) {
    return null;
  }

  const isImage = isImageFile(graphData.filename);

  return (
    <div className="border-2 border-purple-200 dark:border-purple-800 rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow">
      <div className="bg-gradient-to-r from-purple-50 to-pink-50 dark:from-purple-950 dark:to-pink-950 px-4 py-3 flex items-center justify-between border-b-2 border-purple-200 dark:border-purple-800">
        <div className="flex items-center gap-3">
          <div className="p-1.5 rounded-md bg-purple-100 dark:bg-purple-900">
            <FileImage className="h-4 w-4 text-purple-600 dark:text-purple-300" />
          </div>
          <div>
            <div className="text-sm font-semibold text-purple-900 dark:text-purple-100">
              {graphData.filename || 'Graph'}
            </div>
            <div className="text-xs text-purple-600 dark:text-purple-400">
              图表可视化
            </div>
          </div>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={handleDownload}
          className="h-8 px-3 bg-white dark:bg-slate-900 hover:bg-purple-50 dark:hover:bg-purple-950 border-purple-200 dark:border-purple-800"
        >
          <Download className="h-3 w-3 mr-1" />
          下载
        </Button>
      </div>
      {isImage && imageDataUrl ? (
        <div className="bg-gradient-to-br from-slate-50 to-purple-50 dark:from-slate-950 dark:to-purple-950 p-6 flex justify-center">
          <img
            src={imageDataUrl}
            alt={graphData.filename || 'Graph'}
            className={`max-w-full h-auto rounded-lg shadow-xl border-2 border-purple-100 dark:border-purple-900 ${graphData.html_class || ''}`}
            style={{ maxHeight: '600px' }}
          />
        </div>
      ) : (
        <div className="p-6 text-center">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-red-100 dark:bg-red-900 mb-3">
            <AlertCircle className="h-8 w-8 text-red-600 dark:text-red-300" />
          </div>
          <div className="text-sm font-medium text-red-900 dark:text-red-100">
            无法显示图表内容
          </div>
          <div className="text-xs text-red-600 dark:text-red-400 mt-1">
            图表数据格式不正确或已损坏
          </div>
        </div>
      )}
    </div>
  );
}
