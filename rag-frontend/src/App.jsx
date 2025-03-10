import React from "react";
import { Layout, Typography, Divider } from "antd";
import UploadComponent from "./components/UploadComponent";
import QueryComponent from "./components/QueryComponent";
import DocumentList from "./components/DocumentList";

const { Header, Content, Footer } = Layout;
const { Title } = Typography;

function App() {
  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Header style={{ background: "#1890ff", textAlign: "center", padding: "10px" }}>
        <Title style={{ color: "#fff", margin: 0 }}>RAG 文档系统</Title>
      </Header>
      <Content style={{ padding: "20px", maxWidth: "800px", margin: "auto" }}>
        <Title level={3}>📂 上传文档</Title>
        <UploadComponent />
        <Divider />
        <Title level={3}>📜 文档列表</Title>
        <DocumentList />
        <Divider />
        <Title level={3}>💬 询问 RAG 系统</Title>
        <QueryComponent />
      </Content>
      <Footer style={{ textAlign: "center" }}>RAG 系统 © 2025</Footer>
    </Layout>
  );
}

export default App;