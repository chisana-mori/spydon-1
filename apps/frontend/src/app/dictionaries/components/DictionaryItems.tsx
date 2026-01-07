'use client';

import React, { useState } from 'react';
import { Plus, Trash, ChevronRight, ChevronDown, GripVertical, FileText } from 'lucide-react';
import { DictionaryItem } from '@/types/dictionary';
import { Button } from '@/components/ui/button';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/table';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { cn } from '@/lib/utils';
import { Badge } from '@/components/ui/badge';

interface DictionaryItemsProps {
    items: DictionaryItem[];
    onChange: (items: DictionaryItem[]) => void;
    keySameAsValue: boolean;
}

interface DictionaryItemRowProps {
    item: DictionaryItem;
    index: number;
    onChange: (index: number, field: keyof DictionaryItem, value: any) => void;
    onDelete: (index: number) => void;
    keySameAsValue: boolean;
}

/**
 * Individual Row Component handling its own expansion state
 */
function DictionaryItemRow({ item, index, onChange, onDelete, keySameAsValue }: DictionaryItemRowProps) {
    const [isExpanded, setIsExpanded] = useState(false);

    // Auto-sync key if needed
    const handleValueChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const val = e.target.value;
        onChange(index, 'value', val);
        if (keySameAsValue) {
            onChange(index, 'key', val);
        }
    };

    return (
        <>
            <TableRow className={cn("group transition-colors hover:bg-muted/30", isExpanded && "bg-muted/30")}>
                {/* Expand Toggle */}
                <TableCell className="w-[40px] p-2 pl-4 align-top">
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0 mt-1 hover:bg-muted"
                        onClick={() => setIsExpanded(!isExpanded)}
                    >
                        {isExpanded ? (
                            <ChevronDown className="h-4 w-4 text-muted-foreground" />
                        ) : (
                            <ChevronRight className="h-4 w-4 text-muted-foreground" />
                        )}
                    </Button>
                </TableCell>

                {/* Value Input (Main) */}
                <TableCell className="p-2 align-top">
                    <Input
                        value={item.value}
                        onChange={handleValueChange}
                        placeholder="显示值"
                        className="h-10 border-transparent bg-transparent hover:bg-background hover:border-input focus:bg-background focus:border-input transition-all font-medium px-3"
                    />
                </TableCell>

                {/* Key Input */}
                <TableCell className="p-2 align-top">
                    <Input
                        value={item.key}
                        onChange={(e) => onChange(index, 'key', e.target.value)}
                        disabled={keySameAsValue}
                        placeholder="存储键"
                        className={cn(
                            "h-10 border-transparent bg-transparent hover:bg-background hover:border-input focus:bg-background focus:border-input transition-all font-mono text-xs px-3",
                            keySameAsValue && "text-muted-foreground opacity-80 cursor-not-allowed"
                        )}
                    />
                </TableCell>

                {/* Sort Order */}
                <TableCell className="w-[80px] p-2 align-top">
                    <Input
                        type="number"
                        value={item.sort_order}
                        onChange={(e) => onChange(index, 'sort_order', Number(e.target.value))}
                        className="h-10 w-16 text-center border-transparent bg-transparent hover:bg-background hover:border-input focus:bg-background focus:border-input transition-all px-1"
                    />
                </TableCell>

                {/* Status */}
                <TableCell className="w-[80px] p-2 text-center align-top">
                    <div className="h-10 flex items-center justify-center">
                        <Switch
                            checked={item.is_enabled}
                            onCheckedChange={(checked) => onChange(index, 'is_enabled', checked)}
                            className="scale-75 data-[state=checked]:bg-green-500"
                        />
                    </div>
                </TableCell>

                {/* Actions */}
                <TableCell className="w-[50px] p-2 pr-4 text-right align-top">
                    <div className="h-10 flex items-center justify-end">
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-muted-foreground hover:text-destructive hover:bg-destructive/10 opacity-0 group-hover:opacity-100 transition-all"
                            onClick={() => onDelete(index)}
                        >
                            <Trash className="h-4 w-4" />
                        </Button>
                    </div>
                </TableCell>
            </TableRow>

            {/* Expanded Details Row */}
            {isExpanded && (
                <TableRow className="bg-muted/30 hover:bg-muted/30 border-t-0">
                    <TableCell colSpan={6} className="p-0 pb-4">
                        <div className="mx-4 ml-12 p-4 grid grid-cols-2 gap-6 bg-background/50 rounded-lg border border-border/50">
                            {/* Description */}
                            <div className="space-y-2">
                                <span className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold pl-1">描述 (Description)</span>
                                <Textarea
                                    value={item.description || ''}
                                    onChange={(e) => onChange(index, 'description', e.target.value)}
                                    placeholder="输入备注信息..."
                                    className="min-h-[80px] resize-none bg-background text-sm"
                                />
                            </div>

                            {/* Right Column: Default & Extra */}
                            <div className="space-y-4">
                                <div className="space-y-2">
                                    <span className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold pl-1">扩展数据 (JSON)</span>
                                    <Input
                                        value={item.extra || ''}
                                        onChange={(e) => onChange(index, 'extra', e.target.value)}
                                        placeholder='{"color": "red"}'
                                        className="h-9 bg-background font-mono text-xs"
                                    />
                                </div>
                                <div className="flex items-center gap-3 pt-2 pl-1">
                                    <Switch
                                        id={`default-${index}`}
                                        checked={item.is_default}
                                        onCheckedChange={(checked) => onChange(index, 'is_default', checked)}
                                        className="scale-90"
                                    />
                                    <label htmlFor={`default-${index}`} className="text-sm font-medium text-foreground/80 cursor-pointer select-none">
                                        设为默认选项
                                    </label>
                                </div>
                            </div>
                        </div>
                    </TableCell>
                </TableRow>
            )}
        </>
    );
}

