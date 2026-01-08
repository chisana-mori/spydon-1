"use client"

import { useQueries } from "@tanstack/react-query"
import { getDictionaryItemsByCode } from "@/lib/api/dictionaries"

/**
 * 批量预加载多个字典
 * 使用 useQueries 并行获取所有字典，避免串行请求
 */
export function useDictionaryPreload(codes: string[]) {
    const queries = useQueries({
        queries: codes.map((code) => ({
            queryKey: ["dictionary", code],
            queryFn: () => getDictionaryItemsByCode(code),
            staleTime: 1000 * 60 * 5, // 5 minutes
        })),
    })

    const isLoading = queries.some((q) => q.isLoading)
    const isError = queries.some((q) => q.isError)

    return {
        isLoading,
        isError,
        queries,
    }
}

/**
 * 集群编辑页面需要的所有字典 codes
 */
export const CLUSTER_DICTIONARY_CODES = [
    "cluster_version",
    "idc",
    "zone",
    "flow_type",
    "purpose",
    "arch",
    "cluster_group",
] as const
