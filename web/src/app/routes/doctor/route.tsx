import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/doctor")({
  beforeLoad: ({ context }) => {
    const _ctx = context;
    void _ctx;
  },
  component: DoctorLayout,
});

function DoctorLayout() {
  return (
    <div className="min-h-screen">
      <header className="border-b p-4">
        <h2 className="text-lg font-semibold">Doctor Portal — Layout Placeholder</h2>
      </header>
      <main className="p-4">
        <Outlet />
      </main>
    </div>
  );
}
