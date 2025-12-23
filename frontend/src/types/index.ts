export interface User {
  UserID: string;
  Name: string;
  Circle: string;
  Step: number;
}

export interface ReceivedMessage {
  timestamp: string;
  userID: string;
  text: string;
}

export interface ApiResponse {
  status: string;
}
