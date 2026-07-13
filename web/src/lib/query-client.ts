import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Data dianggap "fresh" selama 30 detik — mengurangi refetch berlebihan
      staleTime: 30 * 1000,
      // Retry 1x untuk network error, bukan infinite
      retry: 1,
      // Refetch saat user kembali ke tab browser
      refetchOnWindowFocus: true,
    },
    mutations: {
      retry: 0,
    },
  },
});
