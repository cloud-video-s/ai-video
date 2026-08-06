export type OrderStatusTagType = '' | 'success' | 'warning' | 'info' | 'danger' | 'primary'

export const orderStatusOptions = [
  { value: 1, label: '待支付' },
  { value: 2, label: '支付中' },
  { value: 3, label: '支付成功' },
  { value: 4, label: '订单完成' },
  { value: 5, label: '已取消' },
  { value: 6, label: '支付失败' },
  { value: 7, label: '已退款' },
]

export function orderStatusLabel(value: number) {
  return orderStatusOptions.find((item) => item.value === value)?.label || `未知（${value}）`
}

export function orderStatusType(value: number): OrderStatusTagType {
  if (value === 3 || value === 4) return 'success'
  if (value === 1 || value === 2) return 'warning'
  if (value === 5) return 'info'
  if (value === 7) return 'primary'
  if (value === 6) return 'danger'
  return ''
}
