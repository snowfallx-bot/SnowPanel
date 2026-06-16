export interface SettingItem {
  key: string;
  value: string;
  value_type: string;
  is_encrypted: boolean;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface ListSettingsResponse {
  items: SettingItem[];
}

export interface CreateSettingRequest {
  key: string;
  value: string;
  value_type: string;
  description?: string;
  is_encrypted: boolean;
}

export interface UpdateSettingRequest {
  value?: string;
  value_type?: string;
  description?: string;
  is_encrypted?: boolean;
}

export type SettingValueType = 'string' | 'number' | 'boolean' | 'json' | 'text';

export interface SettingCategory {
  id: string;
  name: string;
  icon: string;
  keys: string[];
}
