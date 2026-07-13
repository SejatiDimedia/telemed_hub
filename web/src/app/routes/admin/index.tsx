import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/admin/")({
  component: AdminDashboard,
});

function AdminDashboard() {
  return (
    <div>
      <h1 className="text-2xl font-bold">Admin Dashboard — Placeholder</h1>
      <p className="mt-2 text-gray-600">User management, verifikasi dokter, dan audit log akan ditampilkan di sini.</p>
    </div>
  );
}
