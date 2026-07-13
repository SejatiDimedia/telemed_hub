import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { ApiError } from "../../../lib/api-client";
import { useDoctors, useDoctorAvailability } from "../../../features/doctor/hooks/use-doctors";
import { usePatientProfile, useUpdatePatientProfile } from "../../../features/patient/hooks/use-patient-profile";
import { useWallet } from "../../../features/wallet/hooks/use-wallet";
import { useBookAppointment, useAppointments, useCancelAppointment } from "../../../features/appointment/hooks/use-appointments";
import { Card } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Dialog } from "../../../components/ui/Dialog";
import { Avatar } from "../../../components/ui/Avatar";
import { Badge } from "../../../components/ui/Badge";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../../components/ui/Tabs";
import { EmptyState } from "../../../components/shared/EmptyState";
import { useState, useMemo, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as zod from "zod";

export const Route = createFileRoute("/patient/appointments")({
  component: PatientAppointmentsPage,
});

// Validation schema for quick profile completion
const profileCompletionSchema = zod.object({
  date_of_birth: zod.string().min(1, "Tanggal lahir wajib diisi"),
  gender: zod.enum(["male", "female"]),
  blood_type: zod.enum(["A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"]),
  phone_number: zod
    .string()
    .min(1, "Nomor telepon wajib diisi")
    .regex(/^\+?[1-9]\d{1,14}$/, "Format nomor telepon tidak valid (E.164)"),
});

type ProfileCompletionFormValues = zod.infer<typeof profileCompletionSchema>;

function PatientAppointmentsPage() {
  const navigate = useNavigate();

  // Selected doctor & slot state
  const [selectedDoctorId, setSelectedDoctorId] = useState<string | null>(null);
  const [selectedSlotId, setSelectedSlotId] = useState<string | null>(null);
  const [isCheckoutOpen, setIsCheckoutOpen] = useState(false);

  // Cancellation dialog state
  const [isCancelDialogOpen, setIsCancelDialogOpen] = useState(false);
  const [cancellingId, setCancellingId] = useState<string | null>(null);
  const [cancelReason, setCancelReason] = useState("");
  const [cancelError, setCancelError] = useState("");

  // Search input
  const [searchQuery, setSearchQuery] = useState("");

  // Data queries
  const { data: doctors, isLoading: isDoctorsLoading } = useDoctors();
  const { data: profile } = usePatientProfile();
  const { data: wallet } = useWallet();
  const { data: availability, isLoading: isAvailabilityLoading } = useDoctorAvailability(
    selectedDoctorId ?? ""
  );
  const { data: appointments, isLoading: isAppointmentsLoading } = useAppointments();

  // Mutations
  const { mutateAsync: updateProfile, isPending: isUpdatingProfile } = useUpdatePatientProfile();
  const { mutateAsync: bookAppointment, isPending: isBookingPending } = useBookAppointment();
  const { mutateAsync: cancelAppointment, isPending: isCancellingPending } = useCancelAppointment();

  // Profile completion form
  const {
    register,
    handleSubmit: handleProfileSubmit,
    reset,
    setError,
    formState: { errors: profileErrors },
  } = useForm<ProfileCompletionFormValues>({
    resolver: zodResolver(profileCompletionSchema),
    defaultValues: {
      blood_type: "O+",
    },
  });

  // Pre-fill profile completion form with existing profile data
  useEffect(() => {
    if (profile) {
      reset({
        date_of_birth: profile.date_of_birth ?? "",
        gender: (profile.gender as "male" | "female") ?? undefined,
        blood_type: (profile.blood_type as any) ?? "O+",
        phone_number: profile.phone_number ?? "",
      });
    }
  }, [profile, reset]);

  // Check if profile is complete (date_of_birth & gender must exist)
  const isProfileComplete = useMemo(() => {
    return !!profile?.date_of_birth && !!profile?.gender;
  }, [profile]);

  // Currency Formatter
  const formatCurrency = (val: number) => {
    return new Intl.NumberFormat("id-ID", {
      style: "currency",
      currency: "IDR",
      minimumFractionDigits: 0,
    }).format(val);
  };

  // Date/Time Formatters
  const formatTime = (isoString: string) => {
    try {
      return new Date(isoString).toLocaleTimeString("id-ID", {
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
      });
    } catch {
      return "00:00";
    }
  };

  const formatDateHeader = (isoString: string) => {
    try {
      return new Date(isoString).toLocaleDateString("id-ID", {
        weekday: "long",
        day: "numeric",
        month: "long",
      });
    } catch {
      return "Jadwal";
    }
  };

  // Filter doctors based on search
  const filteredDoctors = useMemo(() => {
    if (!doctors) return [];
    return doctors.filter(
      (doc) =>
        doc.full_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (doc.specialty && doc.specialty.toLowerCase().includes(searchQuery.toLowerCase()))
    );
  }, [doctors, searchQuery]);

  // Group doctor availability slots by date
  const groupedAvailability = useMemo(() => {
    if (!availability) return {};
    const unbookedSlots = availability.filter((slot) => !slot.is_booked);

    const groups: Record<string, typeof unbookedSlots> = {};
    for (const slot of unbookedSlots) {
      try {
        const parts = new Date(slot.start_time).toISOString().split("T");
        const dateKey = parts[0] || "unknown";
        if (!groups[dateKey]) {
          groups[dateKey] = [];
        }
        groups[dateKey]?.push(slot);
      } catch {
        // Skip malformed
      }
    }

    // Sort time slots within each day
    for (const dateKey of Object.keys(groups)) {
      const slotsForDate = groups[dateKey];
      if (slotsForDate) {
        slotsForDate.sort(
          (a, b) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime()
        );
      }
    }

    return groups;
  }, [availability]);

  // Active Selected Doctor details
  const activeDoctor = useMemo(() => {
    return doctors?.find((d) => d.id === selectedDoctorId) ?? null;
  }, [doctors, selectedDoctorId]);

  // Active Selected Slot details
  const activeSlot = useMemo(() => {
    return availability?.find((s) => s.id === selectedSlotId) ?? null;
  }, [availability, selectedSlotId]);

  // Wallet check
  const hasEnoughBalance = useMemo(() => {
    if (!activeDoctor || !wallet) return false;
    return wallet.balance >= activeDoctor.consultation_fee;
  }, [activeDoctor, wallet]);

  // Handlers
  const handleDoctorSelect = (doctorId: string) => {
    setSelectedDoctorId(doctorId);
    setSelectedSlotId(null); // Reset slot
  };

  const handleOpenCheckout = () => {
    if (!selectedSlotId) return;
    setIsCheckoutOpen(true);
  };

  const onProfileCompleteSubmit = async (data: ProfileCompletionFormValues) => {
    try {
      await updateProfile(data);
    } catch (err) {
      if (err instanceof ApiError && err.details) {
        for (const detail of err.details) {
          setError(detail.field as any, {
            type: "server",
            message: detail.issue,
          });
        }
      }
    }
  };

  const handleCancelClick = (id: string) => {
    setCancellingId(id);
    setCancelReason("");
    setCancelError("");
    setIsCancelDialogOpen(true);
  };

  const handleConfirmCancel = async () => {
    if (!cancellingId) return;
    const trimmedReason = cancelReason.trim();
    if (!trimmedReason) {
      setCancelError("Alasan pembatalan wajib diisi");
      return;
    }
    try {
      await cancelAppointment({ id: cancellingId, reason: trimmedReason });
      setIsCancelDialogOpen(false);
      setCancellingId(null);
    } catch {
      // Handled
    }
  };

  const getDoctorNameForAppointment = (docId: string) => {
    const doc = doctors?.find((d) => d.id === docId);
    return doc?.full_name ?? "Medical Specialist";
  };

  const getDoctorSpecialtyForAppointment = (docId: string) => {
    const doc = doctors?.find((d) => d.id === docId);
    return doc?.specialty ?? "Consultant";
  };

  const getAppointmentStatusBadge = (status: string) => {
    switch (status) {
      case "confirmed":
        return <Badge variant="success">CONFIRMED</Badge>;
      case "scheduled":
        return <Badge variant="primary">SCHEDULED</Badge>;
      case "pending":
        return <Badge variant="warning">PENDING</Badge>;
      case "completed":
        return <Badge variant="info">COMPLETED</Badge>;
      case "cancelled":
        return <Badge variant="error">CANCELLED</Badge>;
      default:
        return <Badge variant="neutral">{status.toUpperCase()}</Badge>;
    }
  };

  const handleConfirmBooking = async () => {
    if (!selectedDoctorId || !selectedSlotId) return;
    try {
      await bookAppointment({
        doctor_id: selectedDoctorId,
        availability_id: selectedSlotId,
      });
      setIsCheckoutOpen(false);
      setSelectedSlotId(null);
      // Redirect back to dashboard to see the new appointment
      void navigate({ to: "/patient" });
    } catch {
      // Error is caught and displayed by react-query & toast
    }
  };

  return (
    <div className="flex flex-col gap-8">
      {/* Page Header */}
      <section className="flex flex-col md:flex-row justify-between items-end gap-6 select-none">
        <div className="max-w-2xl">
          <h1 className="font-display text-headline-lg text-primary mb-2 font-bold">Booking & Jadwal Medis</h1>
          <p className="font-body text-body-lg text-on-surface-variant leading-relaxed">
            Kelola konsultasi medis Anda di sini. Cari dokter spesialis, lakukan reservasi slot baru, atau pantau riwayat janji temu aktif Anda.
          </p>
        </div>
      </section>

      <Tabs defaultValue="book" className="w-full">
        <TabsList className="select-none">
          <TabsTrigger value="book">Jadwalkan Konsultasi Baru</TabsTrigger>
          <TabsTrigger value="history">Janji Temu Saya</TabsTrigger>
        </TabsList>

        <TabsContent value="book" className="mt-4">
          {/* Main Grid: Doctor List vs Schedule Picker */}
          <div className="grid grid-cols-12 gap-6 items-start">
            {/* Left Column: Doctor Search & Cards */}
            <div className="col-span-12 lg:col-span-7 flex flex-col gap-6">
              {/* Search bar */}
              <div className="w-full flex gap-4 select-none">
                <div className="relative flex-1 flex items-center">
                  <span className="material-symbols-outlined absolute left-4 text-on-surface-variant text-[20px] pointer-events-none">
                    search
                  </span>
                  <input
                    className="w-full bg-white border border-outline-variant/50 rounded-xl py-3 pl-12 pr-4 text-sm focus:ring-1 focus:ring-primary outline-none transition-all"
                    placeholder="Cari nama dokter atau spesialisasi (mis. Kardiolog)..."
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                  />
                </div>
              </div>

              {/* Doctor Cards */}
              {isDoctorsLoading ? (
                <div className="flex flex-col gap-4 animate-pulse">
                  <div className="h-28 bg-surface-container-low rounded-card"></div>
                  <div className="h-28 bg-surface-container-low rounded-card"></div>
                </div>
              ) : filteredDoctors.length > 0 ? (
                <div className="flex flex-col gap-4">
                  {filteredDoctors.map((doc) => {
                    const isSelected = selectedDoctorId === doc.id;
                    return (
                      <Card
                        key={doc.id}
                        variant={isSelected ? "default" : "interactive"}
                        onClick={() => handleDoctorSelect(doc.id)}
                        className={`p-5 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 transition-all duration-200 border border-outline-variant/10 ${
                          isSelected
                            ? "bg-primary/5 shadow-level-1 ring-1 ring-primary border-primary"
                            : "bg-surface-container-lowest hover:bg-surface-container-lowest"
                        }`}
                      >
                        <div className="flex items-center gap-4">
                          <Avatar name={doc.full_name} size="lg" />
                          <div>
                            <div className="flex items-center gap-2">
                              <h4 className="font-bold text-on-surface text-md leading-tight">{doc.full_name}</h4>
                              {doc.is_credential_verified && (
                                <span
                                  className="material-symbols-outlined text-green-600 text-[18px] select-none"
                                  style={{ fontVariationSettings: "'FILL' 1" }}
                                >
                                  verified
                                </span>
                              )}
                            </div>
                            <p className="text-sm text-on-surface-variant/80 font-semibold mt-1">
                              {doc.specialty ?? "Dokter Umum"}
                            </p>
                            <p className="text-xs text-primary font-bold mt-2">
                              Tarif: {formatCurrency(Number(doc.consultation_fee))}
                            </p>
                          </div>
                        </div>
                        <Button
                          variant={isSelected ? "primary" : "outline"}
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDoctorSelect(doc.id);
                          }}
                          className="rounded-full select-none"
                        >
                          {isSelected ? "Jadwal Dipilih" : "Pilih Dokter"}
                        </Button>
                      </Card>
                    );
                  })}
                </div>
              ) : (
                <EmptyState
                  icon="search_off"
                  title="Dokter Tidak Ditemukan"
                  description="Tidak ada dokter yang cocok dengan kriteria pencarian Anda. Silakan coba kata kunci lain."
                />
              )}
            </div>

            {/* Right Column: Slot Picker & Booking Action */}
            <div className="col-span-12 lg:col-span-5 flex flex-col gap-6">
              {selectedDoctorId ? (
                <Card variant="elevation" className="p-6 border border-outline-variant/10">
                  <div className="flex items-center gap-3 mb-6 select-none">
                    <Avatar name={activeDoctor?.full_name ?? ""} size="md" />
                    <div>
                      <h4 className="font-bold text-on-surface text-sm leading-tight">
                        Jadwal {activeDoctor?.full_name}
                      </h4>
                      <p className="text-xs text-on-surface-variant font-semibold mt-0.5">
                        Pilih slot sesi video-call yang tersedia
                      </p>
                    </div>
                  </div>

                  {isAvailabilityLoading ? (
                    <div className="flex flex-col gap-4 animate-pulse">
                      <div className="h-6 bg-surface-container rounded w-1/3"></div>
                      <div className="grid grid-cols-3 gap-2">
                        <div className="h-10 bg-surface-container rounded"></div>
                        <div className="h-10 bg-surface-container rounded"></div>
                      </div>
                    </div>
                  ) : Object.keys(groupedAvailability).length > 0 ? (
                    <div className="flex flex-col gap-6 max-h-[400px] overflow-y-auto pr-1">
                      {Object.entries(groupedAvailability).map(([dateKey, slots]) => (
                        <div key={dateKey} className="flex flex-col gap-2">
                          <h5 className="text-xs font-bold text-on-surface-variant uppercase tracking-wider select-none">
                            {slots[0] ? formatDateHeader(slots[0].start_time) : ""}
                          </h5>
                          <div className="grid grid-cols-3 gap-2">
                            {slots.map((slot) => {
                              const isSlotSelected = selectedSlotId === slot.id;
                              return (
                                <button
                                  key={slot.id}
                                  type="button"
                                  onClick={() => setSelectedSlotId(slot.id)}
                                  className={`py-2 rounded-lg border text-xs font-semibold select-none transition-all duration-150 ${
                                    isSlotSelected
                                      ? "bg-primary text-white border-primary font-bold shadow-sm"
                                      : "bg-surface-container-low border-outline-variant/30 text-on-surface hover:border-primary"
                                  }`}
                                >
                                  {formatTime(slot.start_time)}
                                </button>
                              );
                            })}
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <EmptyState
                      icon="event_busy"
                      title="Jadwal Praktek Kosong"
                      description="Dokter saat ini tidak memiliki slot ketersediaan janji temu. Silakan pilih dokter lain."
                      className="border-none p-4"
                    />
                  )}

                  <div className="h-px bg-outline-variant/20 w-full my-6"></div>

                  <Button
                    disabled={!selectedSlotId}
                    onClick={handleOpenCheckout}
                    leftIcon="shopping_cart_checkout"
                    className="w-full py-3.5 rounded-xl font-bold"
                  >
                    Lanjutkan Pemesanan
                  </Button>
                </Card>
              ) : (
                <Card variant="elevation" className="p-8 border border-outline-variant/10 text-center select-none">
                  <div className="w-16 h-16 bg-primary/5 rounded-full flex items-center justify-center mx-auto mb-4 text-primary">
                    <span className="material-symbols-outlined text-[36px]">contact_page</span>
                  </div>
                  <h4 className="text-headline-sm font-bold text-on-surface mb-2">Pilih Dokter Terlebih Dahulu</h4>
                  <p className="text-body-sm text-on-surface-variant max-w-xs mx-auto">
                    Silakan pilih dokter spesialis dari daftar di sebelah kiri untuk melihat kalender ketersediaan jadwal mereka.
                  </p>
                </Card>
              )}
            </div>
          </div>
        </TabsContent>

        <TabsContent value="history" className="mt-4">
          <Card variant="elevation" className="border border-outline-variant/10 overflow-hidden">
            {isAppointmentsLoading ? (
              <div className="p-8 flex flex-col gap-4 animate-pulse">
                <div className="h-10 bg-surface-container rounded"></div>
                <div className="h-10 bg-surface-container rounded"></div>
                <div className="h-10 bg-surface-container rounded"></div>
              </div>
            ) : appointments && appointments.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="w-full text-left">
                  <thead className="bg-surface-container-low text-on-surface-variant text-label-sm uppercase tracking-wider select-none">
                    <tr>
                      <th className="px-6 py-4 font-bold">Dokter</th>
                      <th className="px-6 py-4 font-bold">Jadwal Sesi</th>
                      <th className="px-6 py-4 font-bold">Status</th>
                      <th className="px-6 py-4 font-bold text-right">Aksi</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-outline-variant/10">
                    {appointments
                      .sort((a, b) => new Date(b.scheduled_at).getTime() - new Date(a.scheduled_at).getTime())
                      .map((apt) => {
                        const canCancel = apt.status !== "cancelled" && apt.status !== "completed" && new Date(apt.scheduled_at).getTime() > Date.now();
                        return (
                          <tr key={apt.id} className="hover:bg-surface-container-lowest/30 transition-colors">
                            <td className="px-6 py-4 flex items-center gap-3">
                              <Avatar name={getDoctorNameForAppointment(apt.doctor_id)} size="sm" />
                              <div>
                                <p className="font-bold text-on-surface text-sm leading-tight">
                                  {getDoctorNameForAppointment(apt.doctor_id)}
                                </p>
                                <p className="text-xs text-on-surface-variant/80 font-semibold mt-0.5">
                                  {getDoctorSpecialtyForAppointment(apt.doctor_id)}
                                </p>
                              </div>
                            </td>
                            <td className="px-6 py-4 text-body-sm text-on-surface font-semibold select-all">
                              {new Date(apt.scheduled_at).toLocaleString("id-ID", {
                                weekday: "long",
                                day: "numeric",
                                month: "long",
                                year: "numeric",
                                hour: "2-digit",
                                minute: "2-digit"
                              })}
                            </td>
                            <td className="px-6 py-4 select-none">
                              {getAppointmentStatusBadge(apt.status)}
                            </td>
                            <td className="px-6 py-4 text-right select-none">
                              <Button
                                variant="outline"
                                size="sm"
                                disabled={!canCancel || isCancellingPending}
                                onClick={() => handleCancelClick(apt.id)}
                                className="text-error border-error/20 hover:bg-error/5 py-1 px-3 rounded-full text-xs font-bold"
                              >
                                Batalkan
                              </Button>
                            </td>
                          </tr>
                        );
                      })}
                  </tbody>
                </table>
              </div>
            ) : (
              <EmptyState
                icon="calendar_today"
                title="Tidak Ada Jadwal Janji Temu"
                description="Anda belum memiliki riwayat reservasi janji temu medis apa pun di akun Anda."
              />
            )}
          </Card>
        </TabsContent>
      </Tabs>

      {/* Booking Checkout & Confirmation Dialog */}
      <Dialog
        isOpen={isCheckoutOpen}
        onClose={() => setIsCheckoutOpen(false)}
        title="Ringkasan & Pembayaran Janji Temu"
        size="md"
        footer={
          isProfileComplete && hasEnoughBalance ? (
            <div className="flex gap-3 justify-end w-full select-none">
              <Button variant="outline" onClick={() => setIsCheckoutOpen(false)} className="px-6 py-2.5 rounded-full">
                Batal
              </Button>
              <Button
                onClick={handleConfirmBooking}
                isLoading={isBookingPending}
                className="px-6 py-2.5 rounded-full"
              >
                Bayar & Reservasi
              </Button>
            </div>
          ) : undefined
        }
      >
        <div className="flex flex-col gap-6">
          {/* Appointment Details Summary */}
          <div className="p-4 rounded-xl bg-surface-container-low border border-outline-variant/20 flex gap-4 select-none">
            <Avatar name={activeDoctor?.full_name ?? ""} size="md" />
            <div className="flex-1 min-w-0">
              <h5 className="font-bold text-on-surface text-sm truncate">{activeDoctor?.full_name}</h5>
              <p className="text-xs text-on-surface-variant mt-0.5">{activeDoctor?.specialty}</p>
              <div className="mt-3 flex flex-col sm:flex-row sm:items-center gap-3 text-xs font-semibold text-on-surface-variant/80">
                <div className="flex items-center gap-1.5">
                  <span className="material-symbols-outlined text-primary text-[18px]">calendar_today</span>
                  <span>{activeSlot ? formatDateHeader(activeSlot.start_time) : ""}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="material-symbols-outlined text-primary text-[18px]">schedule</span>
                  <span>Pukul {activeSlot ? formatTime(activeSlot.start_time) : ""}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Profile Gate Check */}
          {!isProfileComplete ? (
            <div className="p-5 rounded-xl border border-amber-500/30 bg-amber-500/5 flex flex-col gap-4">
              <div className="flex items-start gap-3 select-none">
                <span className="material-symbols-outlined text-amber-600 text-[24px]">warning</span>
                <div>
                  <h5 className="font-bold text-amber-950 text-sm">Profil Medis Belum Lengkap</h5>
                  <p className="text-xs text-amber-900 mt-1 leading-relaxed">
                    Sesuai dengan regulasi medis TeleMedHub, silakan lengkapi tanggal lahir dan jenis kelamin Anda sebelum melanjutkan pemesanan.
                  </p>
                </div>
              </div>

              <form onSubmit={handleProfileSubmit(onProfileCompleteSubmit)} className="flex flex-col gap-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label htmlFor="date_of_birth" className="block text-xs font-bold text-on-surface-variant mb-1">
                      Tanggal Lahir
                    </label>
                    <input
                      id="date_of_birth"
                      type="date"
                      {...register("date_of_birth")}
                      className="w-full bg-white border border-outline-variant/50 rounded-lg py-2 px-3 text-xs outline-none focus:ring-1 focus:ring-primary"
                    />
                    {profileErrors.date_of_birth && (
                      <p className="text-[10px] text-error mt-1 font-semibold">{profileErrors.date_of_birth.message}</p>
                    )}
                  </div>
                  <div>
                    <label htmlFor="gender" className="block text-xs font-bold text-on-surface-variant mb-1">
                      Jenis Kelamin
                    </label>
                    <select
                      id="gender"
                      {...register("gender")}
                      className="w-full bg-white border border-outline-variant/50 rounded-lg py-2 px-3 text-xs outline-none focus:ring-1 focus:ring-primary"
                    >
                      <option value="">Pilih...</option>
                      <option value="male">Laki-laki</option>
                      <option value="female">Perempuan</option>
                    </select>
                    {profileErrors.gender && (
                      <p className="text-[10px] text-error mt-1 font-semibold">{profileErrors.gender.message}</p>
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label htmlFor="blood_type" className="block text-xs font-bold text-on-surface-variant mb-1">
                      Golongan Darah
                    </label>
                    <select
                      id="blood_type"
                      {...register("blood_type")}
                      className="w-full bg-white border border-outline-variant/50 rounded-lg py-2 px-3 text-xs outline-none focus:ring-1 focus:ring-primary"
                    >
                      <option value="A+">A+</option>
                      <option value="A-">A-</option>
                      <option value="B+">B+</option>
                      <option value="B-">B-</option>
                      <option value="AB+">AB+</option>
                      <option value="AB-">AB-</option>
                      <option value="O+">O+</option>
                      <option value="O-">O-</option>
                    </select>
                    {profileErrors.blood_type && (
                      <p className="text-[10px] text-error mt-1 font-semibold">{profileErrors.blood_type.message}</p>
                    )}
                  </div>
                  <div>
                    <label htmlFor="phone_number" className="block text-xs font-bold text-on-surface-variant mb-1">
                      Nomor Telepon
                    </label>
                    <input
                      id="phone_number"
                      type="tel"
                      placeholder="+628123..."
                      {...register("phone_number")}
                      className="w-full bg-white border border-outline-variant/50 rounded-lg py-2 px-3 text-xs outline-none focus:ring-1 focus:ring-primary"
                    />
                    {profileErrors.phone_number && (
                      <p className="text-[10px] text-error mt-1 font-semibold">{profileErrors.phone_number.message}</p>
                    )}
                  </div>
                </div>

                <Button
                  type="submit"
                  isLoading={isUpdatingProfile}
                  className="w-full py-2.5 rounded-lg text-xs font-bold shadow-sm"
                >
                  Simpan & Daftarkan Profil
                </Button>
              </form>
            </div>
          ) : (
            <>
              {/* Payment Info & Balance Check */}
              <div className="flex flex-col gap-4 border-t border-outline-variant/10 pt-4 select-none">
                <div className="flex justify-between items-center text-sm font-semibold">
                  <span className="text-on-surface-variant">Biaya Konsultasi Dokter</span>
                  <span className="text-on-surface font-bold">
                    {activeDoctor ? formatCurrency(activeDoctor.consultation_fee) : "Rp 0"}
                  </span>
                </div>
                <div className="flex justify-between items-center text-sm font-semibold">
                  <span className="text-on-surface-variant">Saldo Dompet Anda</span>
                  <span className={`font-bold ${hasEnoughBalance ? "text-green-600" : "text-error"}`}>
                    {wallet ? formatCurrency(wallet.balance) : "Rp 0"}
                  </span>
                </div>
              </div>

              {/* Warnings / Action Gates */}
              {!hasEnoughBalance ? (
                <div className="p-4 rounded-xl border border-error/30 bg-error/5 flex flex-col gap-3">
                  <div className="flex items-start gap-2.5 select-none">
                    <span className="material-symbols-outlined text-error text-[20px]">error</span>
                    <p className="text-xs text-on-error-container font-semibold leading-relaxed">
                      Saldo dompet digital Anda tidak mencukupi untuk menyelesaikan reservasi ini. Silakan lakukan pengisian saldo terlebih dahulu.
                    </p>
                  </div>
                  <Button
                    onClick={() => {
                      setIsCheckoutOpen(false);
                      void navigate({ to: "/patient/wallet" });
                    }}
                    className="w-full py-2.5 rounded-lg bg-error hover:bg-error/90 text-white text-xs font-bold shadow-sm select-none"
                  >
                    Top-Up Saldo Sekarang
                  </Button>
                </div>
              ) : (
                <div className="p-4 rounded-xl border border-green-600/30 bg-green-600/5 flex items-start gap-2.5 select-none">
                  <span className="material-symbols-outlined text-green-600 text-[20px]">check_circle</span>
                  <p className="text-xs text-green-800 font-semibold leading-relaxed">
                    Saldo dompet digital Anda mencukupi. Pemotongan saldo akan diproses secara aman dan instan setelah konfirmasi.
                  </p>
                </div>
              )}
            </>
          )}
        </div>
      </Dialog>

      {/* Cancellation Confirmation Dialog */}
      <Dialog
        isOpen={isCancelDialogOpen}
        onClose={() => setIsCancelDialogOpen(false)}
        title="Batalkan Janji Temu"
        size="md"
        footer={
          <div className="flex gap-3 justify-end w-full select-none">
            <Button variant="outline" onClick={() => setIsCancelDialogOpen(false)} className="px-6 py-2.5 rounded-full">
              Kembali
            </Button>
            <Button
              onClick={handleConfirmCancel}
              isLoading={isCancellingPending}
              className="px-6 py-2.5 rounded-full bg-error hover:bg-error/90 text-white border-none"
            >
              Konfirmasi Batal
            </Button>
          </div>
        }
      >
        <div className="flex flex-col gap-4">
          <div className="flex items-start gap-3 p-4 rounded-xl bg-error/5 border border-error/20 select-none">
            <span className="material-symbols-outlined text-error text-[24px]">warning</span>
            <div>
              <h5 className="font-bold text-error text-sm">Apakah Anda yakin?</h5>
              <p className="text-xs text-on-error-container mt-1 leading-relaxed">
                Tindakan ini akan membatalkan sesi konsultasi Anda secara permanen. Pengembalian dana akan didepositkan kembali ke dompet digital Anda secara otomatis.
              </p>
            </div>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="cancel_reason" className="text-xs font-bold text-on-surface-variant select-none">
              Alasan Pembatalan
            </label>
            <textarea
              id="cancel_reason"
              placeholder="Harap masukkan alasan pembatalan janji temu medis Anda di sini..."
              value={cancelReason}
              onChange={(e) => {
                setCancelReason(e.target.value);
                if (e.target.value.trim()) setCancelError("");
              }}
              className="w-full min-h-[100px] bg-white border border-outline-variant/50 rounded-xl p-3 text-sm outline-none focus:ring-1 focus:ring-primary resize-y"
            />
            {cancelError && (
              <p className="text-[10px] text-error font-semibold mt-1 select-none">{cancelError}</p>
            )}
          </div>
        </div>
      </Dialog>
    </div>
  );
}
