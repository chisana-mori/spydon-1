'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import dynamic from 'next/dynamic';
import { appConfig } from '@/config';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { copyTextToClipboard } from '@/lib/clipboard';

import {
  Download,
  FileText,
  Loader2,
  AlertCircle,
  Copy,
  Check,
  RefreshCw,
  Code,
  Eye,
  ChevronDown,
  ChevronRight
} from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
// Enrichment 结构化渲染相关
import { parseFinding } from '@prototypes/alerts/lib/enrichment-parser';
import { EnrichmentRenderer } from '@prototypes/alerts/components/enrichment/EnrichmentRenderer';
import type { Finding } from '@prototypes/alerts/types/enrichment';

// 动态导入 JsonView 以避免 SSR 问题
const JsonView = dynamic(() => import('@uiw/react-json-view').then(mod => mod.default), {
  ssr: false,
  loading: () => <div className="text-sm text-muted-foreground">加载 JSON 查看器...</div>
});

interface RawPayloadViewerProps {
  rawPayloadKey?: string;
  alertId: string;
  alertTitle?: string;
  className?: string;
}

type DataFormat = 'markdown' | 'json' | 'yaml' | 'xml' | 'text';

function RawPayloadViewerComponent({
  rawPayloadKey,
  alertId,
  alertTitle,
  className
}: RawPayloadViewerProps) {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [showRawData, setShowRawData] = useState(false);
  const [finding, setFinding] = useState<Finding | null>(null);
  const [collapsedEnrichments, setCollapsedEnrichments] = useState<Set<number>>(new Set());

  // 检测数据格式
  const detectDataFormat = (content: string): DataFormat => {
    const trimmed = content.trim();

    // 优先检测 JSON（最常见的格式）
    if ((trimmed.startsWith('{') && trimmed.endsWith('}')) ||
      (trimmed.startsWith('[') && trimmed.endsWith(']'))) {
      try {
        JSON.parse(trimmed);
        return 'json';
      } catch {
        // JSON 解析失败，继续检测其他格式
      }
    }

    // 检测 XML
    if (trimmed.startsWith('<') && trimmed.endsWith('>') &&
      trimmed.includes('</')) {
      return 'xml';
    }

    // 检测 YAML（在 Markdown 之前检测，因为 YAML 更结构化）
    if (trimmed.includes('---\n') ||
      (trimmed.match(/^[a-zA-Z_][a-zA-Z0-9_]*:\s+/m) && !trimmed.includes('# '))) {
      return 'yaml';
    }

    // 检测 Markdown（放在最后，因为它的特征可能与其他格式重叠）
    if (trimmed.includes('# ') || trimmed.includes('## ') ||
      trimmed.includes('### ') || trimmed.includes('```')) {
      return 'markdown';
    }

    return 'text';
  };

  // 加载原始数据
  const loadRawData = useCallback(async () => {
    if (!rawPayloadKey && !alertId) return;

    setLoading(true);
    setError(null);

    try {
      // 使用后端 API 获取原始数据
      const response = await fetch(`${appConfig.apiBaseUrl}/alerts/${alertId}/raw-payload`, {
        credentials: 'include', // 携带Cookie进行认证
      });

      if (response.ok) {
        const contentType = response.headers.get('content-type') || 'text/plain';
        const payloadKey = response.headers.get('x-raw-payload-key') || rawPayloadKey || '';

        // 始终以文本形式获取数据
        const responseData = await response.text();

        const detectedFormat = detectDataFormat(responseData);

        setData({
          key: payloadKey,
          data: responseData,
          contentType,
          format: detectedFormat,
          size: new Blob([responseData]).size,
        });

        // 如果是 JSON，尝试解析为 Finding 并提取 enrichments 进行结构化展示
        if (detectedFormat === 'json') {
          try {
            const obj = JSON.parse(responseData);
            const parsedFinding = parseFinding(obj);
            setFinding(parsedFinding);
          } catch (e) {
            setFinding(null);
          }
        } else {
          setFinding(null);
        }
      } else {
        setError(`获取数据失败: ${response.status} ${response.statusText}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取数据时发生错误');
    } finally {
      setLoading(false);
    }
  }, [alertId, rawPayloadKey]);

  useEffect(() => {
    if (alertId) {
      loadRawData();
    }
  }, [alertId, loadRawData]);

  // 复制到剪贴板
  const copyToClipboard = async () => {
    if (!data) return;

    const success = await copyTextToClipboard(data.data);
    if (success) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } else {
      // 浏览器不支持复制或权限被拒绝
    }
  };

  // 下载原始数据
  const downloadRawData = () => {
    if (!data) return;

    const blob = new Blob([data.data], { type: data.contentType || 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `alert-${alertId}-raw-data.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  // 切换 enrichment 折叠状态
  const toggleEnrichment = (index: number) => {
    setCollapsedEnrichments(prev => {
      const newSet = new Set(prev);
      if (newSet.has(index)) {
        newSet.delete(index);
      } else {
        newSet.add(index);
      }
      return newSet;
    });
  };

  // 全部展开/折叠
  const toggleAllEnrichments = () => {
    if (!finding?.enrichments) return;

    if (collapsedEnrichments.size === finding.enrichments.length) {
      setCollapsedEnrichments(new Set());
    } else {
      setCollapsedEnrichments(new Set(finding.enrichments.map((_, idx) => idx)));
    }
  };

  // 格式化 JSON 数据
  const formatJSON = (content: string): string => {
    try {
      const parsed = JSON.parse(content);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return content;
    }
  };

  // 解析 JSON 数据
  const parseJSON = (content: string): any | null => {
    try {
      return JSON.parse(content);
    } catch {
      return null;
    }
  };

  // 渲染格式化的 JSON（使用内置方案）
  const renderFormattedJSON = (jsonData: any): React.ReactNode => {
    return (
      <pre className="whitespace-pre-wrap font-mono text-sm text-foreground leading-relaxed">
        {JSON.stringify(jsonData, null, 2)}
      </pre>
    );
  };

  const renderMarkdownContent = (content: string): React.ReactNode => {
    return (
      <div className="prose prose-sm max-w-none break-words">
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          components={{
            code: ({ inline, children, ...props }: any) =>
              inline ? (
                <code
                  {...props}
                  className="rounded bg-muted px-1 text-sm font-mono text-foreground"
                >
                  {children}
                </code>
              ) : (
                <pre
                  {...props}
                  className="bg-slate-900 text-white rounded-md p-3 text-sm font-mono overflow-x-auto whitespace-pre-wrap"
                >
                  {children}
                </pre>
              )
          }}
        >
          {content}
        </ReactMarkdown>
      </div>
    );
  };

  // 获取格式化的数据
  const getFormattedData = (format: DataFormat, content: string): React.ReactNode => {
    switch (format) {
      case 'markdown':
        return renderMarkdownContent(content);
      case 'json':
        const jsonData = parseJSON(content);

        if (jsonData) {
          // 使用 JsonView 组件
          return (
            <JsonView
              value={jsonData}
              collapsed={false}
              displayDataTypes={false}
              displayObjectSize={true}
              enableClipboard={true}
              style={{
                backgroundColor: 'transparent',
                fontSize: '13px',
                fontFamily: 'monospace',
              }}
            />
          );
        }
        // 如果解析失败，尝试格式化
        return <pre className="whitespace-pre-wrap font-mono text-sm leading-relaxed">{formatJSON(content)}</pre>;
      case 'yaml':
      case 'xml':
      case 'text':
      default:
        return <pre className="whitespace-pre-wrap font-mono text-sm leading-relaxed">{content}</pre>;
    }
  };

  const formattedContent = useMemo(() => {
    if (!data) {
      return null;
    }
    return getFormattedData(data.format, data.data);
  }, [data]);

  // 如果没有原始数据键且没有告警ID，不显示组件
  if (!rawPayloadKey && !alertId) {
    return null;
  }

  return (
    <Card className={className}>
      <CardHeader className="pb-3 bg-gradient-to-r from-slate-50 to-blue-50 dark:from-slate-900 dark:to-blue-950 border-b-2 border-blue-100 dark:border-blue-900">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-3">
            <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-900">
              <FileText className="h-5 w-5 text-blue-600 dark:text-blue-300" />
            </div>
            <div>
              <div className="font-bold text-slate-900 dark:text-slate-100">上下文摘要</div>
              {alertTitle && (
                <div className="text-xs font-normal text-slate-600 dark:text-slate-400 mt-0.5">
                  {alertTitle}
                </div>
              )}
            </div>
          </CardTitle>
          <div className="flex items-center gap-2">
            {data && (
              <>
                <Badge className="text-xs font-semibold bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 border-0">
                  {data.format?.toUpperCase() || 'TEXT'}
                </Badge>
                <Badge className="text-xs font-semibold bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300 border-0">
                  {(data.size / 1024).toFixed(1)} KB
                </Badge>
              </>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent>
        {loading && (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin mr-2" />
            <span>加载原始数据中...</span>
          </div>
        )}

        {error && (
          <Alert variant="destructive" className="mb-4">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              {error}
              <Button
                variant="outline"
                size="sm"
                onClick={loadRawData}
                className="ml-2"
              >
                <RefreshCw className="h-4 w-4 mr-1" />
                重试
              </Button>
            </AlertDescription>
          </Alert>
        )}

        {data && !loading && (
          <div className="space-y-4">
            {/* 操作按钮 */}
            <div className="flex items-center justify-between gap-2 flex-wrap p-3 bg-muted/50 rounded-lg border border-border/50">
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={copyToClipboard}
                  className={`flex items-center gap-2 transition-all ${copied
                      ? 'bg-green-50 border-green-200 text-green-700 dark:bg-green-950 dark:border-green-800 dark:text-green-300'
                      : 'hover:bg-blue-50 hover:border-blue-200 dark:hover:bg-blue-950 dark:hover:border-blue-800'
                    }`}
                >
                  {copied ? (
                    <>
                      <Check className="h-4 w-4" />
                      已复制
                    </>
                  ) : (
                    <>
                      <Copy className="h-4 w-4" />
                      复制
                    </>
                  )}
                </Button>

                <Button
                  variant="outline"
                  size="sm"
                  onClick={downloadRawData}
                  className="flex items-center gap-2 hover:bg-purple-50 hover:border-purple-200 dark:hover:bg-purple-950 dark:hover:border-purple-800"
                >
                  <Download className="h-4 w-4" />
                  下载
                </Button>

                <Button
                  variant="outline"
                  size="sm"
                  onClick={loadRawData}
                  className="flex items-center gap-2 hover:bg-amber-50 hover:border-amber-200 dark:hover:bg-amber-950 dark:hover:border-amber-800"
                >
                  <RefreshCw className="h-4 w-4" />
                  刷新
                </Button>
              </div>

              {/* 右上角原始数据按钮 */}
              {data.format === 'json' && (
                <Button
                  variant={showRawData ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setShowRawData(!showRawData)}
                  className={`flex items-center gap-2 ${showRawData
                      ? 'bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700'
                      : 'hover:bg-slate-50 hover:border-slate-300 dark:hover:bg-slate-900 dark:hover:border-slate-700'
                    }`}
                >
                  <Code className="h-4 w-4" />
                  原始数据
                </Button>
              )}
            </div>

            {/* 数据展示区域 */}
            <div className="border rounded-lg overflow-hidden bg-muted/30">
              <div className="p-4 text-sm max-h-[600px] overflow-auto">
                {showRawData ? (
                  // 显示原始 JSON 数据
                  <div className="raw-json-view">
                    {data.format === 'json' && parseJSON(data.data) ? (
                      <JsonView
                        value={parseJSON(data.data)}
                        collapsed={false}
                        displayDataTypes={false}
                        displayObjectSize={true}
                        enableClipboard={true}
                        style={{
                          backgroundColor: 'transparent',
                          fontSize: '13px',
                          fontFamily: 'monospace',
                        }}
                      />
                    ) : (
                      <pre className="whitespace-pre-wrap break-words font-mono text-xs text-foreground">
                        {data.data}
                      </pre>
                    )}
                  </div>
                ) : (
                  // 默认显示 enrichments
                  <div className="enrichments-view space-y-0">
                    {finding && finding.enrichments && finding.enrichments.length > 0 ? (
                      <>
                        {/* 全部展开/折叠按钮 */}
                        <div className="mb-4 flex items-center justify-between p-3 bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-950/30 dark:to-purple-950/30 rounded-lg border border-blue-200/50 dark:border-blue-800/50">
                          <div className="flex items-center gap-2">
                            <div className="flex items-center justify-center w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900">
                              <FileText className="h-4 w-4 text-blue-600 dark:text-blue-300" />
                            </div>
                            <div>
                              <div className="text-sm font-semibold text-blue-900 dark:text-blue-100">
                                共 {finding.enrichments.length} 条线索
                              </div>
                              <div className="text-xs text-blue-600 dark:text-blue-400">
                                点击展开查看详细信息
                              </div>
                            </div>
                          </div>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={toggleAllEnrichments}
                            className="h-8 text-xs font-medium bg-white dark:bg-slate-900 hover:bg-blue-50 dark:hover:bg-blue-950 border-blue-200 dark:border-blue-800"
                          >
                            {collapsedEnrichments.size === finding.enrichments.length ? (
                              <>
                                <ChevronDown className="h-3 w-3 mr-1" />
                                全部展开
                              </>
                            ) : (
                              <>
                                <ChevronRight className="h-3 w-3 mr-1" />
                                全部折叠
                              </>
                            )}
                          </Button>
                        </div>

                        {/* Enrichment 列表 */}
                        {finding.enrichments.map((enrichment, idx) => {
                          const isCollapsed = collapsedEnrichments.has(idx);

                          // 根据 enrichment_type 设置颜色主题
                          const getTypeColor = (type?: string) => {
                            const typeMap: Record<string, { bg: string; border: string; text: string; badge: string }> = {
                              'graph': {
                                bg: 'bg-blue-50/50 dark:bg-blue-950/20',
                                border: 'border-blue-200 dark:border-blue-800',
                                text: 'text-blue-700 dark:text-blue-300',
                                badge: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300'
                              },
                              'ai_analysis': {
                                bg: 'bg-purple-50/50 dark:bg-purple-950/20',
                                border: 'border-purple-200 dark:border-purple-800',
                                text: 'text-purple-700 dark:text-purple-300',
                                badge: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300'
                              },
                              'k8s_events': {
                                bg: 'bg-green-50/50 dark:bg-green-950/20',
                                border: 'border-green-200 dark:border-green-800',
                                text: 'text-green-700 dark:text-green-300',
                                badge: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
                              },
                              'node_info': {
                                bg: 'bg-amber-50/50 dark:bg-amber-950/20',
                                border: 'border-amber-200 dark:border-amber-800',
                                text: 'text-amber-700 dark:text-amber-300',
                                badge: 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300'
                              },
                              'container_info': {
                                bg: 'bg-cyan-50/50 dark:bg-cyan-950/20',
                                border: 'border-cyan-200 dark:border-cyan-800',
                                text: 'text-cyan-700 dark:text-cyan-300',
                                badge: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900 dark:text-cyan-300'
                              },
                              'text_file': {
                                bg: 'bg-slate-50/50 dark:bg-slate-950/20',
                                border: 'border-slate-200 dark:border-slate-800',
                                text: 'text-slate-700 dark:text-slate-300',
                                badge: 'bg-slate-100 text-slate-700 dark:bg-slate-900 dark:text-slate-300'
                              },
                              'diff': {
                                bg: 'bg-orange-50/50 dark:bg-orange-950/20',
                                border: 'border-orange-200 dark:border-orange-800',
                                text: 'text-orange-700 dark:text-orange-300',
                                badge: 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300'
                              },
                            };

                            return typeMap[type || ''] || {
                              bg: 'bg-card',
                              border: 'border-border',
                              text: 'text-foreground',
                              badge: 'bg-secondary text-secondary-foreground'
                            };
                          };

                          const colors = getTypeColor(enrichment.enrichment_type);

                          return (
                            <div key={idx}>
                              {/* 分隔线 - 除了第一个 */}
                              {idx > 0 && (
                                <div className="relative my-6">
                                  <div className="absolute inset-0 flex items-center">
                                    <div className="w-full border-t border-border"></div>
                                  </div>
                                  <div className="relative flex justify-center">
                                    <span className="bg-muted px-3 py-1 text-xs text-muted-foreground rounded-full">
                                      线索分隔
                                    </span>
                                  </div>
                                </div>
                              )}

                              {/* Enrichment 卡片 */}
                              <div className={`${colors.bg} rounded-lg border-2 ${colors.border} shadow-sm hover:shadow-md transition-all duration-200`}>
                                {/* 可折叠的标题栏 */}
                                <button
                                  onClick={() => toggleEnrichment(idx)}
                                  className="w-full flex items-center justify-between p-4 hover:bg-black/5 dark:hover:bg-white/5 transition-colors rounded-t-lg group"
                                >
                                  <div className="flex items-center gap-3">
                                    <div className={`p-1.5 rounded-md ${colors.badge} transition-transform group-hover:scale-110`}>
                                      {isCollapsed ? (
                                        <ChevronRight className="h-4 w-4" />
                                      ) : (
                                        <ChevronDown className="h-4 w-4" />
                                      )}
                                    </div>
                                    <div className="flex items-center gap-2">
                                      <Badge variant="outline" className={`text-xs font-semibold ${colors.badge} border-0`}>
                                        #{idx + 1}
                                      </Badge>
                                      {enrichment.title && (
                                        <span className={`text-sm font-medium ${colors.text}`}>
                                          {enrichment.title}
                                        </span>
                                      )}
                                    </div>
                                  </div>
                                  <div className="flex items-center gap-2">
                                    {enrichment.enrichment_type && (
                                      <Badge className={`text-xs ${colors.badge} border-0`}>
                                        {enrichment.enrichment_type}
                                      </Badge>
                                    )}
                                  </div>
                                </button>

                                {/* 内容区域 */}
                                {!isCollapsed && (
                                  <div className="p-4 pt-2 border-t-2 border-border/30 bg-card/50">
                                    <EnrichmentRenderer enrichment={enrichment} index={idx} />
                                  </div>
                                )}
                              </div>
                            </div>
                          );
                        })}
                      </>
                    ) : (
                      // 如果没有 enrichments，显示格式化数据
                      formattedContent
                    )}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export const RawPayloadViewer = React.memo(RawPayloadViewerComponent);
RawPayloadViewer.displayName = 'RawPayloadViewer';