export function DictionaryItems({ items, onChange, keySameAsValue }: DictionaryItemsProps) {
    const handleItemChange = (index: number, field: keyof DictionaryItem, value: any) => {
        const newItems = [...items];
        newItems[index] = { ...newItems[index], [field]: value };
        onChange(newItems);
    };

    const handleDelete = (index: number) => {
        const newItems = [...items];
        newItems.splice(index, 1);
        onChange(newItems);
    };

    const handleAddItem = () => {
        const newItem: DictionaryItem = {
            id: 0, // temp id
            key: '',
            value: '',
            is_enabled: true,
            is_default: false,
            sort_order: items.length > 0 ? (items[items.length - 1].sort_order || 0) + 10 : 0,
            description: '',
            extra: '',
        };
        onChange([...items, newItem]);
    };

    return (
        <div className="flex flex-col h-full bg-background space-y-4">
            <div className="flex-1 overflow-auto">
                <Table>
                    <TableHeader className="bg-muted/50 sticky top-0 z-10 shadow-sm">
                        <TableRow>
                            <TableHead className="w-[40px]"></TableHead>
                            <TableHead className="">显示值 (Value)</TableHead>
                            <TableHead className="">存储值 (Key)</TableHead>
                            <TableHead className="w-[80px] text-center">排序</TableHead>
                            <TableHead className="w-[80px] text-center">状态</TableHead>
                            <TableHead className="w-[50px]"></TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {items.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={6} className="h-48 text-center text-muted-foreground border-b-0">
                                    <div className="flex flex-col items-center justify-center gap-3 opacity-60">
                                        <div className="p-3 bg-muted/50 rounded-full">
                                            <FileText className="h-6 w-6" />
                                        </div>
                                        <div className="space-y-1">
                                            <p className="font-medium">暂无选项</p>
                                            <p className="text-xs">点击下方按钮添加第一个选项</p>
                                        </div>
                                    </div>
                                </TableCell>
                            </TableRow>
                        ) : (
                            items.map((item, index) => (
                                <DictionaryItemRow
                                    key={index}
                                    index={index}
                                    item={item}
                                    onChange={handleItemChange}
                                    onDelete={handleDelete}
                                    keySameAsValue={keySameAsValue}
                                />
                            ))
                        )}
                        {/* Quick Add Row at bottom of list */}
                        <TableRow className="hover:bg-transparent border-t-0">
                            <TableCell colSpan={6} className="p-2">
                                <Button
                                    variant="ghost"
                                    className="w-full border-2 border-dashed border-muted hover:border-primary/50 hover:bg-primary/5 text-muted-foreground hover:text-primary h-12 flex items-center justify-center gap-2 transition-all mt-2"
                                    onClick={handleAddItem}
                                >
                                    <Plus className="h-4 w-4" />
                                    添加新行
                                </Button>
                            </TableCell>
                        </TableRow>
                    </TableBody>
                </Table>
            </div>

            <div className="px-6 pb-2 text-xs text-muted-foreground text-center">
                点击行首箭头展开详细配置 • 所有更改将在保存字典时一并提交
            </div>
        </div>
    );
}
