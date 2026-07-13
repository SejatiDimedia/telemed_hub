import { apiClient } from "../../../lib/api-client";
import type { Wallet, WalletTransaction, TopUpRequest } from "../types";

export const walletApi = {
  getWallet: (): Promise<Wallet> => {
    return apiClient.get<Wallet>("/wallet");
  },

  listTransactions: (): Promise<WalletTransaction[]> => {
    return apiClient.get<WalletTransaction[]>("/wallet/transactions");
  },

  topUp: (amount: number, idempotencyKey: string): Promise<WalletTransaction> => {
    const body: TopUpRequest = { amount };
    return apiClient.post<WalletTransaction>("/wallet/top-up", body, {
      headers: {
        "Idempotency-Key": idempotencyKey,
      },
    });
  },
};
