import * as React from "react";
import { cn } from "@/lib/utils";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(({ className, ...props }, ref) => {
  return (
    <input
      className={cn(
        "flex h-10 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none",
        "focus:border-panel-500 focus:ring-2 focus:ring-panel-200",
        className
      )}
      ref={ref}
      {...props}
    />
  );
});

Input.displayName = "Input";
