// src/components/SearchBar.tsx
import React from 'react';
import { Input, Select, Space } from 'antd';
import { SearchOutlined } from '@ant-design/icons';

const { Search } = Input;
const { Option } = Select;

interface SearchBarProps {
  onSearch: (keyword: string) => void;
  onFilterChange: (filterType: string, value: string) => void;
  filters: {
    keyword: string;
    fileType: string;
    sortBy: string;
    sortOrder: string;
  };
}

const SearchBar: React.FC<SearchBarProps> = ({ 
  onSearch, 
  onFilterChange, 
  filters 
}) => {
  return (
    <div style={{ padding: '16px', background: '#fafafa', borderRadius: '8px', marginBottom: '16px' }}>
      <Space wrap size="middle">
        {/* 搜索框 */}
        <Search
          placeholder="搜索文件名..."
          value={filters.keyword}
          onChange={(e) => onFilterChange('keyword', e.target.value)}
          style={{ width: 200 }}
          allowClear
          enterButton={<SearchOutlined />}
        />

        {/* 文件类型过滤 */}
        <Select
          value={filters.fileType}
          onChange={(value) => onFilterChange('fileType', value)}
          style={{ width: 120 }}
        >
          <Option value="all">全部类型</Option>
          <Option value="image">图片</Option>
          <Option value="document">文档</Option>
          <Option value="video">视频</Option>
          <Option value="archive">压缩包</Option>
          <Option value="other">其他</Option>
        </Select>

        {/* 排序方式 */}
        <Select
          value={filters.sortBy}
          onChange={(value) => onFilterChange('sortBy', value)}
          style={{ width: 120 }}
        >
          <Option value="name">按名称</Option>
          <Option value="size">按大小</Option>
          <Option value="date">按时间</Option>
          <Option value="type">按类型</Option>
        </Select>

        {/* 排序顺序 */}
        <Select
          value={filters.sortOrder}
          onChange={(value) => onFilterChange('sortOrder', value)}
          style={{ width: 100 }}
        >
          <Option value="asc">升序</Option>
          <Option value="desc">降序</Option>
        </Select>
      </Space>
    </div>
  );
};

export default SearchBar;