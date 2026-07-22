export interface Specialty {
  id: string;
  name: string;
  image_icon: string;
  description?: string;
}

export interface DoctorProfile {
  id: string;
  user_id: string;
  email: string;
  full_name: string;
  phone_number?: string | null;
  specialty_id?: string | null;
  specialty?: Specialty | null;
  license_number?: string | null;
  is_credential_verified: boolean;
  consultation_fee: number;
}

export interface Availability {
  id: string;
  doctor_id: string;
  start_time: string;
  end_time: string;
  is_booked: boolean;
}
