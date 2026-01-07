
'use client';

import React, { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Loader2 } from 'lucide-react';
import { Dictionary, DictionaryItem } from '@/types/dictionary';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
    Form,
    FormControl,
    FormDescription,
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
import { Switch } from '@/components/ui/switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { DictionaryItems } from './DictionaryItems';
import { toast } from 'sonner';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { BookMarked, LayoutDashboard, List } from 'lucide-react';
import { createDictionary, updateDictionary, batchUpdateDictionaryItems, getDictionary } from '@/lib/api/dictionaries';
import { useQueryClient } from '@tanstack/react-query';

interface DictionaryDrawerProps {
    open: boolean;
    onClose: () => void;
    dictionaryId?: number | null; // null for create
}

const dictionarySchema = z.object({
    code: z.string().min(1, 'Code is required').regex(/^[a-zA-Z0-9_]+$/, 'Only letters, numbers and underscores'),
    name: z.string().min(1, 'Name is required'),
    module: z.string().min(1, 'Module is required'),
    description: z.string().optional(),
    is_enabled: z.boolean(),
    key_same_as_value: z.boolean(),
    sort_order: z.coerce.number(),
});

type DictionaryFormValues = z.infer<typeof dictionarySchema>;

export function DictionaryDrawer({ open, onClose, dictionaryId }: DictionaryDrawerProps) {

    const queryClient = useQueryClient();
    const [isLoading, setIsLoading] = React.useState(false);
    const [activeTab, setActiveTab] = React.useState('basic');
    const [items, setItems] = React.useState<DictionaryItem[]>([]);

    const form = useForm({
        resolver: zodResolver(dictionarySchema),
        defaultValues: {
            code: '',
            name: '',
            module: 'system',
            description: '',
            is_enabled: true,
            key_same_as_value: false,
            sort_order: 0,
        },
    });

    // Load data when editing
    useEffect(() => {
        if (open && dictionaryId) {
            setIsLoading(true);
            getDictionary(dictionaryId)
                .then((data) => {
                    form.reset({
                        code: data.code,
                        name: data.name,
                        module: data.module,
                        description: data.description || '',
                        is_enabled: data.is_enabled,
                        key_same_as_value: data.key_same_as_value,
                        sort_order: data.sort_order,
                    });
                    setItems(data.items || []);
                })
                .catch((err) => {
                    toast.error('加载字典详情失败');
                    console.error(err);
                })
                .finally(() => setIsLoading(false));
            setActiveTab('basic');
        } else if (open) {
            // Create mode
            form.reset({
                code: '',
                name: '',
                module: 'system',
                description: '',
                is_enabled: true,
                key_same_as_value: false,
                sort_order: 0,
            });
            setItems([]);
            setActiveTab('basic');
        }
    }, [open, dictionaryId, form]);

    const onSubmit = async (data: DictionaryFormValues) => {
        setIsLoading(true);
        try {
            if (dictionaryId) {
                // Update
                await updateDictionary(dictionaryId, data);
                await batchUpdateDictionaryItems(dictionaryId, items);
                toast.success('字典更新成功');
            } else {
                // Create
                const newDict = await createDictionary(data);
                // If there are items, add them
                if (items.length > 0) {
                    await batchUpdateDictionaryItems(newDict.id, items);
                }
                toast.success('字典创建成功');
            }

            queryClient.invalidateQueries({ queryKey: ['dictionaries'] });
            onClose();
        } catch (error) {
            console.error(error);
            toast.error('保存失败');
        } finally {
            setIsLoading(false);
        }
    };

    const isKeySameAsValue = form.watch('key_same_as_value');

    return (
        <Sheet open={open} onOpenChange={onClose}>
            <SheetContent className="sm:max-w-[700px] w-full flex flex-col h-full p-0 gap-0 bg-background/95 backdrop-blur-sm">
                <SheetHeader className="px-6 py-4 border-b bg-muted/20">
                    <div className="flex items-center gap-4">
                        <div className="p-3 bg-blue-500/10 rounded-xl ring-1 ring-blue-500/20">
                            <BookMarked className="w-5 h-5 text-blue-500" />
                        </div>
                        <div className="space-y-1">
                            <SheetTitle className="flex items-center gap-3 text-xl">
                                {dictionaryId ? '编辑字典' : '新建字典'}
                                {dictionaryId ? (
                                    <Badge variant={form.watch('is_enabled') ? 'default' : 'secondary'} className={form.watch('is_enabled') ? "h-5 px-2 text-[10px] font-medium bg-green-500/10 text-green-600 border-green-500/20 shadow-none hover:bg-green-500/20" : "h-5 px-2 text-[10px] font-medium bg-red-500/10 text-red-500 border-red-500/20 shadow-none hover:bg-red-500/20"}>
                                        {form.watch('is_enabled') ? '已启用' : '已禁用'}
                                    </Badge>
                                ) : (
                                    <Badge className="h-5 px-2 text-[10px] font-medium bg-blue-500/10 text-blue-600 hover:bg-blue-500/20 border-blue-500/20 shadow-none">
                                        NEW
                                    </Badge>
                                )}
                            </SheetTitle>
                            <SheetDescription className="text-xs">
                                配置系统字典的基本属性与选项值。
                            </SheetDescription>
                        </div>
                    </div>
                </SheetHeader>

                <div className="flex-1 overflow-hidden flex flex-col">
                    <Form {...form}>
                        <form id="dictionary-form" onSubmit={form.handleSubmit(onSubmit)} className="h-full flex flex-col">
                            <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col overflow-hidden">
                                <div className="px-6 py-2 border-b bg-background">
                                    <TabsList className="grid w-full grid-cols-2 bg-muted/50">
                                        <TabsTrigger value="basic" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">
                                            <LayoutDashboard className="w-4 h-4 mr-2" />
                                            基本信息
                                        </TabsTrigger>
                                        <TabsTrigger value="items" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">
                                            <List className="w-4 h-4 mr-2" />
                                            字典项
                                            <Badge variant="secondary" className="ml-2 h-5 px-1.5 text-[10px] min-w-[1.25rem] bg-primary/10 text-primary hover:bg-primary/20 border-0">
                                                {items.length}
                                            </Badge>
                                        </TabsTrigger>
                                    </TabsList>
                                </div>

                                <TabsContent value="basic" className="flex-1 overflow-y-auto p-6 space-y-8 mt-0">
                                    {/* Primary Information */}
                                    <div className="space-y-4">
                                        <FormField
                                            control={form.control}
                                            name="name"
                                            render={({ field }) => (
                                                <FormItem>
                                                    <FormLabel className="text-base font-semibold">字典名称</FormLabel>
                                                    <FormControl>
                                                        <Input {...field} placeholder="例如：用户状态" className="h-10 text-base" />
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
                                                    <FormLabel>描述信息</FormLabel>
                                                    <FormControl>
                                                        <Textarea
                                                            {...field}
                                                            placeholder="请输入关于此字典的详细描述..."
                                                            className="min-h-[80px] resize-none text-sm"
                                                        />
                                                    </FormControl>
                                                    <FormMessage />
                                                </FormItem>
                                            )}
                                        />
                                    </div>

                                    <div className="h-px bg-border/50" />

                                    {/* Technical Details */}
                                    <div className="grid grid-cols-2 gap-6">
                                        <FormField
                                            control={form.control}
                                            name="code"
                                            render={({ field }) => (
                                                <FormItem>
                                                    <FormLabel className="text-xs text-muted-foreground uppercase tracking-wider font-semibold">编码 (Code)</FormLabel>
                                                    <FormControl>
                                                        <Input {...field} placeholder="unique_code_name" disabled={!!dictionaryId} className="font-mono text-sm bg-muted/30" />
                                                    </FormControl>
                                                    <FormMessage />
                                                </FormItem>
                                            )}
                                        />
                                        <FormField
                                            control={form.control}
                                            name="module"
                                            render={({ field }) => (
                                                <FormItem>
                                                    <FormLabel className="text-xs text-muted-foreground uppercase tracking-wider font-semibold">所属模块</FormLabel>
                                                    <FormControl>
                                                        <Input {...field} placeholder="system" className="font-mono text-sm bg-muted/30" />
                                                    </FormControl>
                                                    <FormMessage />
                                                </FormItem>
                                            )}
                                        />
                                    </div>

                                    <div className="grid grid-cols-2 gap-6">
                                        <FormField
                                            control={form.control}
                                            name="sort_order"
                                            render={({ field }) => (
                                                <FormItem>
                                                    <FormLabel className="text-xs text-muted-foreground uppercase tracking-wider font-semibold">显示排序</FormLabel>
                                                    <FormControl>
                                                        <Input type="number" {...field} value={field.value as number} className="font-mono text-sm" />
                                                    </FormControl>
                                                    <FormMessage />
                                                </FormItem>
                                            )}
                                        />
                                    </div>

                                    <div className="h-px bg-border/50" />

                                    {/* Settings Toggles */}
                                    <div className="space-y-5">
                                        <FormField
                                            control={form.control}
                                            name="is_enabled"
                                            render={({ field }) => (
                                                <FormItem className="flex flex-row items-center justify-between space-y-0">
                                                    <div className="space-y-1">
                                                        <FormLabel className="text-sm font-medium flex items-center gap-2">
                                                            启用字典
                                                            {field.value && <span className="flex h-1.5 w-1.5 rounded-full bg-green-500 shadow-[0_0_4px_1px_rgba(34,197,94,0.3)] transition-all" />}
                                                        </FormLabel>
                                                        <FormDescription className="text-xs">
                                                            控制此字典在系统中的可见性
                                                        </FormDescription>
                                                    </div>
                                                    <FormControl>
                                                        <Switch
                                                            checked={field.value}
                                                            onCheckedChange={field.onChange}
                                                        />
                                                    </FormControl>
                                                </FormItem>
                                            )}
                                        />

                                        <FormField
                                            control={form.control}
                                            name="key_same_as_value"
                                            render={({ field }) => (
                                                <FormItem className="flex flex-row items-center justify-between space-y-0">
                                                    <div className="space-y-1">
                                                        <FormLabel className="text-sm font-medium">键值同步</FormLabel>
                                                        <FormDescription className="text-xs">
                                                            自动保持 Key 与 Value 一致
                                                        </FormDescription>
                                                    </div>
                                                    <FormControl>
                                                        <Switch
                                                            checked={field.value}
                                                            onCheckedChange={field.onChange}
                                                        />
                                                    </FormControl>
                                                </FormItem>
                                            )}
                                        />
                                    </div>
                                </TabsContent>

                                <TabsContent value="items" className="flex-1 overflow-y-auto mt-0">
                                    <DictionaryItems
                                        items={items}
                                        onChange={setItems}
                                        keySameAsValue={isKeySameAsValue}
                                    />
                                </TabsContent>
                            </Tabs>
                        </form>
                    </Form>
                </div>

                <SheetFooter className="px-6 py-4 border-t bg-muted/20 sm:space-x-4">
                    <Button variant="outline" onClick={onClose} disabled={isLoading} className="w-24">取消</Button>
                    <Button type="submit" form="dictionary-form" disabled={isLoading} className="w-24">
                        {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                        保存
                    </Button>
                </SheetFooter>
            </SheetContent>
        </Sheet>
    );
}
