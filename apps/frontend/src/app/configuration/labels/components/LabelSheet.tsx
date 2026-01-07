'use client';

import React, { useEffect } from 'react';
import { useForm, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Loader2, Tags, Plus, Trash } from 'lucide-react';
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
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { createLabel, updateLabel, LabelManagement } from '@/lib/api/configuration';
import { useDictionary } from '@/hooks/useDictionary';

const formSchema = z.object({
    name: z.string().min(1, '名称不能为空'),
    key: z.string().min(1, 'Key不能为空'),
    source: z.coerce.number(),
    status: z.coerce.number(),
    values: z.array(z.object({ value: z.string() })).default([]),
});

type FormValues = z.infer<typeof formSchema>;

interface LabelSheetProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    label?: LabelManagement | null;
}

export function LabelSheet({ open, onOpenChange, label }: LabelSheetProps) {
    const queryClient = useQueryClient();
    const { items: sourceItems } = useDictionary('label_source');
    const { items: statusItems } = useDictionary('label_status');

    const form = useForm<FormValues>({
        resolver: zodResolver(formSchema) as any,
        defaultValues: {
            name: '',
            key: '',
            source: 0,
            status: 0,
            values: [],
        },
    });

    const { fields, append, remove } = useFieldArray({
        control: form.control,
        name: "values",
    });

    useEffect(() => {
        if (open && label) {
            form.reset({
                name: label.name,
                key: label.key,
                source: label.source,
                status: label.status,
                values: label.values ? label.values.map(v => ({ value: v })) : [],
            });
        } else if (open) {
            form.reset({
                name: '',
                key: '',
                source: 0,
                status: 0,
                values: [],
            });
        }
    }, [open, label, form]);

    const mutation = useMutation({
        mutationFn: async (data: any) => {
            if (label) {
                return updateLabel(label.id, data);
            } else {
                return createLabel(data);
            }
        },
        onSuccess: () => {
            toast.success(label ? '标签更新成功' : '标签创建成功');
            queryClient.invalidateQueries({ queryKey: ['labels'] });
            onOpenChange(false);
        },
        onError: (error) => {
            console.error(error);
            toast.error(label ? '更新失败' : '创建失败');
        },
    });

    const onSubmit = (data: FormValues) => {
        // Map object array back to string array and filter empty
        const cleanValues = {
            ...data,
            values: data.values.map(v => v.value).filter(v => v.trim() !== '')
        };
        mutation.mutate(cleanValues);
    };

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent className="sm:max-w-[500px] overflow-y-auto">
                <SheetHeader className="space-y-3 pb-6 border-b">
                    <div className="flex items-center gap-4">
                        <div className="p-3 bg-blue-500/10 rounded-xl ring-1 ring-blue-500/20">
                            <Tags className="w-5 h-5 text-blue-500" />
                        </div>
                        <div className="space-y-1">
                            <SheetTitle className="text-xl">
                                {label ? '编辑标签' : '新建标签'}
                            </SheetTitle>
                            <SheetDescription className="text-sm">
                                {label ? '修改现有标签的属性。' : '创建一个新的标签配置。'}
                            </SheetDescription>
                        </div>
                    </div>
                </SheetHeader>

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6 mt-6 pb-20">
                        <FormField
                            control={form.control}
                            name="name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>名称</FormLabel>
                                    <FormControl>
                                        <Input {...field} placeholder="请输入标签名称" />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="key"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Key</FormLabel>
                                    <FormControl>
                                        <Input {...field} placeholder="请输入标签Key" />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <div className="space-y-3">
                            <div className="flex justify-between items-center">
                                <FormLabel className="text-sm font-medium">可选值 (Values)</FormLabel>
                                <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    onClick={() => append({ value: "" })}
                                    className="h-8 text-blue-600 border-blue-200 hover:bg-blue-50 hover:text-blue-700 transition-colors"
                                >
                                    <Plus className="w-3 h-3 mr-1" /> 添加值
                                </Button>
                            </div>
                            <div className="space-y-2">
                                {fields.map((field, index) => (
                                    <div key={field.id} className="flex gap-2 group">
                                        <FormField
                                            control={form.control}
                                            name={`values.${index}.value`}
                                            render={({ field }) => (
                                                <FormItem className="flex-1">
                                                    <FormControl>
                                                        <Input {...field} placeholder={`值 ${index + 1}`} className="bg-muted/30 focus:bg-background transition-colors" />
                                                    </FormControl>
                                                    <FormMessage />
                                                </FormItem>
                                            )}
                                        />

                                        <Button
                                            type="button"
                                            variant="ghost"
                                            size="icon"
                                            onClick={() => remove(index)}
                                            className="text-muted-foreground hover:text-red-600 hover:bg-red-50 transition-colors opacity-70 group-hover:opacity-100"
                                        >
                                            <Trash className="w-4 h-4" />
                                        </Button>
                                    </div>
                                ))}
                                {fields.length === 0 && (
                                    <div className="flex flex-col items-center justify-center py-8 border-2 border-dashed border-blue-500/20 rounded-xl bg-blue-500/5 text-center">
                                        <Tags className="w-8 h-8 text-blue-500/30 mb-2" />
                                        <p className="text-sm text-muted-foreground">
                                            暂无配置值
                                        </p>
                                        <Button
                                            type="button"
                                            variant="link"
                                            onClick={() => append({ value: "" })}
                                            className="text-blue-500 h-auto p-0 text-xs mt-1"
                                        >
                                            点击添加第一个值
                                        </Button>
                                    </div>
                                )}
                            </div>
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <FormField
                                control={form.control}
                                name="source"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>来源</FormLabel>
                                        <Select
                                            onValueChange={(val) => field.onChange(Number(val))}
                                            value={field.value?.toString()}
                                        >
                                            <FormControl>
                                                <SelectTrigger>
                                                    <SelectValue placeholder="选择来源" />
                                                </SelectTrigger>
                                            </FormControl>
                                            <SelectContent>
                                                {sourceItems.map(item => (
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
