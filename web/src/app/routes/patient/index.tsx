import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { usePatientProfile } from "../../../features/patient/hooks/use-patient-profile";
import { useAppointments } from "../../../features/appointment/hooks/use-appointments";
import { useDoctors } from "../../../features/doctor/hooks/use-doctors";
import { Card } from "../../../components/ui/Card";
import { EmptyState } from "../../../components/shared/EmptyState";
import { Button } from "../../../components/ui/Button";
import { Avatar } from "../../../components/ui/Avatar";
import { Badge } from "../../../components/ui/Badge";

export const Route = createFileRoute("/patient/")({
  component: PatientDashboard,
});

function PatientDashboard() {
  const navigate = useNavigate();

  // Fetch patient profile
  const { data: profile, isLoading: isProfileLoading } = usePatientProfile();

  // Fetch appointments
  const { data: appointments, isLoading: isAppointmentsLoading } = useAppointments();

  // Fetch doctor profiles for ID mapping
  const { data: doctors } = useDoctors();

  const patientName = profile?.full_name ?? "Patient";
  const patientBlood = profile?.blood_type ?? "A+";
  const patientCode = profile?.id ? `#TMH-${profile.id.slice(0, 4).toUpperCase()}` : "#TMH-MOCK";

  // Filter scheduled, pending, or confirmed appointments
  const activeAppointments = appointments?.filter(
    (apt) => apt.status === "scheduled" || apt.status === "pending" || apt.status === "confirmed"
  ) ?? [];

  // Sort by date ascending to get the next one
  const nextAppointment = activeAppointments.sort(
    (a, b) => new Date(a.scheduled_at).getTime() - new Date(b.scheduled_at).getTime()
  )[0];

  // Resolve Doctor details
  const nextDoctor = nextAppointment && doctors
    ? doctors.find((d) => d.id === nextAppointment.doctor_id)
    : null;

  const nextDoctorName = nextDoctor?.full_name ?? "Medical Specialist";
  const nextDoctorSpecialty = nextDoctor?.specialty ?? "Consultant Physician";

  const formatAppointmentDate = (dateStr: string) => {
    try {
      const d = new Date(dateStr);
      const day = d.getDate();
      const month = d.toLocaleString("en-US", { month: "short" }).toUpperCase();
      const time = d.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit" });
      return { day, month, time };
    } catch {
      return { day: 24, month: "OCT", time: "02:30 PM" };
    }
  };

  const nextDate = nextAppointment ? formatAppointmentDate(nextAppointment.scheduled_at) : null;

  return (
    <div className="flex flex-col gap-8">
      {/* Hero Welcome Section */}
      <section className="flex flex-col md:flex-row justify-between items-end gap-6">
        <div className="max-w-2xl select-none">
          <h2 className="font-display text-headline-lg text-on-background mb-2">
            Good Morning, <span className="text-primary italic font-bold">{patientName}</span>
          </h2>
          <p className="font-body text-body-lg text-on-surface-variant leading-relaxed">
            It’s a clear day to prioritize your health. TeleMedHub is ready to support your medical needs. How are you feeling today?
          </p>
        </div>
        <div>
          <Button
            variant="outline"
            leftIcon="calendar_today"
            onClick={() => navigate({ to: "/patient/appointments" })}
            className="rounded-card px-6 py-3.5 border-outline-variant hover:shadow-level-1 font-semibold"
          >
            Book New Appointment
          </Button>
        </div>
      </section>

      {/* Bento Grid Layout */}
      <div className="grid grid-cols-12 gap-6 items-start">
        {/* Left Column: Health Snapshot & Appointments */}
        <div className="col-span-12 lg:col-span-8 flex flex-col gap-6">
          {/* Health Snapshot Bento Cards (Mocked per PRD/Database boundaries) */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 select-none">
            <Card variant="interactive" className="p-card-padding">
              <div className="w-10 h-10 bg-error/10 text-error rounded-lg flex items-center justify-center mb-4">
                <span className="material-symbols-outlined font-variation-fill" style={{ fontVariationSettings: "'FILL' 1" }}>
                  favorite
                </span>
              </div>
              <p className="text-label-sm font-semibold text-on-surface-variant/80 mb-1">Heart Rate</p>
              <h3 className="text-headline-md font-bold text-on-surface">
                72 <span className="text-label-sm font-normal text-on-surface-variant">bpm</span>
              </h3>
              <div className="mt-2 text-[10px] text-green-600 flex items-center gap-1 font-semibold">
                <span className="material-symbols-outlined text-xs">trending_down</span>
                2% from normal
              </div>
            </Card>

            <Card variant="interactive" className="p-card-padding">
              <div className="w-10 h-10 bg-primary/10 text-primary rounded-lg flex items-center justify-center mb-4">
                <span className="material-symbols-outlined font-variation-fill" style={{ fontVariationSettings: "'FILL' 1" }}>
                  blood_pressure
                </span>
              </div>
              <p className="text-label-sm font-semibold text-on-surface-variant/80 mb-1">Blood Pressure</p>
              <h3 className="text-headline-md font-bold text-on-surface">
                120/80 <span className="text-label-sm font-normal text-on-surface-variant">mmHg</span>
              </h3>
              <div className="mt-2 text-[10px] text-green-600 flex items-center gap-1 font-semibold">
                <span className="material-symbols-outlined text-xs">check_circle</span>
                Healthy range
              </div>
            </Card>

            <Card variant="interactive" className="p-card-padding">
              <div className="w-10 h-10 bg-secondary/10 text-secondary rounded-lg flex items-center justify-center mb-4">
                <span className="material-symbols-outlined font-variation-fill" style={{ fontVariationSettings: "'FILL' 1" }}>
                  directions_walk
                </span>
              </div>
              <p className="text-label-sm font-semibold text-on-surface-variant/80 mb-1">Daily Steps</p>
              <h3 className="text-headline-md font-bold text-on-surface">
                8,432 <span className="text-label-sm font-normal text-on-surface-variant">steps</span>
              </h3>
              <div className="mt-3.5 w-full bg-surface-container rounded-full h-1.5 overflow-hidden">
                <div className="bg-secondary h-full rounded-full" style={{ width: "84%" }}></div>
              </div>
            </Card>

            <Card variant="interactive" className="p-card-padding">
              <div className="w-10 h-10 bg-tertiary/10 text-tertiary rounded-lg flex items-center justify-center mb-4">
                <span className="material-symbols-outlined font-variation-fill" style={{ fontVariationSettings: "'FILL' 1" }}>
                  bedtime
                </span>
              </div>
              <p className="text-label-sm font-semibold text-on-surface-variant/80 mb-1">Sleep Duration</p>
              <h3 className="text-headline-md font-bold text-on-surface">7h 20m</h3>
              <div className="mt-2 text-[10px] text-primary flex items-center gap-1 font-semibold">
                <span className="material-symbols-outlined text-xs">trending_up</span>
                Better than yesterday
              </div>
            </Card>
          </div>

          {/* Upcoming Appointment Featured Card */}
          {isAppointmentsLoading || isProfileLoading ? (
            <div className="h-48 w-full bg-surface-container-lowest rounded-card animate-pulse border border-outline-variant/20"></div>
          ) : nextAppointment ? (
            <Card variant="elevation" className="overflow-hidden border border-outline-variant/10 group animate-in fade-in duration-500">
              <div className="grid grid-cols-1 md:grid-cols-5">
                <div className="md:col-span-2 relative h-48 md:h-auto overflow-hidden">
                  <img
                    className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-700"
                    alt="Clinical setting"
                    src="https://lh3.googleusercontent.com/aida-public/AB6AXuAtiR0cQv50cZUQ0JmYJNAyNo4anjxyrxtdprJMqur-8vm95A2rnD2GSKaDzDEWHIeFdjXhsvJWlHg7i2rqLD7HO7Kk1qhPCl98U2ivLqjrQDQhTTija9xi4k8i470lKcxc_KKe02Dwo7nrHgugWjHXnT96stqe8I5oBq-_sNcgP0HXlRssH8sviAxERwSYjFEazsxkjaejw2ASbrFR_T2N84fvuZc9_Zho8FLX_V8jnUaTnip4MjlJjwnak6zzNNMaeB4q_VUT74k"
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent flex items-end p-6">
                    <Badge variant="primary" className="uppercase tracking-widest text-[10px] py-1 select-none">
                      Next Appointment
                    </Badge>
                  </div>
                </div>
                <div className="md:col-span-3 p-6 flex flex-col justify-between">
                  <div className="flex justify-between items-start">
                    <div>
                      <h4 className="font-display text-headline-md text-on-surface mb-1 font-bold">
                        {nextDoctorName}
                      </h4>
                      <p className="text-on-surface-variant font-body-md text-sm">
                        {nextDoctorSpecialty}
                      </p>
                    </div>
                    {nextDate && (
                      <div className="bg-surface-container-high px-4 py-2 rounded-xl text-center select-none min-w-[70px]">
                        <p className="text-[10px] font-bold text-on-surface-variant uppercase">{nextDate.month}</p>
                        <p className="text-headline-md font-bold text-primary">{nextDate.day}</p>
                      </div>
                    )}
                  </div>
                  <div className="mt-6 flex flex-wrap gap-4 items-center">
                    <div className="flex items-center gap-2 text-on-surface-variant text-sm font-semibold select-none">
                      <span className="material-symbols-outlined text-primary text-[20px]">schedule</span>
                      <span>{nextDate?.time} (45 min)</span>
                    </div>
                    <div className="flex items-center gap-2 text-on-surface-variant text-sm font-semibold select-none">
                      <span className="material-symbols-outlined text-primary text-[20px]">videocam</span>
                      <span>Telehealth Call</span>
                    </div>
                    <Button
                      onClick={() => alert("Menghubungkan ke layanan Video Call...")}
                      className="ml-auto bg-primary text-white px-6 py-2.5 rounded-full font-semibold shadow-level-1 hover:bg-primary-container transition-all"
                    >
                      Join Call
                    </Button>
                  </div>
                </div>
              </div>
            </Card>
          ) : (
            <EmptyState
              icon="calendar_today"
              title="Belum Ada Janji Temu"
              description="Anda belum memiliki konsultasi medis terdekat. Jadwalkan janji temu dengan dokter spesialis kami."
              actionLabel="Jadwalkan Janji Temu"
              onActionClick={() => navigate({ to: "/patient/appointments" })}
              actionIcon="add_circle"
            />
          )}

          {/* Recent Activity Timeline */}
          <div className="bg-white p-card-padding rounded-card shadow-level-1 border border-outline-variant/10">
            <div className="flex justify-between items-center mb-6">
              <h3 className="font-display text-headline-md text-on-surface font-bold">Recent Medical Activity</h3>
              <Link to="/patient/records" className="text-primary text-label-md hover:underline font-bold">
                View All Records
              </Link>
            </div>
            <div className="relative pl-8 space-y-8 before:content-[''] before:absolute before:left-[11px] before:top-2 before:bottom-2 before:w-[2px] before:bg-surface-container-high select-none">
              <div className="relative">
                <div className="absolute -left-[30px] top-0 w-[24px] h-[24px] bg-primary rounded-full border-4 border-white flex items-center justify-center"></div>
                <div>
                  <div className="flex justify-between items-start mb-1">
                    <h5 className="font-bold text-on-surface text-sm">Lab Results Available</h5>
                    <span className="text-[12px] text-on-surface-variant">2 hours ago</span>
                  </div>
                  <p className="text-body-sm text-on-surface-variant">
                    Your blood chemistry results from Oct 20th are now ready for review.
                  </p>
                  <div className="mt-2.5 flex gap-2">
                    <button
                      onClick={() => alert("Mengunduh laporan PDF...")}
                      className="px-4 py-1.5 bg-surface-container-low text-primary rounded-lg text-sm font-semibold hover:bg-surface-container transition-colors"
                    >
                      Download PDF
                    </button>
                  </div>
                </div>
              </div>
              <div className="relative">
                <div className="absolute -left-[30px] top-0 w-[24px] h-[24px] bg-surface-container-highest rounded-full border-4 border-white"></div>
                <div>
                  <div className="flex justify-between items-start mb-1">
                    <h5 className="font-bold text-on-surface text-sm">Prescription Refilled</h5>
                    <span className="text-[12px] text-on-surface-variant">Yesterday</span>
                  </div>
                  <p className="text-body-sm text-on-surface-variant">
                    Dr. Sarah Jenkins approved your refill for Lisinopril 20mg. Ready for pickup at City Central Pharmacy.
                  </p>
                </div>
              </div>
              <div className="relative">
                <div className="absolute -left-[30px] top-0 w-[24px] h-[24px] bg-surface-container-highest rounded-full border-4 border-white"></div>
                <div>
                  <div className="flex justify-between items-start mb-1">
                    <h5 className="font-bold text-on-surface text-sm">Diet Adjustment Approved</h5>
                    <span className="text-[12px] text-on-surface-variant">Oct 19, 2023</span>
                  </div>
                  <p className="text-body-sm text-on-surface-variant italic">
                    "The adjustments to your diet seem to be working well. Keep up the daily walking routine!"
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Right Column: Quick Actions & Personal Profile */}
        <div className="col-span-12 lg:col-span-4 flex flex-col gap-6">
          {/* Quick Actions Grid */}
          <div className="grid grid-cols-2 gap-4">
            <Card
              variant="interactive"
              onClick={() => alert("Fitur Request Refill belum tersedia.")}
              className="p-6 flex flex-col items-center justify-center gap-3 group text-center"
            >
              <div className="w-12 h-12 bg-primary/5 text-primary rounded-2xl flex items-center justify-center group-hover:bg-primary group-hover:text-white transition-colors">
                <span className="material-symbols-outlined text-3xl">pill</span>
              </div>
              <span className="text-label-md font-bold text-on-surface">Request Refill</span>
            </Card>
            <Card
              variant="interactive"
              onClick={() => navigate({ to: "/patient/appointments" })}
              className="p-6 flex flex-col items-center justify-center gap-3 group text-center"
            >
              <div className="w-12 h-12 bg-primary/5 text-primary rounded-2xl flex items-center justify-center group-hover:bg-primary group-hover:text-white transition-colors">
                <span className="material-symbols-outlined text-3xl">chat</span>
              </div>
              <span className="text-label-md font-bold text-on-surface">Message Doctor</span>
            </Card>
            <Card
              variant="interactive"
              onClick={() => navigate({ to: "/patient/records" })}
              className="p-6 flex flex-col items-center justify-center gap-3 group text-center"
            >
              <div className="w-12 h-12 bg-primary/5 text-primary rounded-2xl flex items-center justify-center group-hover:bg-primary group-hover:text-white transition-colors">
                <span className="material-symbols-outlined text-3xl">folder</span>
              </div>
              <span className="text-label-md font-bold text-on-surface">View Records</span>
            </Card>
            <Card
              variant="interactive"
              onClick={() => navigate({ to: "/patient/wallet" })}
              className="p-6 flex flex-col items-center justify-center gap-3 group text-center"
            >
              <div className="w-12 h-12 bg-primary/5 text-primary rounded-2xl flex items-center justify-center group-hover:bg-primary group-hover:text-white transition-colors">
                <span className="material-symbols-outlined text-3xl">payments</span>
              </div>
              <span className="text-label-md font-bold text-on-surface">Pay Bill</span>
            </Card>
          </div>

          {/* AI Health Coach Widget */}
          <div className="bg-primary rounded-card shadow-level-2 p-6 text-white relative overflow-hidden select-none">
            <div className="absolute -right-10 -bottom-10 w-48 h-48 bg-white/10 rounded-full blur-3xl"></div>
            <div className="absolute -left-10 -top-10 w-32 h-32 bg-white/5 rounded-full blur-2xl"></div>
            <div className="relative z-10">
              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 bg-white/20 backdrop-blur-md rounded-full flex items-center justify-center">
                  <span className="material-symbols-outlined" style={{ fontVariationSettings: "'FILL' 1" }}>
                    smart_toy
                  </span>
                </div>
                <h4 className="font-bold text-lg">AI Health Coach</h4>
              </div>
              <p className="font-body text-body-md text-white/90 leading-relaxed mb-6 italic">
                "I noticed your sleep duration has increased by 15% this week. This is positively impacting your morning heart rate variability!"
              </p>
              <div className="space-y-3">
                <div className="bg-white/10 backdrop-blur-sm p-4 rounded-xl border border-white/20 hover:bg-white/20 transition-all cursor-pointer">
                  <p className="text-sm font-semibold mb-1">Morning Tip</p>
                  <p className="text-xs text-white/70">
                    Stay hydrated! Aim for 2L of water today based on your active steps forecast.
                  </p>
                </div>
                <button
                  onClick={() => alert("Asisten AI Triage sedang diinisialisasi...")}
                  className="w-full py-3 bg-white text-primary rounded-full font-bold text-sm hover:bg-surface transition-colors shadow-lg active:scale-95 duration-100"
                >
                  Ask a Question
                </button>
              </div>
            </div>
          </div>

          {/* Personal Profile Preview */}
          <Card variant="elevation" className="p-card-padding text-center select-none">
            <div className="relative w-24 h-24 mx-auto mb-4">
              <Avatar name={patientName} size="lg" status="online" className="w-full h-full border-4 border-surface-container" />
            </div>
            <h5 className="font-bold text-on-surface text-lg">{patientName}</h5>
            <p className="text-sm text-on-surface-variant/80 mb-6">Patient ID: {patientCode}</p>
            <div className="grid grid-cols-2 gap-4 border-t border-outline-variant/20 pt-6">
              <div>
                <p className="text-[10px] text-on-surface-variant uppercase font-bold mb-1">Blood Type</p>
                <p className="font-bold text-primary text-md">{patientBlood}</p>
              </div>
              <div>
                <p className="text-[10px] text-on-surface-variant uppercase font-bold mb-1">Weight</p>
                <p className="font-bold text-primary text-md">78 kg</p>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}
