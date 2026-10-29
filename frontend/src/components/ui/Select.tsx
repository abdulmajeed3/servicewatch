import { clsx } from 'clsx'
import type { SelectHTMLAttributes } from 'react'
import { forwardRef } from 'react'

interface Props extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string
  error?: string
  helpText?: string
}

export const Select = forwardRef<HTMLSelectElement, Props>(
  ({ label, error, helpText, className, id, children, ...props }, ref) => {
    const selectId = id ?? label?.toLowerCase().replace(/\s+/g, '-')
    return (
      <div className="flex flex-col gap-1">
        {label && (
          <label htmlFor={selectId} className="text-sm font-medium text-gray-700 dark:text-gray-300">
            {label}
          </label>
        )}
        <select
          ref={ref}
          id={selectId}
          {...props}
          className={clsx(
            'block w-full rounded-md border px-3 py-2 text-sm shadow-sm',
            'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
            'dark:bg-gray-800 dark:text-gray-100',
            error ? 'border-red-400 dark:border-red-500' : 'border-gray-300 dark:border-gray-600',
            className,
          )}
        >
          {children}
        </select>
        {error && <p className="text-xs text-red-600 dark:text-red-400">{error}</p>}
        {helpText && !error && <p className="text-xs text-gray-500 dark:text-gray-400">{helpText}</p>}
      </div>
    )
  },
)
Select.displayName = 'Select'
