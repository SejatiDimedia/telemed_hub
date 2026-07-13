import type { ButtonHTMLAttributes, ReactNode } from "react";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "outline" | "text";
  size?: "sm" | "md" | "lg";
  isLoading?: boolean;
  leftIcon?: string;
  rightIcon?: string;
  children: ReactNode;
}

export function Button({
  variant = "primary",
  size = "md",
  isLoading = false,
  leftIcon,
  rightIcon,
  children,
  className = "",
  disabled,
  ...props
}: ButtonProps) {
  // Base classes
  const baseClasses =
    "inline-flex items-center justify-center font-body font-semibold rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 active:scale-[0.98] disabled:opacity-50 disabled:pointer-events-none disabled:active:scale-100";

  // Variant classes
  const variantClasses = {
    primary:
      "bg-primary text-on-primary hover:bg-primary/95 focus:ring-primary/50 shadow-sm",
    secondary:
      "bg-secondary text-on-secondary hover:bg-secondary/95 focus:ring-secondary/50 shadow-sm",
    outline:
      "border border-outline text-primary hover:bg-primary-container/10 focus:ring-primary/50",
    text: "text-primary hover:bg-primary-container/10 focus:ring-primary/50",
  };

  // Size classes
  const sizeClasses = {
    sm: "px-3 py-1.5 text-label-sm gap-1.5",
    md: "px-4 py-2 text-label-md gap-2",
    lg: "px-6 py-3 text-body-md gap-2.5",
  };

  return (
    <button
      disabled={disabled || isLoading}
      className={`${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${className}`}
      {...props}
    >
      {isLoading && (
        <svg
          className="animate-spin -ml-1 mr-2 h-4 w-4 text-current"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          />
        </svg>
      )}

      {!isLoading && leftIcon && (
        <span className="material-symbols-outlined text-[1.25em]">{leftIcon}</span>
      )}

      <span>{children}</span>

      {!isLoading && rightIcon && (
        <span className="material-symbols-outlined text-[1.25em]">{rightIcon}</span>
      )}
    </button>
  );
}
