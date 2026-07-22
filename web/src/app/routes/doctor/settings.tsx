import { createFileRoute } from "@tanstack/react-router";
import { useDoctorProfileMe, useUpdateDoctorProfile } from "../../../features/doctor/hooks/use-doctors";
import { Card } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as zod from "zod";
import { useEffect } from "react";
import { ApiError } from "../../../lib/api-client";

export const Route = createFileRoute("/doctor/settings")({
  component: DoctorSettingsPage,
});

const doctorSettingsSchema = zod.object({
  phone_number: zod.string()
    .min(1, "Nomor telepon wajib diisi")
    .regex(/^\+?[1-9]\d{1,14}$/, "Format nomor telepon tidak valid (E.164, misal: +6281234567890)"),
  specialty_id: zod.string().min(1, "Spesialisasi wajib diisi"),
  license_number: zod.string().min(1, "Nomor izin praktek wajib diisi"),
  consultation_fee: zod.number()
    .min(1, "Tarif konsultasi harus lebih dari 0"),
});

type DoctorSettingsFormValues = zod.infer<typeof doctorSettingsSchema>;

function DoctorSettingsPage() {
  const { data: profile, isLoading } = useDoctorProfileMe();
  const { mutateAsync: updateProfile, isPending: isUpdating } = useUpdateDoctorProfile();

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    formState: { errors },
  } = useForm<DoctorSettingsFormValues>({
    resolver: zodResolver(doctorSettingsSchema),
    defaultValues: {
      phone_number: "",
      specialty_id: "",
      license_number: "",
      consultation_fee: 0,
    },
  });

  // Pre-fill form values when profile is loaded
  useEffect(() => {
    if (profile) {
      setValue("phone_number", profile.phone_number ?? "");
      setValue("specialty_id", profile.specialty_id ?? "");
      setValue("license_number", profile.license_number ?? "");
      setValue("consultation_fee", profile.consultation_fee ?? 0);
    }
  }, [profile, setValue]);

  const onSubmit = async (data: DoctorSettingsFormValues) => {
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

  if (isLoading) {
    return <div className="p-8 text-center text-on-surface-variant animate-pulse font-semibold">Memuat pengaturan profil...</div>;
  }

  return (
    <div className="flex flex-col gap-8 max-w-2xl mx-auto">
      {/* Page Header */}
      <section className="flex flex-col gap-2 select-none">
        <h1 className="font-display text-headline-lg text-primary font-bold">Pengaturan Akun & Praktek</h1>
        <p className="font-body text-body-lg text-on-surface-variant leading-relaxed">
          Kelola informasi profil profesional, nomor kontak, izin praktek, dan tarif konsultasi medis Anda di sini.
        </p>
      </section>

      {/* Main Settings Card */}
      <Card variant="elevation" className="p-6 border border-outline-variant/10">
        <h3 className="font-display text-headline-sm font-bold text-on-surface mb-6 select-none flex items-center gap-2">
          <span className="material-symbols-outlined text-primary">clinical_notes</span>
          Informasi Profesional & Kontak
        </h3>

        <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-6">
          {/* Read-only fields */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 select-none">
            <div>
              <label className="block text-xs font-bold text-on-surface-variant mb-1">
                Nama Lengkap Dokter
              </label>
              <input
                type="text"
                disabled
                value={profile?.full_name ?? ""}
                className="w-full bg-surface-container-low border border-outline-variant/30 text-on-surface-variant rounded-lg py-2.5 px-3 text-sm outline-none cursor-not-allowed font-medium"
              />
              <p className="text-[10px] text-on-surface-variant/70 mt-1">Nama lengkap hanya dapat diubah oleh Admin.</p>
            </div>
            <div>
              <label className="block text-xs font-bold text-on-surface-variant mb-1">
                Status Kredensial Medis
              </label>
              <div className="flex items-center gap-2 mt-2 select-none">
                {profile?.is_credential_verified ? (
                  <span className="inline-flex items-center gap-1 text-xs text-green-600 font-bold bg-green-600/10 px-3 py-1 rounded-full border border-green-600/20">
                    <span className="material-symbols-outlined text-[16px]">verified</span>
                    Terverifikasi
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 text-xs text-amber-600 font-bold bg-amber-600/10 px-3 py-1 rounded-full border border-amber-600/20">
                    <span className="material-symbols-outlined text-[16px]">warning</span>
                    Belum Terverifikasi
                  </span>
                )}
              </div>
            </div>
          </div>

          <div className="h-px bg-outline-variant/20 w-full"></div>

          {/* Editable Fields */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label htmlFor="settings_phone" className="block text-xs font-bold text-on-surface-variant mb-1 select-none">
                Nomor Telepon
              </label>
              <input
                id="settings_phone"
                type="tel"
                placeholder="misal: +6281234567890"
                {...register("phone_number")}
                className="w-full bg-white border border-outline-variant/50 rounded-lg py-2.5 px-3 text-sm outline-none focus:ring-1 focus:ring-primary transition-all"
              />
              {errors.phone_number && (
                <p className="text-xs text-error font-semibold mt-1">{errors.phone_number.message}</p>
              )}
            </div>
            <div>
              <label htmlFor="settings_fee" className="block text-xs font-bold text-on-surface-variant mb-1 select-none">
                Tarif Konsultasi (Rupiah)
              </label>
              <input
                id="settings_fee"
                type="number"
                placeholder="misal: 150000"
                {...register("consultation_fee", { valueAsNumber: true })}
                className="w-full bg-white border border-outline-variant/50 rounded-lg py-2.5 px-3 text-sm outline-none focus:ring-1 focus:ring-primary transition-all"
              />
              {errors.consultation_fee && (
                <p className="text-xs text-error font-semibold mt-1">{errors.consultation_fee.message}</p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label htmlFor="settings_specialty" className="block text-xs font-bold text-on-surface-variant mb-1 select-none">
                Spesialisasi Medis
              </label>
              <select
                id="settings_specialty"
                {...register("specialty_id")}
                className="w-full bg-white border border-outline-variant/50 rounded-lg py-2.5 px-3 text-sm outline-none focus:ring-1 focus:ring-primary transition-all"
              >
                <option value="">Pilih Spesialisasi</option>
                {/* Fallback hardcoded for now, waiting for backend GET /specialties */}
                <option value="f47ac10b-58cc-4372-a567-0e02b2c3d479">Cardiology</option>
                <option value="f47ac10b-58cc-4372-a567-0e02b2c3d480">Neurology</option>
                <option value="f47ac10b-58cc-4372-a567-0e02b2c3d481">Pediatrics</option>
                <option value="f47ac10b-58cc-4372-a567-0e02b2c3d482">General Practitioner</option>
                <option value="f47ac10b-58cc-4372-a567-0e02b2c3d483">Dermatology</option>
              </select>
              {errors.specialty_id && (
                <p className="text-xs text-error font-semibold mt-1">{errors.specialty_id.message}</p>
              )}
            </div>
            <div>
              <label htmlFor="settings_license" className="block text-xs font-bold text-on-surface-variant mb-1 select-none">
                Nomor Izin Praktek
              </label>
              <input
                id="settings_license"
                type="text"
                placeholder="misal: STR/2026/8932"
                {...register("license_number")}
                className="w-full bg-white border border-outline-variant/50 rounded-lg py-2.5 px-3 text-sm outline-none focus:ring-1 focus:ring-primary transition-all"
              />
              {errors.license_number && (
                <p className="text-xs text-error font-semibold mt-1">{errors.license_number.message}</p>
              )}
            </div>
          </div>

          <Button
            type="submit"
            isLoading={isUpdating}
            leftIcon="check"
            className="w-full py-3.5 rounded-xl font-bold mt-4 select-none"
          >
            Simpan Perubahan
          </Button>
        </form>
      </Card>
    </div>
  );
}
