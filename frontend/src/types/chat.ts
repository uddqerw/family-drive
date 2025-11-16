export interface ChatMessage {
  id: number;
  user_id: number;
  username: string;
  content: string;
  type: 'text' | 'image' | 'file';
  room: string;
  created_at: string;
}

export interface ChatUser {
  id: number;
  username: string;
  online: boolean;
  lastSeen?: string;
}

export interface ChatState {
  messages: ChatMessage[];
  users: ChatUser[];
  currentRoom: string;
  isConnected: boolean;
}