export interface User {
  UserID: string;
  Name: string;
  Circle: string;
  Step: number;
  primaryCircleId?: number;
}

export interface ReceivedMessage {
  timestamp: string;
  userID: string;
  text: string;
}

export interface ApiResponse {
  status: string;
}

export interface Circle {
  id: number;
  name: string;
  createdBy: string;
  createdAt: string;
}

export interface CircleMember {
  userId: string;
  name: string;
  joinedAt: string;
}
