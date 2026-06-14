import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function centsToDisplay(cents: number): string {
  return (cents / 10000).toLocaleString("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
  })
}

export function centsToDollars(cents: number): number {
  return cents / 10000
}

export function formatPercent(pct: number): string {
  return pct.toFixed(1) + "%"
}

export function formatNumber(n: number): string {
  return n.toLocaleString("en-US")
}
