import { createFileRoute } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as zod from "zod";
import { useWallet, useWalletTransactions, useTopUpWallet, walletKeys } from "../../../features/wallet/hooks/use-wallet";
import { walletApi } from "../../../features/wallet/api/wallet-api";
import { usePatientProfile } from "../../../features/patient/hooks/use-patient-profile";
import { Card } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Badge } from "../../../components/ui/Badge";
import { EmptyState } from "../../../components/shared/EmptyState";
import { Dialog } from "../../../components/ui/Dialog";
import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

export const Route = createFileRoute("/patient/wallet")({
  component: PatientWalletPage,
});

const topUpSchema = zod.object({
  amount: zod
    .number({ message: "Jumlah top-up harus berupa angka" })
    .min(10000, "Jumlah top-up minimal Rp 10.000")
    .max(10000000, "Jumlah top-up maksimal Rp 10.000.000"),
});

type TopUpSchemaType = zod.infer<typeof topUpSchema>;

function PatientWalletPage() {
  const queryClient = useQueryClient();
  const { data: wallet, isLoading: isWalletLoading } = useWallet();
  const { data: transactions, isLoading: isTransactionsLoading } = useWalletTransactions();
  const { data: profile } = usePatientProfile();
  const { mutateAsync: topUp, isPending: isTopUpPending } = useTopUpWallet();

  const [quickAmounts] = useState([50000, 100000, 200000, 500000]);

  // Confirmation dialog state
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [pendingAmount, setPendingAmount] = useState<number | null>(null);

  // Idempotency simulation state
  const [isSimulating, setIsSimulating] = useState(false);
  const [simulationResult, setSimulationResult] = useState<{
    key: string;
    req1: string;
    req2: string;
    status: "success" | "failed" | null;
  } | null>(null);

  const {
    register,
    handleSubmit,
    setValue,
    reset,
    formState: { errors },
  } = useForm<TopUpSchemaType>({
    resolver: zodResolver(topUpSchema),
    defaultValues: {
      amount: 100000,
    },
  });

  const onSubmit = (data: TopUpSchemaType) => {
    // Show confirmation dialog instead of submitting immediately
    setPendingAmount(data.amount);
    setIsConfirmOpen(true);
  };

  const handleConfirmTopUp = async () => {
    if (pendingAmount === null) return;
    setIsConfirmOpen(false);
    try {
      const idempotencyKey = crypto.randomUUID();
      await topUp({ amount: pendingAmount, idempotencyKey });
      reset();
    } catch {
      // Toast notification handled
    } finally {
      setPendingAmount(null);
    }
  };

  const runIdempotencySimulation = async () => {
    setIsSimulating(true);
    setSimulationResult(null);
    const testKey = crypto.randomUUID();
    const testAmount = 50000;

    try {
      // Send two parallel top-up requests with the exact same Idempotency-Key
      const [res1, res2] = await Promise.allSettled([
        walletApi.topUp(testAmount, testKey),
        walletApi.topUp(testAmount, testKey),
      ]);

      const r1Text =
        res1.status === "fulfilled"
          ? `Request 1: Berhasil (ID: ${res1.value.id.slice(0, 8)}, Saldo Baru: ${formatCurrency(res1.value.balance_after)})`
          : `Request 1: Gagal (${res1.reason instanceof Error ? res1.reason.message : String(res1.reason)})`;

      const r2Text =
        res2.status === "fulfilled"
          ? `Request 2: Berhasil (Duplicate Response, ID: ${res2.value.id.slice(0, 8)}, Saldo: ${formatCurrency(res2.value.balance_after)})`
          : `Request 2: Gagal (${res2.reason instanceof Error ? res2.reason.message : String(res2.reason)})`;

      setSimulationResult({
        key: testKey,
        req1: r1Text,
        req2: r2Text,
        status: "success",
      });

      // Refresh wallet profile and ledger list
      void queryClient.invalidateQueries({ queryKey: walletKeys.all });
    } catch (err) {
      setSimulationResult({
        key: testKey,
        req1: "Error",
        req2: "Error",
        status: "failed",
      });
    } finally {
      setIsSimulating(false);
    }
  };

  const patientName = profile?.full_name ?? "Patient Account";
  const currentBalance = wallet?.balance ?? 0;

  const formatCurrency = (val: number) => {
    return new Intl.NumberFormat("id-ID", {
      style: "currency",
      currency: "IDR",
      minimumFractionDigits: 0,
    }).format(val);
  };

  const getTransactionBadge = (type: string) => {
    switch (type) {
      case "top_up":
        return <Badge variant="success">TOP UP</Badge>;
      case "appointment_fee":
        return <Badge variant="primary">CONSULTATION</Badge>;
      case "order_payment":
        return <Badge variant="secondary">PHARMACY</Badge>;
      case "refund":
        return <Badge variant="info">REFUND</Badge>;
      default:
        return <Badge variant="neutral">{type.toUpperCase()}</Badge>;
    }
  };

  return (
    <div className="flex flex-col gap-8">
      {/* Page Header */}
      <section className="flex flex-col md:flex-row justify-between items-end gap-6 select-none">
        <div className="max-w-2xl">
          <h1 className="font-display text-headline-lg text-primary mb-2 font-bold">Billing Center</h1>
          <p className="font-body text-body-lg text-on-surface-variant leading-relaxed">
            Manage your digital wallet balance, view detailed transaction ledgers, and check prescription billing records.
          </p>
        </div>
        <div className="flex gap-4">
          <Button
            variant="outline"
            leftIcon="ios_share"
            onClick={() => alert("Mengekspor laporan keuangan...")}
            className="rounded-card border-outline-variant hover:bg-surface-container"
          >
            Export Statement
          </Button>
        </div>
      </section>

      {/* Financial Bento Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left Column: Virtual Card & Top Up Form */}
        <div className="col-span-12 lg:col-span-5 flex flex-col gap-6">
          {/* Reusable Card: Premium Virtual Debit Card */}
          {isWalletLoading ? (
            <div className="h-56 w-full bg-surface-container-lowest rounded-card animate-pulse border border-outline-variant/20"></div>
          ) : (
            <Card className="relative overflow-hidden bg-gradient-to-br from-primary via-primary-container to-secondary text-white p-8 h-56 rounded-card shadow-level-2 border-none flex flex-col justify-between select-none">
              {/* Card Aura Backgrounds */}
              <div className="absolute -right-6 -top-6 w-32 h-32 bg-white/10 rounded-full blur-2xl"></div>
              <div className="absolute -left-6 -bottom-6 w-40 h-40 bg-white/5 rounded-full blur-3xl"></div>

              <div className="relative z-10 flex justify-between items-start w-full">
                <div>
                  <p className="text-xs uppercase tracking-widest text-white/75 font-semibold">Digital Wallet Balance</p>
                  <h2 className="text-[32px] font-bold mt-1 tracking-tight">
                    {formatCurrency(currentBalance)}
                  </h2>
                </div>
                <span className="material-symbols-outlined text-[36px] text-white/90">
                  contactless
                </span>
              </div>

              <div className="relative z-10 flex justify-between items-end w-full">
                <div>
                  <p className="text-[10px] uppercase tracking-wider text-white/60 font-semibold mb-1">Card Holder</p>
                  <p className="text-sm font-bold tracking-wide truncate max-w-[200px] uppercase">{patientName}</p>
                </div>
                <div className="text-right">
                  <p className="text-[10px] uppercase tracking-wider text-white/60 font-semibold mb-1">Provider</p>
                  <p className="text-sm font-bold tracking-widest">TELEMEDHUB</p>
                </div>
              </div>
            </Card>
          )}

          {/* Reusable Card: Top Up Balance Form */}
          <Card variant="elevation" className="p-6 border border-outline-variant/10">
            <h4 className="text-headline-md font-bold text-on-surface mb-4">Top-Up Saldo</h4>
            <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-5">
              <div>
                <label className="block text-label-sm font-bold text-on-surface-variant mb-2">
                  Pilih Nominal Cepat
                </label>
                <div className="grid grid-cols-4 gap-2">
                  {quickAmounts.map((amt) => (
                    <button
                      key={amt}
                      type="button"
                      onClick={() => setValue("amount", amt, { shouldValidate: true })}
                      className="py-2.5 rounded-lg border border-outline-variant/50 text-xs font-semibold text-on-surface hover:bg-primary/5 hover:border-primary active:scale-[0.98] transition-all select-none"
                    >
                      {amt / 1000}k
                    </button>
                  ))}
                </div>
              </div>

              <div className="h-px bg-outline-variant/20 w-full"></div>

              <div>
                <label htmlFor="amount" className="block text-label-sm font-bold text-on-surface-variant mb-2">
                  Atau Masukkan Nominal Kustom
                </label>
                <div className="relative flex items-center">
                  <span className="absolute left-4 text-sm font-bold text-on-surface-variant select-none">
                    Rp
                  </span>
                  <input
                    id="amount"
                    type="number"
                    {...register("amount", { valueAsNumber: true })}
                    className="w-full bg-surface-container-low border border-outline-variant/50 rounded-xl py-3 pl-12 pr-4 text-sm font-bold focus:ring-1 focus:ring-primary focus:border-primary outline-none transition-all"
                    placeholder="0"
                  />
                </div>
                {errors.amount && (
                  <p className="text-xs text-error mt-2 font-semibold select-none">{errors.amount.message}</p>
                )}
              </div>

              <Button
                type="submit"
                isLoading={isTopUpPending}
                leftIcon="add_circle"
                className="w-full py-3.5 rounded-xl font-bold mt-2 shadow-level-1"
              >
                Top-Up Sekarang
              </Button>
            </form>
          </Card>

          {/* Idempotency Test Simulation Card */}
          <Card variant="elevation" className="p-6 border border-outline-variant/10">
            <div className="flex justify-between items-center mb-3 select-none">
              <h4 className="text-headline-md font-bold text-on-surface">Uji Idempotensi</h4>
              <Badge variant="info">Sandbox</Badge>
            </div>
            <p className="text-body-sm text-on-surface-variant mb-4 select-none">
              Simulasi pengiriman dua request top-up secara paralel menggunakan **Idempotency-Key** yang sama. Membuktikan saldo hanya bertambah sekali.
            </p>
            <Button
              variant="outline"
              leftIcon="double_arrow"
              isLoading={isSimulating}
              onClick={runIdempotencySimulation}
              className="w-full py-3 rounded-xl font-bold border-primary text-primary hover:bg-primary/5 select-none"
            >
              Jalankan Simulasi Double-Submit
            </Button>

            {simulationResult && (
              <div className="mt-4 p-4 rounded-xl bg-surface-container-low/50 border border-outline-variant/20 flex flex-col gap-2 font-mono text-xs text-on-surface-variant select-text">
                <p className="font-semibold text-on-surface text-label-sm font-sans mb-1 select-none">
                  Hasil Simulasi:
                </p>
                <p className="truncate"><span className="font-bold select-none text-primary">Key:</span> {simulationResult.key}</p>
                <p className="text-green-600 font-semibold">{simulationResult.req1}</p>
                <p className="text-amber-600 font-semibold">{simulationResult.req2}</p>
                <div className="mt-2 text-[10px] text-on-surface-variant font-sans select-none">
                  💡 *Request kedua mengembalikan respons duplikat secara instan tanpa menambah saldo lagi.*
                </div>
              </div>
            )}
          </Card>
        </div>

        {/* Right Column: Ledger Transaction Logs */}
        <div className="col-span-12 lg:col-span-7 flex flex-col gap-6">
          <Card variant="elevation" className="border border-outline-variant/10 overflow-hidden">
            <div className="p-6 border-b border-outline-variant/20 flex items-center justify-between">
              <h4 className="text-headline-md font-bold text-on-surface">Riwayat Transaksi Ledger</h4>
              <Badge variant="neutral">Append-Only Logs</Badge>
            </div>

            {isTransactionsLoading ? (
              <div className="p-8 flex flex-col gap-4 animate-pulse">
                <div className="h-10 bg-surface-container rounded"></div>
                <div className="h-10 bg-surface-container rounded"></div>
                <div className="h-10 bg-surface-container rounded"></div>
              </div>
            ) : transactions && transactions.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="w-full text-left">
                  <thead className="bg-surface-container-low text-on-surface-variant text-label-sm uppercase tracking-wider select-none">
                    <tr>
                      <th className="px-6 py-4 font-bold">Waktu</th>
                      <th className="px-6 py-4 font-bold">Tipe</th>
                      <th className="px-6 py-4 font-bold">Nominal</th>
                      <th className="px-6 py-4 font-bold">Saldo Akhir</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-outline-variant/10">
                    {transactions
                      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
                      .map((tx) => {
                        const isCredit = tx.type === "top_up" || tx.type === "refund";
                        const amountStr = isCredit
                          ? `+${formatCurrency(tx.amount)}`
                          : `-${formatCurrency(tx.amount)}`;

                        return (
                          <tr key={tx.id} className="hover:bg-surface-container-lowest/30 transition-colors">
                            <td className="px-6 py-4 text-body-sm text-on-surface-variant select-none">
                              {new Date(tx.created_at).toLocaleString("id-ID")}
                            </td>
                            <td className="px-6 py-4 select-none">
                              {getTransactionBadge(tx.type)}
                            </td>
                            <td
                              className={`px-6 py-4 font-bold ${isCredit ? "text-green-600" : "text-on-surface"
                                }`}
                            >
                              {amountStr}
                            </td>
                            <td className="px-6 py-4 text-body-sm text-on-surface-variant/80 select-none">
                              {formatCurrency(tx.balance_after)}
                            </td>
                          </tr>
                        );
                      })}
                  </tbody>
                </table>
              </div>
            ) : (
              <EmptyState
                icon="account_balance_wallet"
                title="Belum Ada Transaksi"
                description="Dompet digital Anda aktif dan siap digunakan. Lakukan top-up untuk mulai menjadwalkan konsultasi."
                className="border-none"
              />
            )}
          </Card>
        </div>
      </div>

      {/* Confirmation Dialog */}
      <Dialog
        isOpen={isConfirmOpen}
        onClose={() => setIsConfirmOpen(false)}
        title="Konfirmasi Pengisian Saldo"
        size="sm"
        footer={
          <div className="flex gap-3 justify-end w-full select-none">
            <Button
              variant="outline"
              onClick={() => setIsConfirmOpen(false)}
              className="px-6 py-2.5 rounded-full"
            >
              Batal
            </Button>
            <Button
              onClick={handleConfirmTopUp}
              className="px-6 py-2.5 rounded-full"
            >
              Konfirmasi
            </Button>
          </div>
        }
      >
        <div className="flex flex-col gap-3 py-2">
          <p className="text-body-md text-on-surface-variant leading-relaxed">
            Apakah Anda yakin ingin melakukan pengisian saldo sebesar:
          </p>
          <p className="text-headline-md font-bold text-primary select-all">
            {pendingAmount !== null ? formatCurrency(pendingAmount) : ""}
          </p>
          <p className="text-xs text-on-surface-variant/80 select-none">
            Dana akan ditambahkan secara aman ke saldo dompet digital TeleMedHub Anda.
          </p>
        </div>
      </Dialog>
    </div>
  );
}
