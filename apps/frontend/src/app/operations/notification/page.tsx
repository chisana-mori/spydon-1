'use client';

import React from 'react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Mail, FileText, Users, Bell, Send, Sparkles } from "lucide-react";
import { SendEmailTab } from './components/SendEmailTab';
import { EmailTemplatesTab } from './components/EmailTemplatesTab';
import { EmailContactsTab } from './components/EmailContactsTab';

export default function NotificationManagementPage() {
    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <div className="p-3 bg-gradient-to-br from-orange-500/10 to-amber-500/10 rounded-2xl ring-1 ring-orange-500/20 shadow-lg shadow-orange-500/5">
                        <Bell className="h-7 w-7 text-orange-500" />
                    </div>
                    <div>
                        <h1 className="text-3xl font-bold tracking-tight bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
                            通知管理
                        </h1>
                        <p className="text-sm text-muted-foreground mt-1 flex items-center gap-2">
                            <Sparkles className="w-3.5 h-3.5" />
                            统一管理邮件发送、模板配置与联系人列表
                        </p>
                    </div>
                </div>
            </div>

            {/* Tabs */}
            <Tabs defaultValue="send" className="space-y-6">
                <TabsList className="w-full justify-start h-auto p-1 bg-muted/50 rounded-lg border">
                    <TabsTrigger
                        value="send"
                        className="flex-1 py-2.5 rounded-md data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm transition-all"
                    >
                        <Send className="w-4 h-4 mr-2" />
                        <span className="font-medium">邮件发送</span>
                    </TabsTrigger>
                    <TabsTrigger
                        value="templates"
                        className="flex-1 py-2.5 rounded-md data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm transition-all"
                    >
                        <FileText className="w-4 h-4 mr-2" />
                        <span className="font-medium">模板管理</span>
                    </TabsTrigger>
                    <TabsTrigger
                        value="contacts"
                        className="flex-1 py-2.5 rounded-md data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm transition-all"
                    >
                        <Users className="w-4 h-4 mr-2" />
                        <span className="font-medium">联系人</span>
                    </TabsTrigger>
                </TabsList>

                <TabsContent value="send" className="space-y-0">
                    <div className="rounded-2xl border bg-gradient-to-br from-card to-card/50 shadow-sm overflow-hidden">
                        <SendEmailTab />
                    </div>
                </TabsContent>

                <TabsContent value="templates" className="space-y-0">
                    <div className="rounded-2xl border bg-gradient-to-br from-card to-card/50 shadow-sm overflow-hidden">
                        <EmailTemplatesTab />
                    </div>
                </TabsContent>

                <TabsContent value="contacts" className="space-y-0">
                    <div className="rounded-2xl border bg-gradient-to-br from-card to-card/50 shadow-sm overflow-hidden">
                        <EmailContactsTab />
                    </div>
                </TabsContent>
            </Tabs>
        </div>
    );
}
