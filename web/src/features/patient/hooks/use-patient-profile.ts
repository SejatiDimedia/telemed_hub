import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { patientApi } from "../api/patient-api";
import type { UpdatePatientRequest } from "../types";
import { useToastStore } from "../../../stores/toast-store";

export const patientKeys = {
  all: ["patients"] as const,
  profile: () => [...patientKeys.all, "profile"] as const,
};

export function usePatientProfile() {
  return useQuery({
    queryKey: patientKeys.profile(),
    queryFn: patientApi.getMe,
    staleTime: 5 * 60 * 1000, // 5 minutes cache
  });
}

export function useUpdatePatientProfile() {
  const queryClient = useQueryClient();
  const addToast = useToastStore((state) => state.addToast);

  return useMutation({
    mutationFn: (data: UpdatePatientRequest) => patientApi.updateMe(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: patientKeys.profile() });
      addToast({
        type: "success",
        title: "Profil Diperbarui",
        message: "Profil rekam medis Anda berhasil disimpan.",
      });
    },
    onError: (error: any) => {
      addToast({
        type: "error",
        title: "Gagal Memperbarui Profil",
        message: error instanceof Error ? error.message : "Terjadi kesalahan pada server.",
      });
    },
  });
}
