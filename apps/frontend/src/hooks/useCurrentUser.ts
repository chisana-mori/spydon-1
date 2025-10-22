'use client'

import { useCallback, useEffect, useState } from 'react'
import RobustaAPI from '@/lib/api'
import type { User } from '@/types/api'

interface UseCurrentUserResult {
  user: User | null
  loading: boolean
  error: string | null
  refresh: () => Promise<void>
}

export function useCurrentUser(): UseCurrentUserResult {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchProfile = useCallback(async () => {
    setLoading(true)
    setError(null)

    try {
      const profile = await RobustaAPI.getProfile()
      setUser(profile)
    } catch (err: any) {
      setUser(null)
      setError(err?.message ?? '获取用户信息失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void fetchProfile()
  }, [fetchProfile])

  return {
    user,
    loading,
    error,
    refresh: fetchProfile,
  }
}
