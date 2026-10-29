import { clsx } from 'clsx'
import type { InputHTMLAttributes } from 'react'
import { forwardRef } from 'react'

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  helpText?: string
}

export const Input = forwardRef<HTMLInputElement, Props>(
  ({ label, error, helpText, className, id, ...props }, ref) => {
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, '-')
    return (
      <div className="flex flex-col gap-1">
        {label && (
          <label htmlFor={inputId} className="text-sm font-medium text-gray-700 dark:text-gray-300">
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          {...props}
          className={clsx(
            'block w-full rounded-md border px-3 py-2 text-sm shadow-sm',
            'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
            'disabled:cursor-not-allowed disabled:bg-gray-50 disabled:text-gray-500',
            'dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400',
            error
              ? 'border-red-400 focus:ring-red-500 dark:border-red-500'
              : 'border-gray-300 dark:border-gray-600',
            className,
          )}
        />
        {error && <p className="text-xs text-red-600 dark:text-red-400">{error}</p>}
        {helpText && !error && <p className="text-xs text-gray-500 dark:text-gray-400">{helpText}</p>}
      </div>
    )
  },
)
Input.displayName = 'Input'
