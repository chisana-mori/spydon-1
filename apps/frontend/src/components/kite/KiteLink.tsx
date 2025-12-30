'use client'

import { ExternalLink, Box } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { getKiteUrl, isKiteEnabled } from '@/config'

interface KiteLinkProps {
  clusterName: string
  resource?: string
  namespace?: string
  resourceName?: string
  variant?: 'button' | 'icon' | 'link'
  size?: 'sm' | 'default' | 'lg'
  className?: string
}

/**
 * Kite Dashboard 跳转组件
 * 提供从 Spydon 跳转到 Kite 进行集群操作的能力
 */
export function KiteLink({
  clusterName,
  resource,
  namespace,
  resourceName,
  variant = 'button',
  size = 'sm',
  className = '',
}: KiteLinkProps) {
  if (!isKiteEnabled()) {
    return null
  }

  const url = getKiteUrl({ clusterName, resource, namespace, resourceName })

  const handleClick = () => {
    window.open(url, '_blank', 'noopener,noreferrer')
  }

  const tooltipText = resource
    ? `在 Kite 中查看 ${resource}`
    : `在 Kite Dashboard 中管理集群 ${clusterName}`

  if (variant === 'icon') {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={handleClick}
              className={`h-8 w-8 text-muted-foreground hover:text-primary ${className}`}
            >
              <Box className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            <p>{tooltipText}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    )
  }

  if (variant === 'link') {
    return (
      <button
        onClick={handleClick}
        className={`inline-flex items-center gap-1 text-sm text-blue-600 hover:text-blue-800 hover:underline transition-colors ${className}`}
      >
        <Box className="h-3.5 w-3.5" />
        <span>Kite</span>
        <ExternalLink className="h-3 w-3" />
      </button>
    )
  }

  // Default: button variant
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="outline"
            size={size}
            onClick={handleClick}
            className={`gap-2 ${className}`}
          >
            <Box className="h-4 w-4" />
            <span>Kite</span>
            <ExternalLink className="h-3 w-3" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          <p>{tooltipText}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

/**
 * 集群名称带 Kite 链接的组件
 * 用于在表格或列表中显示集群名称，并提供快速跳转到 Kite 的能力
 */
export function ClusterNameWithKite({
  clusterName,
  className = '',
}: {
  clusterName: string
  className?: string
}) {
  if (!isKiteEnabled()) {
    return <span className={className}>{clusterName}</span>
  }

  const url = getKiteUrl({ clusterName })

  return (
    <span className={`inline-flex items-center gap-2 ${className}`}>
      <span>{clusterName}</span>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              onClick={() => window.open(url, '_blank', 'noopener,noreferrer')}
              className="inline-flex items-center justify-center h-5 w-5 rounded hover:bg-muted transition-colors"
            >
              <Box className="h-3.5 w-3.5 text-muted-foreground hover:text-primary" />
            </button>
          </TooltipTrigger>
          <TooltipContent>
            <p>在 Kite 中管理此集群</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </span>
  )
}
