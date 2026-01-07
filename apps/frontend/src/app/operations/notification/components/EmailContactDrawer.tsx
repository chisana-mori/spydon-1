'use client';

import React, { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Loader2 } from 'lucide-react';
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
} from '@/components/ui/sheet';
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
            <SheetContent side="right" className="w-full sm:max-w-md overflow-y-auto">
                <SheetHeader>
                    <SheetTitle>{isEdit ? '编辑联系人' : '新建联系人'}</SheetTitle>
                    <SheetDescription>
                        {isEdit ? '修改联系人的名称和邮箱地址。' : '创建新的邮件联系人或联系人组。'}
                    </SheetDescription>
                </SheetHeader>

                {isLoadingContact ? (
                    <div className="flex items-center justify-center h-40">
                        <Loader2 className="w-6 h-6 animate-spin" />
                    </div>
                ) : (
                    <form onSubmit={handleSubmit} className="space-y-6 mt-6">
                        <div className="space-y-4">
                            <div className="space-y-2">
                                <Label htmlFor="name">名称 *</Label>
                                <Input
                                    id="name"
                                    value={formData.name}
                                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                    placeholder="例如：运维团队、张三"
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="address">邮箱地址 *</Label>
                                <Textarea
                                    id="address"
                                    value={formData.address}
                                    onChange={(e) => setFormData({ ...formData, address: e.target.value })}
                                    placeholder="输入邮箱地址，多个地址用逗号分隔"
                                    rows={4}
                                />
                                <p className="text-xs text-muted-foreground">
                                    支持多个邮箱地址，用逗号 (,) 分隔
                                </p>
                            </div>
                        </div>

                        {/* Submit */}
                        <div className="flex justify-end gap-3 pt-4 border-t">
                            <Button type="button" variant="outline" onClick={onClose}>
                                取消
                            </Button>
                            <Button type="submit" disabled={isSubmitting}>
                                {isSubmitting && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                                {isEdit ? '保存更改' : '创建联系人'}
                            </Button>
                        </div>
                    </form>
                )}
            </SheetContent>
        </Sheet>
    );
}
