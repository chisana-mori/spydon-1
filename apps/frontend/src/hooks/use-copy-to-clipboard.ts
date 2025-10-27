import { useState, useCallback } from 'react'

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

  const copyToClipboard = useCallback(async (text: string): Promise<boolean> => {
    if (!navigator?.clipboard) {
      return false
    }

    try {
      await navigator.clipboard.writeText(text)
      setIsCopied(true)
      
      setTimeout(() => {
        setIsCopied(false)
      }, timeout)
      
      return true
    } catch (error) {
      setIsCopied(false)
      return false
    }
  }, [timeout])

  return { isCopied, copyToClipboard }
}
