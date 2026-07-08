export interface ServiceCategory {
  id: string;
  department_id: string;
  name: string;
  description?: string;
  is_active: boolean;
  sort_order: number;
  created_by: string;
  created_at: string;
  updated_at: string;
  creator_name?: string;
  department_name?: string;
  plan_count: number;
  is_shared?: boolean;
}

export interface ServiceCategoryShare {
  id: string;
  category_id: string;
  department_id: string;
  granted_by: string;
  created_at: string;
  department_name?: string;
}
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
export interface Province {
  id: string;
  name: string;
  sort_order: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}
