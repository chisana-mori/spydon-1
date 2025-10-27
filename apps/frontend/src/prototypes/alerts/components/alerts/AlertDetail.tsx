'use client';

import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  Clock,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Info,
  ExternalLink,
  Tag,
  Server
} from 'lucide-react';
import { Alert } from '@prototypes/alerts/types/alerts';
import { RawPayloadViewer } from './RawPayloadViewer';

interface AlertDetailProps {
  alert: Alert;
  className?: string;
}

export function AlertDetail({ alert, className }: AlertDetailProps) {
  // 获取严重程度颜色
  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'destructive';
      case 'high': return 'destructive';
      case 'medium': return 'default';
      case 'low': return 'secondary';
      default: return 'outline';
    }
  };

  // 获取状态图标
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
      case 'firing': return <AlertTriangle className="h-4 w-4" />;
      case 'resolved': return <CheckCircle className="h-4 w-4" />;
      case 'acknowledged': return <Info className="h-4 w-4" />;
      default: return <XCircle className="h-4 w-4" />;
    }
  };

  // 获取状态颜色
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
      case 'firing': return 'destructive';
      case 'resolved': return 'default';
      case 'acknowledged': return 'secondary';
      default: return 'outline';
    }
  };

  // 获取显示用的状态文本
  const getStatusText = (status: string) => {
    switch (status) {
      case 'firing': return '触发中';
      case 'active': return '活跃';
      case 'resolved': return '已解决';
      case 'acknowledged': return '已确认';
      default: return status.toUpperCase();
    }
  };

  return (
    <div className={`space-y-6 ${className}`}>
      {/* 基本信息 */}
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <div className="space-y-2">
              <CardTitle className="text-xl">{alert.title}</CardTitle>
              {alert.description && (
                <p className="text-muted-foreground">{alert.description}</p>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Badge variant={getSeverityColor(alert.severity)}>
                {alert.severity.toUpperCase()}
              </Badge>
              <Badge variant={getStatusColor(alert.status)} className="flex items-center gap-1">
                {getStatusIcon(alert.status)}
                {getStatusText(alert.status)}
              </Badge>
            </div>
          </div>
        </CardHeader>
        
        <CardContent className="space-y-4">
          {/* 时间信息 */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm">
                创建时间: {new Date(alert.created_at).toLocaleString()}
              </span>
            </div>
            {alert.cluster && (
              <div className="flex items-center gap-2">
                <Server className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm">
                  集群: {alert.cluster.name}
                </span>
              </div>
            )}
          </div>

          {/* 来源信息 */}
          <div className="flex items-center gap-2">
            <ExternalLink className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm">来源: {alert.source}</span>
            {alert.generatorURL && (
              <a 
                href={alert.generatorURL} 
                target="_blank" 
                rel="noopener noreferrer"
                className="text-blue-600 hover:text-blue-800 text-sm"
              >
                查看详情
              </a>
            )}
          </div>

          {alert.fingerprint && (
            <div className="text-sm text-muted-foreground">
              指纹: <code className="bg-muted px-1 rounded">{alert.fingerprint}</code>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 标签和注释 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 标签 */}
        {alert.labels && Object.keys(alert.labels).length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <Tag className="h-5 w-5" />
                标签
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {Object.entries(alert.labels).map(([key, value]) => (
                  <div key={key} className="flex items-center justify-between py-1">
                    <span className="text-sm font-medium">{key}</span>
                    <Badge variant="outline" className="text-xs">
                      {value}
                    </Badge>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* 注释 */}
        {alert.annotations && Object.keys(alert.annotations).length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <Info className="h-5 w-5" />
                注释
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {Object.entries(alert.annotations)
                  .filter(([key]) => key !== 'enrichment_keys' && key !== 'investigate_uri')
                  .map(([key, value]) => (
                    <div key={key} className="space-y-1">
                      <div className="text-sm font-medium">{key}</div>
                      <div className="text-sm text-muted-foreground bg-muted p-2 rounded">
                        {value}
                      </div>
                    </div>
                  ))}
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* 原始数据展示 */}
      {alert.raw_payload_key && (
        <>
          <Separator />
          <RawPayloadViewer
            rawPayloadKey={alert.raw_payload_key}
            alertId={alert.id}
            alertTitle={alert.title}
          />
        </>
      )}
    </div>
  );
}
