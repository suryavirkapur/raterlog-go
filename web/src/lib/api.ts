export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:18080";

export type User = {
  id: string;
  name: string;
  email: string;
};

export type Company = {
  id: string;
  name: string;
  billing: boolean;
};

export type Channel = {
  id: string;
  company_id: string;
  name: string;
  icon: string;
};

export type ApiToken = {
  id: number;
  name: string;
  token: string;
  company_id: string;
};

export type Member = {
  user_id: string;
  name: string;
  email: string;
  role: string;
};

export type Invite = {
  id: string;
  email: string;
  company_id: string;
  company_name?: string;
  token: string;
  status: string;
  created_at: string;
  expires_at: string;
};

export type CompanyDetail = {
  company: Company;
  channels: Channel[];
  tokens: ApiToken[];
  members: Member[];
  invites: Invite[];
};

export type LogEvent = {
  channel_id: string;
  timestamp: string;
  event_name: string;
  event_payload: string;
  metadata: string | null;
};

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {})
    }
  });
  if (!response.ok) {
    let message = response.statusText;
    try {
      const body = (await response.json()) as { error?: string };
      message = body.error ?? message;
    } catch {}
    throw new Error(message);
  }
  return (await response.json()) as T;
}
