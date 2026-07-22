import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { Sidebar } from "../../../components/shared/Sidebar";
import { Avatar } from "../../../components/ui/Avatar";
import { useDoctorProfileMe } from "../../../features/doctor/hooks/use-doctors";
import { useAuth } from "../../../context/auth-context";

export const Route = createFileRoute("/doctor")({
  beforeLoad: ({ context }) => {
    if (!context.auth.isAuthenticated) {
      throw redirect({ to: "/login" });
    }
    if (context.auth.user?.role !== "doctor") {
      throw redirect({ to: "/login" });
    }
  },
  component: DoctorLayout,
});

const doctorNavItems = [
  { label: "Dashboard", icon: "dashboard", to: "/doctor" },
  { label: "Manage Schedule", icon: "calendar_today", to: "/doctor/schedule" },
  { label: "Settings", icon: "settings", to: "/doctor/settings" },
];

function DoctorLayout() {
  const { user } = useAuth();
  const { data: profile } = useDoctorProfileMe();

  const displayName = profile?.full_name ?? user?.email.split("@")[0] ?? "Doctor";
  const specialty = profile?.specialty?.name ?? "Medical Specialist";

  return (
    <div className="bg-background text-on-background font-body min-h-screen">
      {/* Side Navigation Bar */}
      <Sidebar items={doctorNavItems} />

      {/* Top App Bar */}
      <header className="h-20 fixed top-0 right-0 left-[280px] z-30 bg-background/80 backdrop-blur-md flex items-center justify-between px-gutter border-b border-outline-variant/30 select-none">
        {/* Title / Search */}
        <div className="flex items-center gap-4 w-1/2">
          <div className="relative w-full max-w-md flex items-center">
            <span className="material-symbols-outlined absolute left-3 text-on-surface-variant text-[20px] pointer-events-none">
              search
            </span>
            <input
              className="w-full bg-surface-container-low border-none rounded-full py-2 pl-10 pr-4 text-sm focus:ring-1 focus:ring-primary outline-none transition-all duration-200"
              placeholder="Search patients, consultations, or records..."
              type="text"
            />
          </div>
        </div>

        {/* Right Action Profile */}
        <div className="flex items-center gap-6">
          <div className="flex gap-2">
            <button className="p-2 rounded-full hover:bg-surface-container transition-colors text-on-surface-variant">
              <span className="material-symbols-outlined">help</span>
            </button>
            <button className="p-2 rounded-full hover:bg-surface-container transition-colors text-on-surface-variant">
              <span className="material-symbols-outlined">dark_mode</span>
            </button>
          </div>
          <div className="h-8 w-px bg-outline-variant/30"></div>
          <div className="flex items-center gap-3 cursor-pointer group">
            <div className="text-right">
              <p className="text-label-md font-bold text-on-surface leading-tight">
                {displayName}
              </p>
              <p className="text-[11px] font-semibold text-primary">{specialty}</p>
            </div>
            <Avatar name={displayName} size="md" />
          </div>
        </div>
      </header>

      {/* Main Content Pane */}
      <main className="pl-[280px] pt-20 min-h-screen flex flex-col justify-between">
        <div className="flex-1 p-gutter">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
