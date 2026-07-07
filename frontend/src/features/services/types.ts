export interface ServiceCategory {
  id: string;
  name: string;
  description?: string;
  is_active: boolean;
  sort_order: number;
  created_by: string;
  created_at: string;
  updated_at: string;
  creator_name?: string;
  plan_count: number;
}

export interface ServicePlan {
  id: string;
  category_id: string;
  name: string;
  price: number;
  duration_days: number;
  speed_download?: string;
  speed_upload?: string;
  data_cap?: string;
  province: string;
  connection_type: string;
  installation_fee: number;
  router_included: boolean;
  ip_type: string;
  description?: string;
  cabinet_notes?: string;
  features?: string;
  is_active: boolean;
  sort_order: number;
  created_by: string;
  created_at: string;
  updated_at: string;
  creator_name?: string;
  category_name?: string;
}

export const IRAQ_PROVINCES = [
  'بغداد', 'البصرة', 'نينوى', 'أربيل', 'النجف', 'كربلاء',
  'الأنبار', 'ديالى', 'كركوك', 'بابل', 'واسط', 'ذي قار',
  'ميسان', 'المثنى', 'القادسية', 'صلاح الدين', 'دهوك', 'السليمانية',
];
