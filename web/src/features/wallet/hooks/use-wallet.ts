import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { walletApi } from "../api/wallet-api";
import { useToastStore } from "../../../stores/toast-store";

export const walletKeys = {
  all: ["wallet"] as const,
  profile: () => [...walletKeys.all, "details"] as const,
  transactions: () => [...walletKeys.all, "transactions"] as const,
};

export function useWallet() {
  return useQuery({
    queryKey: walletKeys.profile(),
    queryFn: walletApi.getWallet,
    staleTime: 1 * 60 * 1000, // 1 minute cache
  });
}

export function useWalletTransactions() {
  return useQuery({
    queryKey: walletKeys.transactions(),
    queryFn: walletApi.listTransactions,
    staleTime: 30 * 1000, // 30 seconds cache
  });
}

export function useTopUpWallet() {
  const queryClient = useQueryClient();
  const addToast = useToastStore((state) => state.addToast);

  return useMutation({
    mutationFn: ({ amount, idempotencyKey }: { amount: number; idempotencyKey: string }) =>
      walletApi.topUp(amount, idempotencyKey),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: walletKeys.all });
      addToast({
        type: "success",
        title: "Top-up Berhasil",
        message: `Saldo sebesar Rp ${data.amount.toLocaleString("id-ID")} telah ditambahkan ke dompet digital Anda.`,
      });
    },
  });
}
