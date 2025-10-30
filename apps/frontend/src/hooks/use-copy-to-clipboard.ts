import { useState, useCallback, useRef, useEffect } from 'react'
import { copyTextToClipboard } from '@/lib/clipboard'

interface UseCopyToClipboardProps {
  timeout?: number
}

interface UseCopyToClipboardReturn {
  isCopied: boolean
  copyToClipboard: (text: string) => Promise<boolean>
}

export function useCopyToClipboard({ 
  timeout = 2000 
}: UseCopyToClipboardProps = {}): UseCopyToClipboardReturn {
  const [isCopied, setIsCopied] = useState<boolean>(false)
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const copyToClipboard = useCallback(async (text: string): Promise<boolean> => {
    const success = await copyTextToClipboard(text)

    if (success) {
      setIsCopied(true)
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
      timeoutRef.current = setTimeout(() => {
        setIsCopied(false)
        timeoutRef.current = null
      }, timeout)
    } else {
      setIsCopied(false)
    }

    return success
  }, [timeout])

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
        timeoutRef.current = null
      }
    }
  }, [])

  return { isCopied, copyToClipboard }
}
