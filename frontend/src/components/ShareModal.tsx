// frontend/src/components/ShareModal.tsx
import React, { useState } from 'react';

interface ShareModalProps {
  file: { filename: string };
  onClose: () => void;
}

const ShareModal: React.FC<ShareModalProps> = ({ file, onClose }) => {
  const [shareOptions, setShareOptions] = useState({
    expire_hours: 24,
    max_access: 10,
    password: '',
  });
  const [shareLink, setShareLink] = useState('');
  const [loading, setLoading] = useState(false);

  const createShare = async () => {
    setLoading(true);
    try {
      const response = await fetch(`/api/files/share/${encodeURIComponent(file.filename)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...shareOptions,
          user_id: 1 // 从登录状态获取
        })
      });
      
      const result = await response.json();
      if (result.success) {
        setShareLink(result.data.share_url);
      }
    } catch (error) {
      console.error('创建分享失败:', error);
      alert('创建分享失败，请检查后端服务');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="share-modal">
        <h3>分享文件: {file.filename}</h3>
        <button className="close-btn" onClick={onClose}>×</button>
        
        {!shareLink ? (
          <div className="share-form">
            <div className="form-group">
              <label>有效期:</label>
              <input 
                type="number" 
                value={shareOptions.expire_hours}
                onChange={e => setShareOptions({...shareOptions, expire_hours: +e.target.value})}
              />
              <span>小时</span>
            </div>
            
            <div className="form-group">
              <label>最大访问次数:</label>
              <input 
                type="number" 
                value={shareOptions.max_access}
                onChange={e => setShareOptions({...shareOptions, max_access: +e.target.value})}
              />
            </div>
            
            <div className="form-group">
              <label>访问密码:</label>
              <input 
                type="password" 
                placeholder="可选"
                value={shareOptions.password}
                onChange={e => setShareOptions({...shareOptions, password: e.target.value})}
              />
            </div>
            
            <button onClick={createShare} disabled={loading}>
              {loading ? '生成中...' : '生成分享链接'}
            </button>
          </div>
        ) : (
          <div className="share-result">
            <p>✅ 分享链接创建成功！</p>
            <div className="link-container">
              <input type="text" value={shareLink} readOnly />
              <button onClick={() => navigator.clipboard.writeText(shareLink)}>
                复制
              </button>
            </div>
            <p>链接有效期: {shareOptions.expire_hours}小时</p>
            <button onClick={onClose}>关闭</button>
          </div>
        )}
      </div>
    </div>
  );
};

export default ShareModal;