# 反向代理配置指南

## 概述

本文档说明如何在不同反向代理后面部署 ScoreRoute AI Gateway。

## Nginx 配置

### 关键配置点

**必须添加以下请求头：**
```nginx
# Authorization header 必须显式转发
proxy_set_header Authorization $http_authorization;

# 建议添加的代理头
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
```

### 完整配置示例

```nginx
server {
    listen 443 ssl;
    server_name api.your-domain.com;

    ssl_certificate /etc/ssl/certs/your-cert.pem;
    ssl_certificate_key /etc/ssl/private/your-cert.key;

    client_max_body_size 64M;

    # 前端静态文件
    location / {
        root /path/to/scoreroute/web/dist;
        try_files $uri $uri/ /index.html;
    }

    # API 代理 - 必须转发 Authorization
    location /api/ {
        proxy_pass http://127.0.0.1:3000/api/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Authorization $http_authorization;
        proxy_read_timeout 300s;
    }

    # V1 API 代理 (流式响应)
    location /v1/ {
        proxy_pass http://127.0.0.1:3000/v1/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Authorization $http_authorization;
        proxy_read_timeout 300s;
        
        # 流式响应配置
        proxy_buffering off;
        proxy_cache off;
        chunked_transfer_encoding on;
    }
}
```

## Apache 配置

```apache
<VirtualHost *:443>
    ServerName api.your-domain.com
    
    SSLEngine on
    SSLCertificateFile /path/to/cert.pem
    SSLCertificateKeyFile /path/to/key.pem

    # 前端静态文件
    DocumentRoot /path/to/scoreroute/web/dist
    
    <Directory "/path/to/scoreroute/web/dist">
        RewriteEngine On
        RewriteBase /
        RewriteRule ^index\.html$ - [L]
        RewriteRule . /index.html [L]
    </Directory>

    # API 代理
    ProxyPreserveHost On
    ProxyPass /api/ http://127.0.0.1:3000/api/
    ProxyPassReverse /api/ http://127.0.0.1:3000/api/
    
    # V1 API 代理 (流式响应)
    ProxyPass /v1/ http://127.0.0.1:3000/v1/
    ProxyPassReverse /v1/ http://127.0.0.1:3000/v1/

    # 转发 Authorization header
    RequestHeader set Authorization "%{HTTP_AUTHORIZATION}e"
</VirtualHost>
```

## 常见问题

### Q: API 返回 401 未授权错误？

**A:** 检查 Nginx 是否转发了 Authorization header：
```nginx
# 错误：缺少此行
proxy_set_header Authorization $http_authorization;
```

### Q: 流式响应不工作？

**A:** 确保关闭代理缓冲：
```nginx
proxy_buffering off;
proxy_cache off;
```

### Q: CORS 问题？

**A:** 添加适当的 CORS 头：
```nginx
add_header Access-Control-Allow-Origin *;
add_header Access-Control-Allow-Methods "GET, POST, OPTIONS";
add_header Access-Control-Allow-Headers "Authorization, Content-Type";
```

## 使用 Let's Encrypt SSL

```bash
# 安装 certbot
sudo apt install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d api.your-domain.com

# 自动续期测试
sudo certbot renew --dry-run
```
