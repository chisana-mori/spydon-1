import { useQuery } from '@tanstack/react-query';
import { getDictionaryItemsByCode } from '@/lib/api/dictionaries';
import { useMemo } from 'react';

export function useDictionary(code: string) {
    const { data: items = [], isLoading, error } = useQuery({
        queryKey: ['dictionary', code],
        queryFn: () => getDictionaryItemsByCode(code),
        staleTime: 5 * 60 * 1000, // cache for 5 minutes
        enabled: !!code,
    });

    const dict = useMemo(() => {
        const map: Record<string | number, string> = {};
        items.forEach((item) => {
            // Support both string and number keys (dictionaries usually use string values for keys, strict typing might need adjustment)
            // But here we want to map "Value" to "Label" usually? 
            // Wait, standard dict items have: Key (string), Value (string).
            // In our DB seed: 
            // label_source: key="0", value="内部"
            // So we want to map implicit "0" (int) from backend model to "内部".
            // The item.key is the "value" stored in DB (e.g. 0, 1). item.value is the display label.
            map[item.key] = item.value;
            // Also map as number if applicable, for easier lookup
            if (!isNaN(Number(item.key))) {
                map[Number(item.key)] = item.value;
            }
        });
        return map;
    }, [items]);

    return { items, dict, isLoading, error };
}
