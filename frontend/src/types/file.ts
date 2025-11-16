// src/types/file.ts
export interface FileItem {
  id: number;
  name: string;
  size: number;
  type: string;
  uploadTime: string;
  category: 'image' | 'document' | 'video' | 'archive' | 'other';
}

export interface SearchFilters {
  keyword: string;
  fileType: string;
  sortBy: 'name' | 'size' | 'date' | 'type';
  sortOrder: 'asc' | 'desc';
}