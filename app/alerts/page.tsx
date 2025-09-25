'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { AlertList } from '@/components/alerts/AlertList';
import { alertsAPI } from '@/lib/alerts-api';
import { Alert, AlertsQueryParams } from '@/types/alerts';
import { 
  Search, 
  Filter, 
  RefreshCw,
  AlertTriangle,
  CheckCircle,
  Clock
} from 'lucide-react';

export default function AlertsPage() {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stats, setStats] = useState<any>(null);
  
  // 查询参数
  const [queryParams, setQueryParams] = useState<AlertsQueryParams>({
    page: 1,
    pageSize: 20,
  });
  
  // 搜索和过滤状态
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedSeverity, setSelectedSeverity] = useState<string>('');
  const [selectedStatus, setSelectedStatus] = useState<string>('');

  // 加载告警数据
  const loadAlerts = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const params: AlertsQueryParams = {
        ...queryParams,
        search: searchTerm || undefined,
        severity: selectedSeverity ? [selectedSeverity] : undefined,
        status: selectedStatus ? [selectedStatus] : undefined,
      };
      
      const response = await alertsAPI.getAlerts(params);
      setAlerts(response.alerts);
      
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载告警数据失败');
    } finally {
      setLoading(false);
    }
  };

  // 加载统计数据
  const loadStats = async () => {
    try {
      const statsData = await alertsAPI.getAlertStats();
      setStats(statsData);
    } catch (err) {
      console.error('Failed to load stats:', err);
    }
  };

  // 初始加载
  useEffect(() => {
    loadAlerts();
    loadStats();
  }, []);

  // 查询参数变化时重新加载
  useEffect(() => {
    loadAlerts();
  }, [queryParams, searchTerm, selectedSeverity, selectedStatus]);

  // 处理搜索
  const handleSearch = () => {
    setQueryParams(prev => ({ ...prev, page: 1 }));
  };

  // 处理过滤器变化
  const handleFilterChange = () => {
    setQueryParams(prev => ({ ...prev, page: 1 }));
  };

  // 重置过滤器
  const resetFilters = () => {
    setSearchTerm('');
    setSelectedSeverity('');
    setSelectedStatus('');
    setQueryParams({ page: 1, pageSize: 20 });
  };

  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">告警管理</h1>
          <p className="text-muted-foreground mt-1">
            查看和管理系统告警，包括原始数据分析
          </p>
        </div>
        <Button onClick={loadAlerts} disabled={loading} className="flex items-center gap-2">
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </Button>
      </div>

      {/* 统计卡片 */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-orange-500" />
                <div>
                  <div className="text-2xl font-bold">{stats.total}</div>
                  <div className="text-sm text-muted-foreground">总告警</div>
                </div>
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-red-500" />
                <div>
                  <div className="text-2xl font-bold">{stats.active}</div>
                  <div className="text-sm text-muted-foreground">活跃告警</div>
                </div>
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-2">
                <CheckCircle className="h-5 w-5 text-green-500" />
                <div>
                  <div className="text-2xl font-bold">{stats.resolved}</div>
                  <div className="text-sm text-muted-foreground">已解决</div>
                </div>
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center gap-2">
                <Clock className="h-5 w-5 text-blue-500" />
                <div>
                  <div className="text-2xl font-bold">{stats.acknowledged}</div>
                  <div className="text-sm text-muted-foreground">已确认</div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 搜索和过滤 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Filter className="h-5 w-5" />
            搜索和过滤
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col md:flex-row gap-4">
            {/* 搜索框 */}
            <div className="flex-1 flex gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="搜索告警标题、描述..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10"
                  onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                />
              </div>
              <Button onClick={handleSearch}>搜索</Button>
            </div>
            
            {/* 过滤器 */}
            <div className="flex gap-2">
              <Select value={selectedSeverity} onValueChange={setSelectedSeverity}>
                <SelectTrigger className="w-32">
                  <SelectValue placeholder="严重程度" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">全部</SelectItem>
                  <SelectItem value="critical">严重</SelectItem>
                  <SelectItem value="high">高</SelectItem>
                  <SelectItem value="medium">中</SelectItem>
                  <SelectItem value="low">低</SelectItem>
                </SelectContent>
              </Select>
              
              <Select value={selectedStatus} onValueChange={setSelectedStatus}>
                <SelectTrigger className="w-32">
                  <SelectValue placeholder="状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">全部</SelectItem>
                  <SelectItem value="active">活跃</SelectItem>
                  <SelectItem value="resolved">已解决</SelectItem>
                  <SelectItem value="acknowledged">已确认</SelectItem>
                </SelectContent>
              </Select>
              
              <Button variant="outline" onClick={resetFilters}>
                重置
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 错误提示 */}
      {error && (
        <Card className="border-red-200 bg-red-50">
          <CardContent className="p-4">
            <div className="flex items-center gap-2 text-red-700">
              <AlertTriangle className="h-5 w-5" />
              <span>{error}</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 告警列表 */}
      <AlertList 
        alerts={alerts} 
        loading={loading}
        onAlertClick={(alert) => {
          console.log('Alert clicked:', alert);
          // 这里可以添加更多的点击处理逻辑
        }}
      />
    </div>
  );
}
