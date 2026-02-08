import * as React from "react"

import { cn } from "@/app/lib/utils"

export interface TextareaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        className={cn(
          "flex min-h-[44px] w-full rounded-md border-0",
          "bg-transparent px-3 py-2 text-base shadow-none transition-colors appearance-none outline-none",
          "placeholder:text-[--color-muted-foreground]",
          "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[--color-ring]",
          "disabled:cursor-not-allowed disabled:opacity-50",
          "md:text-sm resize-none",
          className
        )}
        ref={ref}
        {...props}
      />
    )
  }
)
Textarea.displayName = "Textarea"

export { Textarea }
