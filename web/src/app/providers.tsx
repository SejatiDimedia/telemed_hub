import type { ReactNode } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { queryClient } from "../lib/query-client";
import { AuthProvider } from "../context/auth-context";

interface ProvidersProps {
  children: ReactNode;
}

/**
 * All providers dikumpulkan di satu tempat — sesuai docs/17-frontend-folder-structure.md
 */
export function Providers({ children }: ProvidersProps) {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>{children}</AuthProvider>
    </QueryClientProvider>
  );
}
