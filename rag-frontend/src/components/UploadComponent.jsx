import React, { useState } from "react";
import { Upload, Button, message } from "antd";
import { UploadOutlined } from "@ant-design/icons";

const UploadComponent = ({ onUploadSuccess }) => {
  const [uploading, setUploading] = useState(false);

  const handleUpload = async (file) => {
    setUploading(true);
    const formData = new FormData();
    formData.append("file", file);

    try {
      const response = await fetch("http://localhost:8080/upload", {
        method: "POST",
        body: formData,
      });
      const result = await response.json();
      if (response.ok) {
        message.success("文件上传成功");
        onUploadSuccess(); // 刷新文档列表
      } else {
        message.error(result.error || "上传失败");
      }
    } catch (error) {
      message.error("上传请求失败");
    }
    setUploading(false);
  };

  return (
    <Upload
      beforeUpload={(file) => {
        handleUpload(file);
        return false; // 阻止默认上传行为
      }}
      showUploadList={false}
    >
      <Button icon={<UploadOutlined />} loading={uploading}>
        {uploading ? "上传中..." : "点击上传"}
      </Button>
    </Upload>
  );
};

export default UploadComponent;