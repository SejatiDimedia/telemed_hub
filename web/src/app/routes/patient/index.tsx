import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/patient/")({
  component: PatientDashboard,
});

function PatientDashboard() {
  return (
    <div>
      <h1 className="text-2xl font-bold">Patient Dashboard — Placeholder</h1>
      <p className="mt-2 text-gray-600">Booking, resep, wallet, riwayat medis akan ditampilkan di sini.</p>
    </div>
  );
}
