// import logo from './logo.svg';
import DocumentList from './components/DocumentList.jsx';
import QueryComponent from './components/QueryComponent.jsx';
import UploadComponent from './components/UploadComponent.jsx';
import { useState } from 'react';

function App() {
  // 状态管理
  const [documents, setDocuments] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  // 刷新文档列表的函数
  const refreshDocuments = () => {
    setRefreshTrigger(prev => prev + 1);
  };

  return (
    <div style={{ 
      padding: '20px', 
      maxWidth: '1200px', 
      margin: '0 auto',
      fontFamily: 'Arial, sans-serif'
    }}>
      <header style={{ 
        textAlign: 'center', 
        marginBottom: '30px',
        borderBottom: '1px solid #eaeaea',
        paddingBottom: '20px'
      }}>
        <h1 style={{ color: '#1890ff' }}>RAG 知识库系统</h1>
        <p style={{ color: '#666' }}>上传文档，提问问题，获取智能回答</p>
      </header>

      <div style={{ 
        display: 'grid', 
        gridTemplateColumns: '1fr 1fr', 
        gap: '20px',
        marginBottom: '30px'
      }}>
        {/* 上传组件 */}
        <div style={{ 
          padding: '20px', 
          border: '1px solid #eaeaea', 
          borderRadius: '8px',
          backgroundColor: '#f9f9f9'
        }}>
          <h2 style={{ marginTop: 0 }}>上传文档</h2>
          <UploadComponent 
            setIsLoading={setIsLoading} 
            refreshDocuments={refreshDocuments} 
          />
        </div>

        {/* 查询组件 */}
        <div style={{ 
          padding: '20px', 
          border: '1px solid #eaeaea', 
          borderRadius: '8px',
          backgroundColor: '#f9f9f9'
        }}>
          <h2 style={{ marginTop: 0 }}>提问问题</h2>
          <QueryComponent />
        </div>
      </div>

      {/* 文档列表 */}
      <div className="documents-section">
        <DocumentList 
          refreshTrigger={refreshTrigger} 
          onRefresh={refreshDocuments} 
        />
      </div>

      <footer style={{ 
        marginTop: '30px', 
        textAlign: 'center',
        color: '#999',
        fontSize: '14px',
        borderTop: '1px solid #eaeaea',
        paddingTop: '20px'
      }}>
        <p>RAG 知识库系统 &copy; {new Date().getFullYear()}</p>
      </footer>
    </div>
  );
}

export default App;
