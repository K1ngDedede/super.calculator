import { useQuery } from '@tanstack/react-query'
import { fetchOperations } from '../api'

export function useOperations() {
  return useQuery({
    queryKey: ['operations'],
    queryFn: fetchOperations,
    staleTime: Infinity,
    gcTime: Infinity,
    retry: false,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  })
}
