'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { resolveAppPath } from '@/config';

/**
 * 系统设置入口按钮
 * 现在跳转到独立页面，而不是弹出对话框
 */
export function SystemSettingsDialog() {
    const router = useRouter();

    const handleNavigate = () => {
        router.push(resolveAppPath('/system-settings'));
    };

    return (
        <Button variant="ghost" size="icon" className="relative" onClick={handleNavigate}>
            <Settings className="h-5 w-5" />
            <span className="sr-only">系统设置</span>
        </Button>
    );
}
