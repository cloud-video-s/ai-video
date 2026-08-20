import type { Ref } from 'vue'

type ElementSortOrder = 'ascending' | 'descending' | null

interface TableSortChange {
  prop: string
  order: ElementSortOrder
}

// useRemoteTableSort converts Element Plus sort events into the allowlisted
// query parameters understood by admin list endpoints.
export function useRemoteTableSort(
  page: Ref<number>,
  reload: () => unknown,
  aliases: Record<string, 'id' | 'sort'> = {},
) {
  let sortBy: 'id' | 'sort' | '' = ''
  let sortOrder: 'asc' | 'desc' | '' = ''

  function sortParams(): Record<string, string> {
    if (!sortBy || !sortOrder) return {}
    return { sort_by: sortBy, sort_order: sortOrder }
  }

  function handleSortChange({ prop, order }: TableSortChange) {
    const field = aliases[prop] || prop
    sortBy = field === 'id' || field === 'sort' ? field : ''
    sortOrder = order === 'ascending' ? 'asc' : order === 'descending' ? 'desc' : ''
    page.value = 1
    void reload()
  }

  return { sortParams, handleSortChange }
}
