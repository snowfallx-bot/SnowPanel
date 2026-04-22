export interface UserProfile {
  id: number;
  username: string;
  email: string;
  status: number;
  roles: string[];
  permissions: string[];
  must_change_password: boolean;
}

export interface LoginPayload {
  username: string;
  password: string;
}

export interface LoginResult {
  access_token: string;
  token_type: string;
  expires_in: number;
  user: UserProfile;
}

export interface ChangePasswordPayload {
  current_password: string;
  new_password: string;
}
