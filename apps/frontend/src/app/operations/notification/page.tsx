'use client';

import React from 'react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Mail, Settings, Users, Bell } from "lucide-react";
import { SendEmailTab } from './components/SendEmailTab';
import { EmailTemplatesTab } from './components/EmailTemplatesTab';
import { EmailContactsTab } from './components/EmailContactsTab';

export default function NotificationManagementPage() {
    return (
        <div className="space-y-6">
            <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-xl bg-orange-500/10 text-orange-500 ring-1 ring-orange-500/20">
                    <Bell className="h-6 w-6" />
                </div>
                <div>
                    <h1 className="text-2xl font-bold tracking-tight">通知管理</h1>
                    <p className="text-sm text-muted-foreground mt-0.5">
                        统一管理邮件发送、模板配置与联系人列表。
                    </p>
                </div>
            </div>

            <Tabs defaultValue="send" className="space-y-4">
                <TabsList>
                    <TabsTrigger value="send" className="flex items-center gap-2">
                        <Mail className="w-4 h-4" />
                        邮件发送
                    </TabsTrigger>
                    <TabsTrigger value="templates" className="flex items-center gap-2">
                        <Settings className="w-4 h-4" />
                        模版管理
                    </TabsTrigger>
                    <TabsTrigger value="contacts" className="flex items-center gap-2">
                        <Users className="w-4 h-4" />
                        联系人
                    </TabsTrigger>
                </TabsList>

                <TabsContent value="send" className="space-y-4">
                    <div className="p-4 border rounded-xl bg-card/50">
                        <SendEmailTab />
                    </div>
                </TabsContent>

                <TabsContent value="templates" className="space-y-4">
                    <div className="p-4 border rounded-xl bg-card/50">
                        <EmailTemplatesTab />
                    </div>
                </TabsContent>

                <TabsContent value="contacts" className="space-y-4">
                    <div className="p-4 border rounded-xl bg-card/50">
                        <EmailContactsTab />
                    </div>
                </TabsContent>
            </Tabs>
        </div>
    );
}
