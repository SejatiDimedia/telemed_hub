export interface Wallet {
  balance: number;
  currency: string;
}

export interface WalletTransaction {
  id: string;
  type: "top_up" | "order_payment" | "refund" | "appointment_fee" | string;
  amount: number;
  reference_id?: string | null;
  balance_after: number;
  created_at: string;
}
export interface TopUpRequest {
  amount: number;
}
