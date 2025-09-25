'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import RobustaAPI from '@/lib/api';

export default function AlertsDebugPage() {
  const [alertId, setAlertId] = useState('a2b5f2f4-7e0e-4e44-a2b0-1266ae2c584e');
  const [alertData, setAlertData] = useState<any>(null);
  const [alertsData, setAlertsData] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 测试获取单个告警
  const testGetAlert = async () => {
    try {
      setLoading(true);
      setError(null);
      console.log('Testing getAlert with ID:', alertId);
      
      const response = await RobustaAPI.getAlert(alertId);
      console.log('getAlert response:', response);
      setAlertData(response);
    } catch (err: any) {
      console.error('getAlert error:', err);
      setError(`获取告警失败: ${err.response?.data?.message || err.message}`);
    } finally {
      setLoading(false);
    }
  };

  // 测试获取告警列表
  const testGetAlerts = async () => {
    try {
      setLoading(true);
      setError(null);
      console.log('Testing getAlerts...');
      
      const response = await RobustaAPI.getAlerts(1, 10);
      console.log('getAlerts response:', response);
      setAlertsData(response);
    } catch (err: any) {
      console.error('getAlerts error:', err);
      setError(`获取告警列表失败: ${err.response?.data?.message || err.message}`);
    } finally {
      setLoading(false);
    }
  };

  // 测试 MinIO 连接
  const testMinIO = async () => {
    try {
      setLoading(true);
      setError(null);
      console.log('Testing MinIO connection...');
      
      // 测试 MinIO 端点
      const response = await fetch('http://localhost:9000/minio/health/live');
      console.log('MinIO health response:', response);
      
      if (response.ok) {
        setError('MinIO 连接成功！');
      } else {
        setError(`MinIO 连接失败: ${response.status} ${response.statusText}`);
      }
    } catch (err: any) {
      console.error('MinIO connection error:', err);
      setError(`MinIO 连接错误: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="container mx-auto p-6 space-y-6">
      <h1 className="text-3xl font-bold">告警 API 调试页面</h1>
      
      {/* 测试控制 */}
      <Card>
        <CardHeader>
          <CardTitle>API 测试</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Input
              placeholder="告警 ID"
              value={alertId}
              onChange={(e) => setAlertId(e.target.value)}
              className="flex-1"
            />
            <Button onClick={testGetAlert} disabled={loading}>
              测试获取单个告警
            </Button>
          </div>
          
          <div className="flex gap-2">
            <Button onClick={testGetAlerts} disabled={loading}>
              测试获取告警列表
            </Button>
            <Button onClick={testMinIO} disabled={loading}>
              测试 MinIO 连接
            </Button>
          </div>
          
          {loading && <p className="text-blue-600">加载中...</p>}
          {error && <p className="text-red-600">{error}</p>}
        </CardContent>
      </Card>

      {/* API 基础信息 */}
      <Card>
        <CardHeader>
          <CardTitle>API 配置信息</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2 text-sm">
            <p><strong>API Base URL:</strong> {process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1'}</p>
            <p><strong>MinIO Endpoint:</strong> {process.env.NEXT_PUBLIC_MINIO_ENDPOINT || 'http://localhost:9000'}</p>
            <p><strong>当前测试 ID:</strong> {alertId}</p>
          </div>
        </CardContent>
      </Card>

      {/* 告警列表数据 */}
      {alertsData && (
        <Card>
          <CardHeader>
            <CardTitle>告警列表数据</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="bg-gray-100 p-4 rounded text-xs overflow-auto max-h-96">
              {JSON.stringify(alertsData, null, 2)}
            </pre>
          </CardContent>
        </Card>
      )}

      {/* 单个告警数据 */}
      {alertData && (
        <Card>
          <CardHeader>
            <CardTitle>单个告警数据</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="bg-gray-100 p-4 rounded text-xs overflow-auto max-h-96">
              {JSON.stringify(alertData, null, 2)}
            </pre>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
