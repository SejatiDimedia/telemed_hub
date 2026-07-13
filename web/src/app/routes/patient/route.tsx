import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/patient")({
  beforeLoad: ({ context }) => {
    // Auth guard placeholder — akan diimplementasi setelah auth context terintegrasi dengan router
    // Untuk sekarang, cukup biarkan semua user masuk
    const _ctx = context;
    void _ctx;
  },
  component: PatientLayout,
});

function PatientLayout() {
  return (
    <div className="min-h-screen">
      <header className="border-b p-4">
        <h2 className="text-lg font-semibold">Patient Portal — Layout Placeholder</h2>
      </header>
      <main className="p-4">
        <Outlet />
      </main>
    </div>
  );
}
