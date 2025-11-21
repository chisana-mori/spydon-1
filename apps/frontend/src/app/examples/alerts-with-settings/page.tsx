'use client';

import React from 'react';
import { SystemSettingsDialog } from '@/components/settings/SystemSettingsDialog';
import { AlertList } from '@prototypes/alerts/components/alerts/AlertList';

/**
 * Alert 页面示例
 * 展示如何在页面右上角添加系统设置按钮
 */
export default function AlertsPageExample() {
    return (
        <div className="container mx-auto py-6">
            {/* 页面头部 */}
            <div className="flex items-center justify-between mb-6">
                <div>
                    <h1 className="text-3xl font-bold">告警管理</h1>
                    <p className="text-muted-foreground mt-1">
                        查看和管理集群告警信息
                    </p>
                </div>

                {/* 右上角工具栏 */}
                <div className="flex items-center gap-2">
                    {/* 系统设置按钮 */}
                    <SystemSettingsDialog />
                </div>
            </div>

            {/* 告警列表 */}
            <AlertList alerts={[]} loading={false} />
        </div>
    );
}
