"use client"

import * as React from "react"
import { Check } from "lucide-react"
import { cn } from "@/lib/utils"

export interface CheckboxProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange'> {
  checked?: boolean
  onCheckedChange?: (checked: boolean) => void
}

const Checkbox = React.forwardRef<HTMLInputElement, CheckboxProps>(
  ({ className, checked, onCheckedChange, disabled, ...props }, ref) => {
    const handleClick = (e: React.MouseEvent) => {
      e.stopPropagation()
      if (!disabled) {
        onCheckedChange?.(!checked)
      }
    }

    return (
      <div
        className={cn(
          "relative inline-flex items-center cursor-pointer",
          disabled && "cursor-not-allowed opacity-50"
        )}
        onClick={handleClick}
      >
        <input
          type="checkbox"
          className="sr-only"
          ref={ref}
          checked={checked}
          onChange={(e) => onCheckedChange?.(e.target.checked)}
          disabled={disabled}
          {...props}
        />
        <div
          className={cn(
            "h-4 w-4 rounded border border-gray-300 flex items-center justify-center transition-colors",
            checked ? "bg-black border-black" : "bg-white hover:border-gray-400",
            className
          )}
        >
          {checked && <Check className="h-3 w-3 text-white" />}
        </div>
      </div>
    )
  }
)
Checkbox.displayName = "Checkbox"

export { Checkbox }
