// src/hooks/useFileSearch.ts
import { useState, useMemo } from 'react';
import { FileItem, SearchFilters } from '../types/file';

const useFileSearch = (files: FileItem[]) => {
  const [filters, setFilters] = useState<SearchFilters>({
    keyword: '',
    fileType: 'all',
    sortBy: 'name',
    sortOrder: 'asc'
  });

  const filteredFiles = useMemo(() => {
    let result = [...files];

    // 关键词搜索
    if (filters.keyword) {
      result = result.filter(file => 
        file.name.toLowerCase().includes(filters.keyword.toLowerCase())
      );
    }

    // 文件类型过滤
    if (filters.fileType !== 'all') {
      result = result.filter(file => file.category === filters.fileType);
    }

    // 排序
    result.sort((a, b) => {
      let comparison = 0;
      
      switch (filters.sortBy) {
        case 'name':
          comparison = a.name.localeCompare(b.name);
          break;
        case 'size':
          comparison = a.size - b.size;
          break;
        case 'date':
          comparison = new Date(a.uploadTime).getTime() - new Date(b.uploadTime).getTime();
          break;
        case 'type':
          comparison = a.category.localeCompare(b.category);
          break;
        default:
          comparison = 0;
      }

      return filters.sortOrder === 'asc' ? comparison : -comparison;
    });

    return result;
  }, [files, filters]);

  return {
    filteredFiles,
    filters,
    setFilters
  };
};

export default useFileSearch;