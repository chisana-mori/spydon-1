'use client';

import React, { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Loader2, AlertTriangle } from 'lucide-react';
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
import { createTaint, updateTaint, TaintManagement } from '@/lib/api/configuration';
import { useDictionary } from '@/hooks/useDictionary';

const formSchema = z.object({
    key: z.string().min(1, 'Key不能为空'),
    value: z.string().optional(),
    effect: z.string().min(1, 'Effect不能为空'),
    description: z.string().optional(),
    type: z.string().optional(),
    status: z.coerce.number(),
});

type FormValues = z.infer<typeof formSchema>;

interface TaintSheetProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    taint?: TaintManagement | null;
}

const EFFECT_OPTIONS = [
    { value: 'NoSchedule', label: 'NoSchedule' },
    { value: 'PreferNoSchedule', label: 'PreferNoSchedule' },
    { value: 'NoExecute', label: 'NoExecute' },
];

export function TaintSheet({ open, onOpenChange, taint }: TaintSheetProps) {
    const queryClient = useQueryClient();
    const { items: statusItems } = useDictionary('taint_status');

    const form = useForm<FormValues>({
        resolver: zodResolver(formSchema) as any,
        defaultValues: {
            key: '',
            value: '',
            effect: 'NoSchedule',
            description: '',
            type: '',
            status: 0,
        },
    });

    useEffect(() => {
        if (open && taint) {
            form.reset({
                key: taint.key,
                value: taint.value,
                effect: taint.effect,
                description: taint.description,
                type: taint.type,
                status: taint.status,
            });
        } else if (open) {
            form.reset({
                key: '',
                value: '',
                effect: 'NoSchedule',
                description: '',
                type: '',
                status: 0,
            });
        }
    }, [open, taint, form]);

    const mutation = useMutation({
        mutationFn: async (values: FormValues) => {
            if (taint) {
                return updateTaint(taint.id, values);
            } else {
                return createTaint(values);
            }
        },
        onSuccess: () => {
            toast.success(taint ? '污点更新成功' : '污点创建成功');
            queryClient.invalidateQueries({ queryKey: ['taints'] });
            onOpenChange(false);
        },
        onError: (error) => {
            console.error(error);
            toast.error(taint ? '更新失败' : '创建失败');
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
                            <AlertTriangle className="w-5 h-5 text-blue-500" />
                        </div>
                        <div className="space-y-1">
                            <SheetTitle className="text-xl">
                                {taint ? '编辑污点' : '新建污点'}
                            </SheetTitle>
                            <SheetDescription className="text-sm">
                                {taint ? '修改现有污点的配置。' : '添加新的污点规则。'}
                            </SheetDescription>
                        </div>
                    </div>
                </SheetHeader>

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6 mt-6 pb-20">
                        <FormField
                            control={form.control}
                            name="key"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Key</FormLabel>
                                    <FormControl>
                                        <Input {...field} placeholder="例如：node-role.kubernetes.io/master" />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="value"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Value</FormLabel>
                                    <FormControl>
                                        <Input {...field} placeholder="可选值" />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="effect"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Effect</FormLabel>
                                    <Select
                                        onValueChange={field.onChange}
                                        value={field.value}
                                    >
                                        <FormControl>
                                            <SelectTrigger>
                                                <SelectValue placeholder="选择Effect" />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            {EFFECT_OPTIONS.map(opt => (
                                                <SelectItem key={opt.value} value={opt.value}>
                                                    {opt.label}
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
                            name="type"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>分类类型</FormLabel>
                                    <FormControl>
                                        <Input {...field} placeholder="例如：系统污点/硬件污点" />
                                    </FormControl>
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

                        <FormField
                            control={form.control}
                            name="description"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>描述</FormLabel>
                                    <FormControl>
                                        <Textarea {...field} placeholder="对此污点的详细说明..." className="resize-none" />
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
