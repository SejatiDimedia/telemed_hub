import { apiClient } from "../../../lib/api-client";
import type { PaginatedResult } from "../../../lib/api-client";
import type { Wallet, WalletTransaction, TopUpRequest, TopUpMidtransResponse } from "../types";

export const walletApi = {
  getWallet: (): Promise<Wallet> => {
    return apiClient.get<Wallet>("/wallet");
  },

  listTransactions: (page = 1, pageSize = 10): Promise<PaginatedResult<WalletTransaction[]>> => {
    return apiClient.getWithPagination<WalletTransaction[]>(
      `/wallet/transactions?page=${page}&page_size=${pageSize}`,
    );
  },

  topUp: (amount: number, idempotencyKey: string): Promise<TopUpMidtransResponse> => {
    const body: TopUpRequest = { amount };
    return apiClient.post<TopUpMidtransResponse>("/wallet/top-up", body, {
      headers: {
        "Idempotency-Key": idempotencyKey,
      },
    });
  },
};
