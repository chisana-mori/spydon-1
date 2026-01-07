import * as React from "react"
import { Check, ChevronsUpDown, Loader2, Plus } from "lucide-react"
import { useQuery } from "@tanstack/react-query"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
    Popover,
    PopoverContent,
    PopoverTrigger,
} from "@/components/ui/popover"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { api } from "@/lib/api"

interface ConfigComboboxProps {
    value: string
    onChange: (value: string) => void
    type: 'label' | 'taint' | 'label-value' | 'taint-value'
    parentKey?: string // For fetching values dependent on a key
    placeholder?: string
    className?: string
    disabled?: boolean
}

export function ConfigCombobox({
    value,
    onChange,
    type,
    parentKey,
    placeholder = "Select or type...",
    className,
    disabled
}: ConfigComboboxProps) {
    const [open, setOpen] = React.useState(false)
    const [search, setSearch] = React.useState("")

    // Update search when value changes externally, but only if not open to avoid jumping
    React.useEffect(() => {
        if (!open) {
            setSearch(value)
        }
    }, [value, open])

    // Fetch data
    const { data, isLoading } = useQuery({
        queryKey: ['config-combobox', type, parentKey],
        queryFn: async () => {
            // Determine endpoint based on base type
            const isSimpleLabel = type === 'label';
            const isSimpleTaint = type === 'taint';
            const isValueLookup = type === 'label-value' || type === 'taint-value';

            const endpoint = (type.startsWith('label')) ? '/configuration/labels' : '/configuration/taints';

            const res = await api.get(`${endpoint}?page=1&size=100`);
            const list = res.data?.list || [];

            if (isSimpleLabel || isSimpleTaint) {
                return list.map((item: any) => item.key) as string[];
            }

            if (isValueLookup && parentKey) {
                const parentItem = list.find((item: any) => item.key === parentKey);
                if (!parentItem) return [];

                if (type === 'label-value') {
                    return (parentItem.values || []) as string[];
                }
                if (type === 'taint-value') {
                    return parentItem.value ? [parentItem.value] : [];
                }
            }

            return [] as string[];
        },
        staleTime: 60 * 1000, // 1 minute cache
        enabled: !((type === 'label-value' || type === 'taint-value') && !parentKey)
    })

    // Filter items based on search
    const filteredItems = React.useMemo(() => {
        if (!data) return []
        if (!search) return data
        return data.filter(item => item.toLowerCase().includes(search.toLowerCase()))
    }, [data, search])

    const handleSelect = (selectedValue: string) => {
        onChange(selectedValue)
        setSearch(selectedValue)
        setOpen(false)
    }

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const newVal = e.target.value
        setSearch(newVal)
        onChange(newVal) // Allow custom typing immediately
    }

    return (
        <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
                <div className={cn("relative", className)}>
                    <Input
                        value={value}
                        onChange={(e) => onChange(e.target.value)}
                        placeholder={placeholder}
                        className={cn("pr-8", className)}
                        disabled={disabled}
                        onClick={() => setOpen(true)}
                    />
                    <Button
                        variant="ghost"
                        size="icon"
                        className="absolute right-0 top-0 h-full w-8 text-muted-foreground hover:bg-transparent"
                        onClick={(e) => {
                            e.stopPropagation()
                            setOpen(!open)
                        }}
                        disabled={disabled}
                    >
                        <ChevronsUpDown className="h-4 w-4" />
                    </Button>
                </div>
            </PopoverTrigger>
            <PopoverContent className="w-[200px] p-0" align="start">
                <div className="p-2 border-b">
                    <Input
                        placeholder="Filter..."
                        value={search}
                        onChange={handleInputChange}
                        className="h-8 text-xs"
                        autoFocus
                    />
                </div>
                <ScrollArea className="h-[200px]">
                    <div className="p-1">
                        {isLoading ? (
                            <div className="flex items-center justify-center py-4 text-xs text-muted-foreground">
                                <Loader2 className="h-3 w-3 animate-spin mr-2" />
                                Loading...
                            </div>
                        ) : filteredItems.length === 0 ? (
                            <div className="py-2 px-2 text-xs text-muted-foreground text-center">
                                No suggestions found.
                                {search && (
                                    <div className="mt-1 font-medium text-foreground">
                                        Use "{search}"
                                    </div>
                                )}
                            </div>
                        ) : (
                            filteredItems.map((item) => (
                                <div
                                    key={item}
                                    className={cn(
                                        "relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
                                        value === item && "bg-accent/50"
                                    )}
                                    onClick={() => handleSelect(item)}
                                >
                                    <Check
                                        className={cn(
                                            "mr-2 h-4 w-4",
                                            value === item ? "opacity-100" : "opacity-0"
                                        )}
                                    />
                                    {item}
                                </div>
                            ))
                        )}
                    </div>
                </ScrollArea>
            </PopoverContent>
        </Popover>
    )
}
