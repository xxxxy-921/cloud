import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { api, type PaginatedResponse } from "@/lib/api"

interface UseListPageOptions {
  queryKey: string
  endpoint: string
  pageSize?: number
  extraParams?: Record<string, string>
}

export function useListPage<T>({ queryKey, endpoint, pageSize = 20, extraParams }: UseListPageOptions) {
  const [keyword, setKeyword] = useState("")
  const [searchKeyword, setSearchKeyword] = useState("")
  const [page, setPage] = useState(1)

  const { data, isLoading } = useQuery({
    queryKey: [queryKey, searchKeyword, page, extraParams],
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) })
      if (searchKeyword) params.set("keyword", searchKeyword)
      if (extraParams) {
        for (const [k, v] of Object.entries(extraParams)) {
          params.set(k, v)
        }
      }
      return api.get<PaginatedResponse<T>>(`${endpoint}?${params}`)
    },
  })

  function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    setSearchKeyword(keyword)
    setPage(1)
  }

  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / pageSize)

  return {
    keyword,
    setKeyword,
    page,
    setPage,
    items,
    total,
    totalPages,
    isLoading,
    handleSearch,
  }
}
