// LIFFバックエンドAPIクライアント

const API_BASE = import.meta.env.VITE_API_BASE || '';

interface ApiOptions {
  method?: string;
  body?: any;
  accessToken: string;
}

async function apiCall<T>(endpoint: string, options: ApiOptions): Promise<T> {
  const { method = 'GET', body, accessToken } = options;

  const headers: HeadersInit = {
    'Authorization': `Bearer ${accessToken}`,
    'Content-Type': 'application/json',
    'ngrok-skip-browser-warning': 'true',
  };

  const response = await fetch(`${API_BASE}${endpoint}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`API Error: ${response.status} - ${errorText}`);
  }

  const text = await response.text();
  if (!text) {
    return {} as T;
  }

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error('Invalid JSON response');
  }
}

// ========== ユーザー関連 ==========

export interface User {
  userId: string;
  name: string;
  displayName: string;
  circle: string;
  registered: boolean;
  step: number;
}

export async function getMyInfo(accessToken: string): Promise<User> {
  return apiCall('/api/liff/me', { accessToken });
}

export async function registerUser(accessToken: string, circle: string) {
  return apiCall('/api/liff/register', {
    method: 'POST',
    body: { circle },
    accessToken,
  });
}

export async function getCircleMembers(accessToken: string) {
  return apiCall<{ status: string; members: Array<{ userId: string; name: string; circle: string }> }>(
    '/api/liff/circle/members',
    { accessToken }
  );
}

// ========== イベント関連 ==========

export interface Event {
  id: number;
  name: string;
  totalAmount: number;
  splitAmount: number;
  status: string;
  createdAt: string;
}

export async function getMyEvents(accessToken: string): Promise<{ status: string; events: Event[] }> {
  return apiCall('/api/liff/events', { accessToken });
}

export interface CreateEventRequest {
  eventName: string;
  totalAmount: number;
  participantIds: string[];
}

export async function createEvent(accessToken: string, data: CreateEventRequest) {
  return apiCall('/api/liff/events', {
    method: 'POST',
    body: data,
    accessToken,
  });
}

// ========== 承認関連 ==========

export interface Approval {
  id: number;
  eventId: number;
  participantId: string;
  participantName: string;
  eventName: string;
  amount: number;
  reportedAt: string;
}

export async function getPendingApprovals(accessToken: string): Promise<{ status: string; approvals: Approval[] }> {
  return apiCall('/api/liff/approvals', { accessToken });
}

export async function approvePayments(accessToken: string, participantIds: number[]) {
  return apiCall('/api/liff/approvals', {
    method: 'POST',
    body: { participantIds },
    accessToken,
  });
}
