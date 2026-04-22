export interface UserProfile {
  id: number;
  username: string;
  email: string;
  status: number;
  roles: string[];
  permissions: string[];
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
