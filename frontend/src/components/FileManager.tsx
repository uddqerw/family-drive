import React, { useState, useEffect, useMemo } from 'react';
import {
  Button, Upload, message, Card,
  Row, Col, Tag, Progress, Alert,
  Input, Select, Space
} from 'antd';
import {
  UploadOutlined, DownloadOutlined, DeleteOutlined,
  FileOutlined, FileImageOutlined, FilePdfOutlined,
  FileWordOutlined, FileExcelOutlined, FileZipOutlined,
  VideoCameraOutlined, SearchOutlined
} from '@ant-design/icons';
// import { fileAPI } from '../services/api';
import './FileManager.css';

const { Search } = Input;
const { Option } = Select;

// 文件类型图标映射
const fileIcons = {
  'pdf': <FilePdfOutlined style={{ color: '#ff4d4f' }} />,
  'jpg': <FileImageOutlined style={{ color: '#52c41a' }} />,
  'jpeg': <FileImageOutlined style={{ color: '#52c41a' }} />,
  'png': <FileImageOutlined style={{ color: '#52c41a' }} />,
  'gif': <FileImageOutlined style={{ color: '#52c41a' }} />,
  'doc': <FileWordOutlined style={{ color: '#1890ff' }} />,
  'docx': <FileWordOutlined style={{ color: '#1890ff' }} />,
  'xls': <FileExcelOutlined style={{ color: '#52c41a' }} />,
  'xlsx': <FileExcelOutlined style={{ color: '#52c41a' }} />,
  'zip': <FileZipOutlined style={{ color: '#faad14' }} />,
  'rar': <FileZipOutlined style={{ color: '#faad14' }} />,
  'mp4': <VideoCameraOutlined style={{ color: '#722ed1' }} />,
  'avi': <VideoCameraOutlined style={{ color: '#722ed1' }} />,
  'mov': <VideoCameraOutlined style={{ color: '#722ed1' }} />,
  'default': <FileOutlined style={{ color: '#666' }} />
};

interface FileManagerProps {
  onLogout?: () => void;
}

interface FileItem {
  id: number;
  name: string;
  size: number;
  type: string;
  uploadTime: string;
  category: 'image' | 'document' | 'video' | 'archive' | 'other';
}

interface SearchFilters {
  keyword: string;
  fileType: string;
  sortBy: 'name' | 'size' | 'date' | 'type';
  sortOrder: 'asc' | 'desc';
}

