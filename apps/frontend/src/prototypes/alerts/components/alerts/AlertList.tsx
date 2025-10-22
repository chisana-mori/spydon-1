'use client';

import React, { useState } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { 
  Clock, 
  AlertTriangle, 
  CheckCircle, 
  Info,
  FileText,
  Eye,
  ChevronRight
} from 'lucide-react';
import { Alert } from '@prototypes/alerts/types/alerts';
import { AlertDetail } from './AlertDetail';

interface AlertListProps {
  alerts: Alert[];
  loading?: boolean;
  onAlertClick?: (alert: Alert) => void;
  className?: string;
}

interface AlertItemProps {
  alert: Alert;
  onClick?: (alert: Alert) => void;
}

function AlertItem({ alert, onClick }: AlertItemProps) {
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
      case 'active': return <AlertTriangle className="h-4 w-4" />;
      case 'resolved': return <CheckCircle className="h-4 w-4" />;
      case 'acknowledged': return <Info className="h-4 w-4" />;
      default: return <AlertTriangle className="h-4 w-4" />;
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
    <Card 
      className="cursor-pointer hover:shadow-md transition-shadow"
      onClick={() => onClick?.(alert)}
    >
      <CardContent className="p-4">
        <div className="flex items-start justify-between">
          <div className="flex-1 space-y-2">
            {/* 标题和状态 */}
            <div className="flex items-start gap-3">
              <div className="flex-1">
                <h3 className="font-medium text-sm leading-tight">
                  {alert.title}
                </h3>
                {alert.description && (
                  <p className="text-xs text-muted-foreground mt-1 line-clamp-2">
                    {alert.description}
                  </p>
                )}
              </div>
              <div className="flex items-center gap-2 flex-shrink-0">
                {/* 原始数据指示器 */}
                {alert.raw_payload_key && (
                  <Badge variant="outline" className="text-xs flex items-center gap-1">
                    <FileText className="h-3 w-3" />
                    原始数据
                  </Badge>
                )}
                <Badge variant={getSeverityColor(alert.severity)} className="text-xs">
                  {alert.severity.toUpperCase()}
                </Badge>
              </div>
            </div>

            {/* 元信息 */}
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-1">
                  {getStatusIcon(alert.status)}
                  <span>{getStatusText(alert.status)}</span>
                </div>
                <div className="flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  <span>{new Date(alert.created_at).toLocaleString()}</span>
                </div>
                <div>集群: {alert.cluster?.name || alert.cluster_id || '未知'}</div>
              </div>
              <ChevronRight className="h-4 w-4" />
            </div>

            {/* 重要标签 */}
            {alert.labels && Object.keys(alert.labels).length > 0 && (
              <div className="flex items-center gap-1 flex-wrap">
                {Object.entries(alert.labels)
                  .slice(0, 3)
                  .map(([key, value]) => (
                    <Badge key={key} variant="outline" className="text-xs">
                      {key}: {value}
                    </Badge>
                  ))}
                {Object.keys(alert.labels).length > 3 && (
                  <Badge variant="outline" className="text-xs">
                    +{Object.keys(alert.labels).length - 3} 更多
                  </Badge>
                )}
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export function AlertList({ alerts, loading, onAlertClick, className }: AlertListProps) {
  const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null);

  const handleAlertClick = (alert: Alert) => {
    setSelectedAlert(alert);
    onAlertClick?.(alert);
  };

  if (loading) {
    return (
      <div className={`space-y-4 ${className}`}>
        {[...Array(5)].map((_, i) => (
          <Card key={i} className="animate-pulse">
            <CardContent className="p-4">
              <div className="space-y-3">
                <div className="h-4 bg-muted rounded w-3/4"></div>
                <div className="h-3 bg-muted rounded w-1/2"></div>
                <div className="flex gap-2">
                  <div className="h-5 bg-muted rounded w-16"></div>
                  <div className="h-5 bg-muted rounded w-20"></div>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  if (alerts.length === 0) {
    return (
      <Card className={className}>
        <CardContent className="p-8 text-center">
          <AlertTriangle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="text-lg font-medium mb-2">暂无告警</h3>
          <p className="text-muted-foreground">当前没有符合条件的告警信息</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className={`space-y-4 ${className}`}>
      {/* 告警列表 */}
      <div className="space-y-3">
        {alerts.map((alert) => (
          <AlertItem
            key={alert.id}
            alert={alert}
            onClick={handleAlertClick}
          />
        ))}
      </div>

      {/* 告警详情模态框或侧边栏 */}
      {selectedAlert && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
          <div className="bg-background rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden">
            <div className="flex items-center justify-between p-4 border-b">
              <h2 className="text-lg font-semibold">告警详情</h2>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setSelectedAlert(null)}
              >
                ✕
              </Button>
            </div>
            <div className="p-4 overflow-auto max-h-[calc(90vh-80px)]">
              <AlertDetail alert={selectedAlert} />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
