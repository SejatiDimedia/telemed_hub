import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/doctor/")({
  component: DoctorDashboard,
});

function DoctorDashboard() {
  return (
    <div>
      <h1 className="text-2xl font-bold">Doctor Dashboard — Placeholder</h1>
      <p className="mt-2 text-gray-600">Jadwal, konsultasi, dan catatan pasien akan ditampilkan di sini.</p>
    </div>
  );
}
