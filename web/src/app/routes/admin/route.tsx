import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/admin")({
  beforeLoad: ({ context }) => {
    const _ctx = context;
    void _ctx;
  },
  component: AdminLayout,
});

function AdminLayout() {
  return (
    <div className="min-h-screen">
      <header className="border-b p-4">
        <h2 className="text-lg font-semibold">Admin Panel — Layout Placeholder</h2>
      </header>
      <main className="p-4">
        <Outlet />
      </main>
    </div>
  );
}