const FileManager: React.FC<FileManagerProps> = () => {
  const [files, setFiles] = useState<FileItem[]>([]);
  const [uploading, setUploading] = useState(false);
  const [downloading, setDownloading] = useState<string | null>(null);
  const [downloadStatus, setDownloadStatus] = useState<{show: boolean, type: 'success' | 'error' | 'loading', filename: string} | null>(null);
  const [filters, setFilters] = useState<SearchFilters>({
    keyword: '',
    fileType: 'all',
    sortBy: 'name',
    sortOrder: 'asc'
  });

  // 获取文件分类
  const getFileCategory = (filename: string): FileItem['category'] => {
    const ext = filename.split('.').pop()?.toLowerCase() || '';
    const imageExt = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp'];
    const documentExt = ['pdf', 'doc', 'docx', 'txt', 'ppt', 'pptx'];
    const videoExt = ['mp4', 'avi', 'mov', 'wmv', 'flv', 'mkv'];
    const archiveExt = ['zip', 'rar', '7z', 'tar', 'gz'];

    if (imageExt.includes(ext)) return 'image';
    if (documentExt.includes(ext)) return 'document';
    if (videoExt.includes(ext)) return 'video';
    if (archiveExt.includes(ext)) return 'archive';
    return 'other';
  };

  // 获取文件图标
  const getFileIcon = (filename: string) => {
    const ext = filename.split('.').pop()?.toLowerCase();
    return fileIcons[ext as keyof typeof fileIcons] || fileIcons.default;
  };

  // 格式化文件大小
  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  // 格式化日期
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  // 加载文件列表
  const loadFiles = async () => {
    try {
      console.log('🔄 开始加载文件列表...');
      const response = await fetch('http://localhost:8000/api/files/list');
      
      if (response.ok) {
        const result = await response.json();
        console.log('📁 后端返回数据:', result);
        
        if (result.success && result.data && Array.isArray(result.data)) {
          const filesWithCategory = result.data.map((file: any) => ({
            id: file.id || Date.now(),
            name: file.name || '未知文件',
            size: file.size || 0,
            type: file.type || 'file',
            uploadTime: file.uploadTime || new Date().toISOString(),
            category: getFileCategory(file.name)
          }));
          
          setFiles(filesWithCategory);
          console.log('✅ 加载成功，文件数:', filesWithCategory.length);
        }
      } else {
        console.log('❌ HTTP请求失败');
      }
    } catch (error) {
      console.error('🚨 加载文件列表失败:', error);
      message.error('加载文件列表失败');
    }
  };

  // 过滤和排序文件
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

  // 文件上传
  const handleUpload = async (file: File) => {
    setUploading(true);
    const formData = new FormData();
    formData.append('file', file);

    try {
      console.log('📤 上传文件:', file.name, '大小:', file.size);
      const response = await fetch('http://localhost:8000/api/files/upload', {
        method: 'POST',
        body: formData,
      });

      if (response.ok) {
        const result = await response.json();
        console.log('✅ 上传成功:', result);
        message.success(`文件 "${file.name}" 上传成功`);
        await loadFiles(); // 重新加载文件列表
      } else {
        throw new Error('上传失败');
      }
    } catch (error) {
      console.error('❌ 上传失败:', error);
      message.error('文件上传失败');
    } finally {
      setUploading(false);
    }
    return false;
  };

  // 文件下载
  const handleDownload = async (filename: string) => {
    console.log('🚀 开始下载:', filename);
    setDownloading(filename);

    setDownloadStatus({
      show: true,
      type: 'loading',
      filename: filename
    });

    try {
      // 直接使用 fetch 下载
      const response = await fetch(`http://localhost:8000/api/files/download/${filename}`);
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', filename);
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);

      setDownloadStatus({
        show: true,
        type: 'success',
        filename: filename
      });

      console.log('✅ 下载完成:', filename);

      setTimeout(() => {
        setDownloadStatus(null);
      }, 3000);

    } catch (error: any) {
      console.error('❌ 下载失败:', error);

      setDownloadStatus({
        show: true,
        type: 'error',
        filename: filename
      });

      message.error('下载失败，请重试');

      setTimeout(() => {
        setDownloadStatus(null);
      }, 5000);
    } finally {
      setDownloading(null);
    }
  };

  // 文件删除
  const handleDelete = async (filename: string) => {
    if (!window.confirm(`确定要删除文件 "${filename}" 吗？此操作不可撤销。`)) {
      return;
    }

    try {
      const response = await fetch(`http://localhost:8000/api/files/delete/${filename}`, {
        method: 'DELETE',
      });

      if (response.ok) {
        message.success(`文件 "${filename}" 删除成功`);
        await loadFiles(); // 重新加载文件列表
      } else {
        throw new Error('删除失败');
      }
    } catch (error: any) {
      console.error('删除失败:', error);
      message.error('文件删除失败');
    }
  };

  // 处理过滤条件变化
  const handleFilterChange = (filterType: string, value: string) => {
    setFilters(prev => ({
      ...prev,
      [filterType]: value
    }));
  };

  // 获取文件类型统计
  const getFileStats = () => {
    const stats = {
      total: files.length,
      images: files.filter(f => f.category === 'image').length,
      documents: files.filter(f => f.category === 'document').length,
      videos: files.filter(f => f.category === 'video').length,
      archives: files.filter(f => f.category === 'archive').length,
      others: files.filter(f => f.category === 'other').length
    };
    return stats;
  };

  useEffect(() => {
    loadFiles();
  }, []);

  const fileStats = getFileStats();

  return (
    <div className="enhanced-file-manager">
      <Card
        title={
          <div className="card-header">
            <span>🏠 家庭网盘</span>
            <Space>
              <Tag color="blue">
                {filteredFiles.length} / {files.length} 个文件
              </Tag>
              {fileStats.images > 0 && <Tag color="green">📸 {fileStats.images}</Tag>}
              {fileStats.documents > 0 && <Tag color="blue">📄 {fileStats.documents}</Tag>}
              {fileStats.videos > 0 && <Tag color="purple">🎥 {fileStats.videos}</Tag>}
            </Space>
          </div>
        }
        className="file-manager-card"
        extra={
          <Button 
            icon={<SearchOutlined />} 
            onClick={loadFiles}
            type="primary"
          >
            刷新列表
          </Button>
        }
      >
        {/* 搜索和筛选工具栏 */}
        <div className="search-toolbar">
          <Space wrap size="middle" style={{ width: '100%' }}>
            {/* 搜索框 */}
            <Search
              placeholder="搜索文件名..."
              value={filters.keyword}
              onChange={(e) => handleFilterChange('keyword', e.target.value)}
              style={{ width: 200 }}
              allowClear
              enterButton={<SearchOutlined />}
            />

            {/* 文件类型过滤 */}
            <Select
              value={filters.fileType}
              onChange={(value) => handleFilterChange('fileType', value)}
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
              onChange={(value) => handleFilterChange('sortBy', value)}
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
              onChange={(value) => handleFilterChange('sortOrder', value)}
              style={{ width: 100 }}
            >
              <Option value="asc">升序 ↑</Option>
              <Option value="desc">降序 ↓</Option>
            </Select>
          </Space>
        </div>

        {/* 下载状态提示 */}
        {downloadStatus?.show && (
          <div className="download-alert">
            {downloadStatus.type === 'loading' && (
              <Alert
                message={`📥 正在下载: ${downloadStatus.filename}`}
                type="info"
                showIcon
                closable
                onClose={() => setDownloadStatus(null)}
              />
            )}
            {downloadStatus.type === 'success' && (
              <Alert
                message={`✅ 下载完成: ${downloadStatus.filename}`}
                type="success"
                showIcon
                closable
                onClose={() => setDownloadStatus(null)}
              />
            )}
            {downloadStatus.type === 'error' && (
              <Alert
                message={`❌ 下载失败: ${downloadStatus.filename}`}
                type="error"
                showIcon
                closable
                onClose={() => setDownloadStatus(null)}
              />
            )}
          </div>
        )}

        {/* 上传区域 */}
        <div className="upload-section">
          <Upload.Dragger
            multiple
            showUploadList={false}
            beforeUpload={handleUpload}
            className="upload-dragger"
            disabled={uploading}
          >
            <div className="upload-content">
              <UploadOutlined className="upload-icon" />
              <div className="upload-text">
                <div>点击或拖拽文件到此处上传</div>
                <div className="upload-hint">支持单个或批量上传，最大 10MB</div>
              </div>
            </div>
          </Upload.Dragger>
          {uploading && (
            <div className="upload-progress">
              <Progress percent={50} status="active" showInfo={false} />
              <div>上传中...</div>
            </div>
          )}
        </div>

        {/* 文件列表 */}
        <div className="file-list-section">
          {filteredFiles.length === 0 ? (
            <div className="empty-state">
              <FileOutlined className="empty-icon" />
              <div className="empty-text">
                {files.length === 0 ? '暂无文件' : '未找到匹配的文件'}
              </div>
              <div className="empty-hint">
                {files.length === 0
                  ? '上传第一个文件开始使用家庭网盘'
                  : '尝试调整搜索条件或清除筛选'
                }
              </div>
            </div>
          ) : (
            <>
              <div className="file-count">
                找到 {filteredFiles.length} 个文件
                {filters.keyword && ` (搜索: "${filters.keyword}")`}
                {filters.fileType !== 'all' && ` (类型: ${filters.fileType})`}
              </div>
              <div className="file-grid-container">
                <Row gutter={[16, 16]} className="file-grid">
                  {filteredFiles.map((file, index) => (
                    <Col xs={24} sm={12} md={8} lg={6} key={file.id || index}>
                      <div className="file-card">
                        <div className="file-header">
                          {getFileIcon(file.name)}
                          <span className="file-name" title={file.name}>
                            {file.name}
                          </span>
                        </div>
                        <div className="file-info">
                          <div className="file-meta">
                            <div className="file-size">
                              <strong>大小:</strong> {formatFileSize(file.size)}
                            </div>
                            <div className="file-date">
                              <strong>上传:</strong> {formatDate(file.uploadTime)}
                            </div>
                            <div className="file-type">
                              <Tag color={
                                file.category === 'image' ? 'green' :
                                file.category === 'document' ? 'blue' :
                                file.category === 'video' ? 'purple' :
                                file.category === 'archive' ? 'orange' : 'default'
                              }>
                                {file.category === 'image' ? '图片' :
                                 file.category === 'document' ? '文档' :
                                 file.category === 'video' ? '视频' :
                                 file.category === 'archive' ? '压缩包' : '其他'}
                              </Tag>
                            </div>
                          </div>
                          <div className="file-actions">
                            <Button
                              type="link"
                              icon={<DownloadOutlined />}
                              onClick={() => handleDownload(file.name)}
                              title="下载"
                              loading={downloading === file.name}
                              disabled={!!downloading}
                            >
                              下载
                            </Button>
                            <Button
                              type="link"
                              danger
                              icon={<DeleteOutlined />}
                              onClick={() => handleDelete(file.name)}
                              title="删除"
                              disabled={!!downloading}
                            >
                              删除
                            </Button>
                          </div>
                        </div>
                      </div>
                    </Col>
                  ))}
                </Row>
              </div>
            </>
          )}
        </div>
      </Card>
    </div>
  );
};

export default FileManager;