import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { walletApi } from "../api/wallet-api";
import { useToastStore } from "../../../stores/toast-store";

export const walletKeys = {
  all: ["wallet"] as const,
  profile: () => [...walletKeys.all, "details"] as const,
  transactions: (page: number) => [...walletKeys.all, "transactions", page] as const,
};

export function useWallet() {
  return useQuery({
    queryKey: walletKeys.profile(),
    queryFn: walletApi.getWallet,
    staleTime: 1 * 60 * 1000, // 1 minute cache
  });
}

export function useWalletTransactions(page = 1, pageSize = 10) {
  return useQuery({
    queryKey: walletKeys.transactions(page),
    queryFn: () => walletApi.listTransactions(page, pageSize),
    staleTime: 30 * 1000, // 30 seconds cache
    placeholderData: (prev) => prev, // Keep previous data while loading next page
  });
}

export function useTopUpWallet() {
  const queryClient = useQueryClient();
  const addToast = useToastStore((state) => state.addToast);

  return useMutation({
    mutationFn: ({ amount, idempotencyKey }: { amount: number; idempotencyKey: string }) =>
      walletApi.topUp(amount, idempotencyKey),
    onSuccess: (data) => {
      window.location.href = data.redirect_url;
    },
    onError: () => {
      addToast({
        type: "error",
        title: "Top-up Gagal",
        message: "Terjadi kesalahan saat memproses permintaan top-up. Silakan coba lagi.",
      });
    },
  });
}
