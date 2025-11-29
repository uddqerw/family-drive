import React, { useState, useEffect, useMemo } from 'react';
import {
  Button, Upload, message, Card,
  Row, Col, Tag, Progress, Alert,
  Input, Select, Space, Modal
} from 'antd';
import {
  UploadOutlined, DownloadOutlined, DeleteOutlined,
  FileOutlined, FileImageOutlined, FilePdfOutlined,
  FileWordOutlined, FileExcelOutlined, FileZipOutlined,
  VideoCameraOutlined, SearchOutlined, ShareAltOutlined,
  ExclamationCircleOutlined
} from '@ant-design/icons';
import './FileManager.css';

const { Search } = Input;
const { Option } = Select;
const { confirm } = Modal;

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
  isPrivate?: boolean;
}

interface SearchFilters {
  keyword: string;
  fileType: string;
  sortBy: 'name' | 'size' | 'date' | 'type';
  sortOrder: 'asc' | 'desc';
}

// 分享模态框组件
interface ShareModalProps {
  file: FileItem;
  onClose: () => void;
  visible: boolean;
}

const ShareModal: React.FC<ShareModalProps> = ({ file, onClose, visible }) => {
  const [shareOptions, setShareOptions] = useState({
    expire_hours: 24,
    max_access: 10,
    password: '',
  });
  const [shareLink, setShareLink] = useState('');
  const [loading, setLoading] = useState(false);
  const [accessPassword, setAccessPassword] = useState(''); // 访问密码验证

  const createShare = async () => {
    setLoading(true);
    try {
      const response = await fetch(`https://localhost:8000/api/files/share/${encodeURIComponent(file.name)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...shareOptions,
          user_id: 1
        })
      });

      const result = await response.json();
      if (result.success) {
        setShareLink(result.data.share_url);
        message.success('分享链接创建成功！');
      } else {
        message.error(result.message || '创建分享失败');
      }
    } catch (error) {
      console.error('创建分享失败:', error);
      message.error('创建分享失败，请检查后端服务');
    } finally {
      setLoading(false);
    }
  };

  // 安全下载（需要密码验证）
  const handleSecureDownload = async () => {
    if (!accessPassword) {
      message.error('请输入访问密码');
      return;
    }

    try {
      const response = await fetch(`https://localhost:8000/api/files/secure-download/${encodeURIComponent(file.name)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          password: accessPassword,
          share_token: shareLink.split('/').pop() // 从分享链接提取token
        })
      });

      if (response.ok) {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.setAttribute('download', file.name);
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        window.URL.revokeObjectURL(url);
        message.success('下载成功！');
        
        // 减少剩余访问次数
        setShareOptions(prev => ({
          ...prev,
          max_access: Math.max(0, prev.max_access - 1)
        }));
      } else {
        const errorData = await response.json();
        message.error(errorData.error || '密码错误或链接已失效');
      }
    } catch (error) {
      console.error('安全下载失败:', error);
      message.error('下载失败，请重试');
    }
  };

  const handleCopyLink = () => {
    navigator.clipboard.writeText(shareLink);
    message.success('链接已复制到剪贴板！');
  };

  const resetForm = () => {
    setShareLink('');
    setAccessPassword('');
    setShareOptions({
      expire_hours: 24,
      max_access: 10,
      password: '',
    });
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  return (
    <Modal
      title={`🔗 生成分享链接: ${file.name}`}
      open={visible}
      onCancel={handleClose}
      footer={null}
      width={500}
    >
      {!shareLink ? (
        <div className="share-form">
          <div style={{ marginBottom: 16 }}>
            <label>有效期:</label>
            <Input
              type="number"
              value={shareOptions.expire_hours}
              onChange={e => setShareOptions({...shareOptions, expire_hours: +e.target.value})}
              addonAfter="小时"
              style={{ marginTop: 8 }}
            />
          </div>

          <div style={{ marginBottom: 16 }}>
            <label>最大访问次数:</label>
            <Input
              type="number"
              value={shareOptions.max_access}
              onChange={e => setShareOptions({...shareOptions, max_access: +e.target.value})}
              style={{ marginTop: 8 }}
            />
          </div>

          <div style={{ marginBottom: 24 }}>
            <label>访问密码 (可选):</label>
            <Input.Password
              placeholder="设置访问密码"
              value={shareOptions.password}
              onChange={e => setShareOptions({...shareOptions, password: e.target.value})}
              style={{ marginTop: 8 }}
            />
          </div>

          <Button
            type="primary"
            onClick={createShare}
            loading={loading}
            block
          >
            {loading ? '生成中...' : '生成分享链接'}
          </Button>
        </div>
      ) : (
        <div className="share-result">
          <Alert
            message="分享链接创建成功！"
            type="success"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <div style={{ marginBottom: 16 }}>
            <Input
              value={shareLink}
              readOnly
              style={{ marginBottom: 12 }}
              addonAfter={
                <Button type="primary" onClick={handleCopyLink}>
                  复制
                </Button>
              }
            />
            
            {/* 密码保护文件的下载区域 */}
            {shareOptions.password ? (
              <div style={{ 
                padding: '16px', 
                background: '#f6ffed', 
                border: '1px solid #b7eb8f',
                borderRadius: '6px'
              }}>
                <div style={{ marginBottom: 12, fontWeight: 500 }}>
                  🔒 此文件受密码保护
                </div>
                <Input.Password
                  placeholder="请输入访问密码"
                  value={accessPassword}
                  onChange={e => setAccessPassword(e.target.value)}
                  style={{ marginBottom: 8 }}
                />
                <Button 
                  type="primary" 
                  onClick={handleSecureDownload}
                  block
                >
                  验证密码并下载
                </Button>
              </div>
            ) : (
              <div style={{ textAlign: 'center' }}>
                <Button 
                  type="primary" 
                  icon={<DownloadOutlined />}
                  onClick={() => {
                    // 普通下载（无密码）
                    window.open(shareLink, '_blank');
                  }}
                >
                  直接下载文件
                </Button>
              </div>
            )}
          </div>
          <div style={{ color: '#666', fontSize: 12 }}>
            <div>有效期: {shareOptions.expire_hours} 小时</div>
            <div>最大访问次数: {shareOptions.max_access} 次</div>
            {shareOptions.password && <div>访问密码: 已设置</div>}
          </div>
          <Button onClick={handleClose} block style={{ marginTop: 16 }}>
            关闭
          </Button>
        </div>
      )}
    </Modal>
  );
};

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
  
  const [shareModalVisible, setShareModalVisible] = useState(false);
  const [selectedFile, setSelectedFile] = useState<FileItem | null>(null);

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

  // 修复：loadFiles 函数 - 处理 null 和错误情况
  const loadFiles = async () => {
    try {
      console.log('🔄 开始加载文件列表...');
      const response = await fetch('https://localhost:8000/api/files/list');

      if (response.ok) {
        const result = await response.json();
        console.log('📁 后端返回数据:', result);

        // 修复：处理 null 和空数据的情况
        if (!result) {
          console.log('📁 后端返回空数据，使用空数组');
          setFiles([]);
          return;
        }

        // 修复：统一处理不同的数据格式
        let fileArray: any[] = [];
        
        if (Array.isArray(result)) {
          // 格式1: 直接返回数组
          fileArray = result;
        } else if (result.success && Array.isArray(result.data)) {
          // 格式2: {success: true, data: [...]}
          fileArray = result.data;
        } else if (Array.isArray(result.data)) {
          // 格式3: {data: [...]}
          fileArray = result.data;
        } else {
          // 格式4: 其他未知格式，使用空数组
          console.log('❓ 未知数据格式，使用空数组');
          fileArray = [];
        }

        const filesWithCategory = fileArray.map((file: any, index: number) => ({
          id: file.id || Date.now() + index,
          name: file.name || '未知文件',
          size: file.size || 0,
          type: file.type || 'file',
          uploadTime: file.uploadTime || new Date().toISOString(),
          category: getFileCategory(file.name || ''),
          isPrivate: file.isPrivate || false
        }));

        setFiles(filesWithCategory);
        console.log('✅ 加载成功，文件数:', filesWithCategory.length);
        
      } else {
        console.log('❌ HTTP 请求失败，状态:', response.status);
        // 修复：HTTP 错误时也设置空数组
        setFiles([]);
        message.error('加载文件列表失败');
      }
    } catch (error) {
      console.error('🚨 加载文件列表失败:', error);
      // 修复：错误时设置空数组，避免页面崩溃
      setFiles([]);
      message.error('加载文件列表失败，请检查网络连接');
    }
  };

  // 过滤和排序文件
  const filteredFiles = useMemo(() => {
    let result = [...files];

    if (filters.keyword) {
      result = result.filter(file =>
        file.name.toLowerCase().includes(filters.keyword.toLowerCase())
      );
    }

    if (filters.fileType !== 'all') {
      result = result.filter(file => file.category === filters.fileType);
    }

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

  // 上传处理函数 - 私有网盘模式
  const handleUpload = async (file: File) => {
    setUploading(true);
    const formData = new FormData();
    formData.append('file', file);
    formData.append('is_hidden', 'true'); // 默认私密文件

    try {
      console.log('📤 上传私密文件:', file.name);
      const response = await fetch('https://localhost:8000/api/files/upload', {
        method: 'POST',
        body: formData,
      });

      if (response.ok) {
        const result = await response.json();
        console.log('✅ 私密上传成功:', result);
        
        // 上传成功后自动打开分享模态框
        const newFile = {
          id: result.id || Date.now(),
          name: file.name,
          size: file.size,
          type: file.type || 'file',
          uploadTime: result.uploadTime || new Date().toISOString(),
          category: getFileCategory(file.name)
        };
        
        setSelectedFile(newFile);
        setShareModalVisible(true);
        
        message.success(`🔒 文件 "${file.name}" 已私密上传！请设置分享链接`);
        await loadFiles();
      } else {
        const errorText = await response.text();
        console.error('❌ 上传失败:', errorText);
        throw new Error('上传失败');
      }
    } catch (error) {
      console.error('❌ 上传失败:', error);
      message.error('文件上传失败，请检查网络连接');
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
      const response = await fetch(`https://localhost:8000/api/files/download/${encodeURIComponent(filename)}`);

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
    confirm({
      title: '确认删除',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除文件 "${filename}" 吗？此操作不可撤销。`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          const response = await fetch(`https://localhost:8000/api/files/delete/${encodeURIComponent(filename)}`, {
            method: 'DELETE',
          });

          if (response.ok) {
            const result = await response.json();
            console.log('✅ 删除成功:', result);
            message.success(`文件 "${filename}" 删除成功`);
            await loadFiles();
          } else {
            const errorText = await response.text();
            console.error('❌ 删除失败:', errorText);
            throw new Error('删除失败');
          }
        } catch (error: any) {
          console.error('删除失败:', error);
          message.error('文件删除失败，请重试');
        }
      },
    });
  };

  // 处理过滤条件变化
  const handleFilterChange = (filterType: string, value: string) => {
    setFilters(prev => ({
      ...prev,
      [filterType]: value
    }));
  };

  // 获取文件类型统计
  /* const getFileStats = () => {
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
*/
  useEffect(() => {
    loadFiles();
  }, []);

  return (
    <div className="enhanced-file-manager">
      <Card
        title={
          <div className="card-header">
            <span>🔒 私有网盘</span>
            <Space>
              <Tag color="blue">
                {filteredFiles.length} 个文件
              </Tag>
              <Tag color="red">🔒 所有文件默认私密</Tag>
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
            <Search
              placeholder="搜索文件名..."
              value={filters.keyword}
              onChange={(e) => handleFilterChange('keyword', e.target.value)}
              style={{ width: 200 }}
              allowClear
              enterButton={<SearchOutlined />}
            />

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

        {/* 私密上传提示 */}
        <div style={{ 
          margin: '16px', 
          padding: '12px 16px', 
          background: '#fff2f0', 
          border: '1px solid #ffccc7',
          borderRadius: '6px',
          fontSize: '14px',
          color: '#a8071a'
        }}>
          🔒 <strong>私密上传提示：</strong>所有文件默认仅自己可见，需要生成分享链接后才能被他人访问
        </div>

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
                <div>点击或拖拽文件到此处私密上传</div>
                <div className="upload-hint">文件默认隐藏，需要分享链接才能访问</div>
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
                  ? '上传第一个私密文件，生成分享链接后他人才能访问'
                  : '所有文件默认私密，需要分享链接才能访问'
                }
              </div>
            </div>
          ) : (
            <>
              <div className="file-count">
                找到 {filteredFiles.length} 个私密文件
                {filters.keyword && ` (搜索: "${filters.keyword}")`}
                {filters.fileType !== 'all' && ` (类型: ${filters.fileType})`}
              </div>
              <div className="file-grid-container">
                <Row gutter={[16, 16]} className="file-grid">
                  {filteredFiles.map((file, index) => (
                    <Col xs={24} sm={12} md={8} lg={6} key={`${file.name}-${file.id || index}-${file.uploadTime}`}>
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
                              icon={<ShareAltOutlined />}
                              onClick={() => {
                                setSelectedFile(file);
                                setShareModalVisible(true);
                              }}
                              title="分享"
                              disabled={!!downloading}
                            >
                              分享
                            </Button>
                            <Button
                              type="link"
                              danger
                              icon={<DeleteOutlined />}
                              onClick={(e: React.MouseEvent) => {
                                e.stopPropagation();
                                e.preventDefault();
                                console.log('🔴 React删除事件:', file.name);
                                handleDelete(file.name);
                              }}
                              title="删除"
                              disabled={!!downloading}
                              style={{ 
                                outline: 'none',
                                flex: 1
                              }}
                              onFocus={(e) => {
                                e.currentTarget.style.outline = '2px solid #ff4d4f';
                                e.currentTarget.style.outlineOffset = '1px';
                              }}
                              onBlur={(e) => {
                                e.currentTarget.style.outline = 'none';
                              }}
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

      {/* 分享模态框 */}
      {selectedFile && (
        <ShareModal
          file={selectedFile}
          visible={shareModalVisible}
          onClose={() => setShareModalVisible(false)}
        />
      )}
    </div>
  );
};

export default FileManager;