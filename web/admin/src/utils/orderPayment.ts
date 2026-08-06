export const orderPaymentOptions = [
  { value: 1, label: 'Apple IAP' },
  { value: 2, label: 'Google Play' },
]

export function orderPaymentLabel(value: number) {
  return orderPaymentOptions.find((item) => item.value === value)?.label || `未知（${value}）`
}
