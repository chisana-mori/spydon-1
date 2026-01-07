'use client';

import React, { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Loader2, Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from '@/components/ui/form';
import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetHeader,
    SheetTitle,
    SheetFooter,
} from '@/components/ui/sheet';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import { Textarea } from '@/components/ui/textarea';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { createDeviceApp, updateDeviceApp, DeviceApp } from '@/lib/api/configuration';
import { useDictionary } from '@/hooks/useDictionary';

const formSchema = z.object({
    app_id: z.string().min(1, 'AppID不能为空'),
    name: z.string().min(1, '名称不能为空'),
    type: z.coerce.number(),
    owner: z.string().optional(),
    feature: z.string().optional(),
    description: z.string().optional(),
    status: z.coerce.number(),
});

type FormValues = z.infer<typeof formSchema>;

interface DeviceAppSheetProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    deviceApp?: DeviceApp | null;
}

export function DeviceAppSheet({ open, onOpenChange, deviceApp }: DeviceAppSheetProps) {
    const queryClient = useQueryClient();
    const { items: typeItems } = useDictionary('device_type');
    const { items: statusItems } = useDictionary('device_status');

    const form = useForm<FormValues>({
        resolver: zodResolver(formSchema) as any,
        defaultValues: {
            app_id: '',
            name: '',
            type: 0,
            owner: '',
            feature: '',
            description: '',
            status: 0,
        },
    });

    useEffect(() => {
        if (open && deviceApp) {
            form.reset({
                app_id: deviceApp.app_id,
                name: deviceApp.name,
                type: deviceApp.type,
                owner: deviceApp.owner || '',
                feature: deviceApp.feature || '',
                description: deviceApp.description || '',
                status: deviceApp.status,
            });
        } else if (open) {
            form.reset({
                app_id: '',
                name: '',
                type: 0,
                owner: '',
                feature: '',
                description: '',
                status: 0,
            });
        }
    }, [open, deviceApp, form]);

    const mutation = useMutation({
        mutationFn: async (values: FormValues) => {
            if (deviceApp) {
                return updateDeviceApp(deviceApp.id, values);
            } else {
                return createDeviceApp(values);
            }
        },
        onSuccess: () => {
            toast.success(deviceApp ? '应用更新成功' : '应用创建成功');
            queryClient.invalidateQueries({ queryKey: ['device-apps'] });
            onOpenChange(false);
        },
        onError: (error) => {
            console.error(error);
            toast.error(deviceApp ? '更新失败' : '创建失败');
        },
    });

    const onSubmit = (values: FormValues) => {
        mutation.mutate(values);
    };

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent className="sm:max-w-[500px] overflow-y-auto">
                <SheetHeader className="space-y-3 pb-6 border-b">
                    <div className="flex items-center gap-4">
                        <div className="p-3 bg-blue-500/10 rounded-xl ring-1 ring-blue-500/20">
                            <Settings className="w-5 h-5 text-blue-500" />
                        </div>
                        <div className="space-y-1">
                            <SheetTitle className="text-xl">
                                {deviceApp ? '编辑设备应用' : '新建设备应用'}
                            </SheetTitle>
                            <SheetDescription className="text-sm">
                                {deviceApp ? '修改设备应用属性。' : '注册新的设备应用类型。'}
                            </SheetDescription>
                        </div>
                    </div>
                </SheetHeader>

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6 mt-6 pb-20">
                        <FormField
                            control={form.control}
                            name="app_id"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>App ID</FormLabel>
                                    <FormControl>
                                        <Input {...field} placeholder="唯一标识，例如：compute-1001" disabled={!!deviceApp} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>应用名称</FormLabel>
                                    <FormControl>
                                        <Input {...field} placeholder="例如：Web Service" />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <div className="grid grid-cols-2 gap-4">
                            <FormField
                                control={form.control}
                                name="type"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>类型</FormLabel>
                                        <Select
                                            onValueChange={(val) => field.onChange(Number(val))}
                                            value={field.value?.toString()}
                                        >
                                            <FormControl>
                                                <SelectTrigger>
                                                    <SelectValue placeholder="选择类型" />
                                                </SelectTrigger>
                                            </FormControl>
                                            <SelectContent>
                                                {typeItems.map(item => (
                                                    <SelectItem key={item.key} value={item.key.toString()}>
                                                        {item.value}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />

                            <FormField
                                control={form.control}
                                name="status"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>状态</FormLabel>
                                        <Select
                                            onValueChange={(val) => field.onChange(Number(val))}
                                            value={field.value?.toString()}
                                        >
                                            <FormControl>
                                                <SelectTrigger>
                                                    <SelectValue placeholder="选择状态" />
                                                </SelectTrigger>
                                            </FormControl>
                                            <SelectContent>
                                                {statusItems.map(item => (
                                                    <SelectItem key={item.key} value={item.key.toString()}>
                                                        {item.value}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                        </div>

                        <FormField
                            control={form.control}
                            name="owner"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>负责人</FormLabel>
                                    <FormControl>
                                        <Input {...field} placeholder="Team or User" />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="feature"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>功能特性</FormLabel>
                                    <FormControl>
                                        <Input {...field} placeholder="Key features..." />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="description"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>用途描述</FormLabel>
                                    <FormControl>
                                        <Textarea {...field} placeholder="Detailed description..." className="resize-none" />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <SheetFooter>
                            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
                            <Button type="submit" disabled={mutation.isPending}>
                                {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                                保存
                            </Button>
                        </SheetFooter>
                    </form>
                </Form>
            </SheetContent>
        </Sheet>
    );
}
