'use client';

import React, { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Loader2, Mail, Users, AtSign } from 'lucide-react';
import { RobustaAPI } from '@/lib/api';
import { CreateEmailContactRequest, UpdateEmailContactRequest } from '@/types/email';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetHeader,
    SheetTitle,
    SheetFooter,
} from '@/components/ui/sheet';
import { Badge } from '@/components/ui/badge';
import { toast } from 'sonner';

interface EmailContactDrawerProps {
    open: boolean;
    onClose: () => void;
    contactId: number | null;
}

export function EmailContactDrawer({ open, onClose, contactId }: EmailContactDrawerProps) {
    const queryClient = useQueryClient();
    const isEdit = contactId !== null;

    const [formData, setFormData] = useState<CreateEmailContactRequest>({
        name: '',
        address: '',
    });

    // Fetch existing contact for edit mode
    const { data: contact, isLoading: isLoadingContact } = useQuery({
        queryKey: ['email-contact', contactId],
        queryFn: () => RobustaAPI.getEmailContact(contactId!),
        enabled: isEdit && open,
    });

    // Initialize form data when contact loads
    useEffect(() => {
        if (contact && isEdit) {
            setFormData({
                name: contact.name,
                address: contact.address,
            });
        } else if (!isEdit && open) {
            setFormData({
                name: '',
                address: '',
            });
        }
    }, [contact, isEdit, open]);

    // Create mutation
    const createMutation = useMutation({
        mutationFn: RobustaAPI.createEmailContact,
        onSuccess: () => {
            toast.success('联系人创建成功');
            queryClient.invalidateQueries({ queryKey: ['email-contacts'] });
            onClose();
        },
        onError: (error: any) => {
            toast.error(error?.response?.data?.message || '创建失败');
        },
    });

    // Update mutation
    const updateMutation = useMutation({
        mutationFn: (data: UpdateEmailContactRequest) =>
            RobustaAPI.updateEmailContact(contactId!, data),
        onSuccess: () => {
            toast.success('联系人更新成功');
            queryClient.invalidateQueries({ queryKey: ['email-contacts'] });
            queryClient.invalidateQueries({ queryKey: ['email-contact', contactId] });
            onClose();
        },
        onError: (error: any) => {
            toast.error(error?.response?.data?.message || '更新失败');
        },
    });

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (!formData.name.trim() || !formData.address.trim()) {
            toast.error('请填写必填字段');
            return;
        }

        if (isEdit) {
            updateMutation.mutate(formData);
        } else {
            createMutation.mutate(formData);
        }
    };

    const isSubmitting = createMutation.isPending || updateMutation.isPending;

    return (
        <Sheet open={open} onOpenChange={(val) => !val && onClose()}>
            <SheetContent className="sm:max-w-[600px] w-full flex flex-col h-full p-0 gap-0 bg-background/95 backdrop-blur-sm">
                <SheetHeader className="px-6 py-4 border-b bg-muted/20">
                    <div className="flex items-center gap-4">
                        <div className="p-3 bg-blue-500/10 rounded-xl ring-1 ring-blue-500/20">
                            <Users className="w-5 h-5 text-blue-500" />
                        </div>
                        <div className="space-y-1">
                            <SheetTitle className="flex items-center gap-3 text-xl">
                                {isEdit ? '编辑联系人' : '新建联系人'}
                                {isEdit ? (
                                    <Badge className="h-5 px-2 text-[10px] font-medium bg-blue-500/10 text-blue-600 hover:bg-blue-500/20 border-blue-500/20 shadow-none">
                                        EDIT
                                    </Badge>
                                ) : (
                                    <Badge className="h-5 px-2 text-[10px] font-medium bg-blue-500/10 text-blue-600 hover:bg-blue-500/20 border-blue-500/20 shadow-none">
                                        NEW
                                    </Badge>
                                )}
                            </SheetTitle>
                            <SheetDescription className="text-xs">
                                {isEdit ? '修改联系人的名称和邮箱地址。' : '创建新的邮件联系人或联系人组。'}
                            </SheetDescription>
                        </div>
                    </div>
                </SheetHeader>

                {isLoadingContact ? (
                    <div className="flex items-center justify-center h-40">
                        <Loader2 className="w-6 h-6 animate-spin" />
                    </div>
                ) : (
                    <form onSubmit={handleSubmit} className="h-full flex flex-col">
                        <div className="flex-1 overflow-y-auto p-6 space-y-8">
                            {/* Name Field */}
                            <div className="space-y-3">
                                <div className="flex items-center gap-2">
                                    <div className="p-1.5 bg-blue-500/10 rounded-lg">
                                        <Users className="w-4 h-4 text-blue-500" />
                                    </div>
                                    <Label htmlFor="name" className="text-base font-semibold">联系人名称</Label>
                                </div>
                                <Input
                                    id="name"
                                    value={formData.name}
                                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                    placeholder="例如：运维团队、张三"
                                    className="h-10 text-base"
                                />
                                <p className="text-xs text-muted-foreground pl-7">
                                    为联系人或联系人组指定一个易于识别的名称
                                </p>
                            </div>

                            <div className="h-px bg-border/50" />

                            {/* Email Address Field */}
                            <div className="space-y-3">
                                <div className="flex items-center gap-2">
                                    <div className="p-1.5 bg-blue-500/10 rounded-lg">
                                        <AtSign className="w-4 h-4 text-blue-500" />
                                    </div>
                                    <Label htmlFor="address" className="text-base font-semibold">邮箱地址</Label>
                                </div>
                                <Textarea
                                    id="address"
                                    value={formData.address}
                                    onChange={(e) => setFormData({ ...formData, address: e.target.value })}
                                    placeholder="user@example.com, team@example.com"
                                    rows={5}
                                    className="font-mono text-sm resize-none min-h-[120px]"
                                />
                                <div className="pl-7 space-y-1">
                                    <p className="text-xs text-muted-foreground flex items-center gap-1.5">
                                        <Mail className="w-3 h-3" />
                                        支持多个邮箱地址，使用逗号 ( , ) 分隔
                                    </p>
                                    <p className="text-[10px] text-muted-foreground/70">
                                        示例：admin@example.com, ops-team@example.com
                                    </p>
                                </div>
                            </div>

                            {/* Email Preview Card */}
                            {formData.address && (
                                <div className="pl-7">
                                    <div className="p-4 rounded-xl border bg-gradient-to-br from-blue-500/5 to-background border-blue-500/10">
                                        <p className="text-[10px] text-muted-foreground uppercase tracking-wider font-semibold mb-2">邮箱预览</p>
                                        <div className="flex flex-wrap gap-2">
                                            {formData.address.split(',').map((email, idx) => (
                                                email.trim() && (
                                                    <Badge key={idx} variant="secondary" className="font-mono text-xs h-6 px-2 bg-blue-500/10 text-blue-600 border-blue-500/20 shadow-none">
                                                        {email.trim()}
                                                    </Badge>
                                                )
                                            ))}
                                        </div>
                                    </div>
                                </div>
                            )}
                        </div>

                        <SheetFooter className="px-6 py-4 border-t bg-muted/20 sm:space-x-4">
                            <Button type="button" variant="outline" onClick={onClose} disabled={isSubmitting} className="w-24">
                                取消
                            </Button>
                            <Button type="submit" disabled={isSubmitting} className="w-28">
                                {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                                {isEdit ? '保存更改' : '创建联系人'}
                            </Button>
                        </SheetFooter>
                    </form>
                )}
            </SheetContent>
        </Sheet>
    );
}
