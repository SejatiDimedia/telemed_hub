export type AppointmentStatus = "pending" | "scheduled" | "confirmed" | "completed" | "cancelled";

export interface Appointment {
  id: string;
  patient_id: string;
  doctor_id: string;
  availability_id: string;
  status: AppointmentStatus;
  scheduled_at: string;
  cancelled_at: string | null;
  cancel_reason: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateAppointmentRequest {
  doctor_id: string;
  availability_id: string;
}

export interface RescheduleAppointmentRequest {
  availability_id: string;
}

export interface CancelAppointmentRequest {
  cancel_reason: string;
}
