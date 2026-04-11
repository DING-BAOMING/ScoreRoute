<template>
  <div class="docs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>开发文档</span>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="API使用指南" name="usage">
          <div class="doc-content">
            <h2>API 使用指南</h2>
            
            <h3>获取 API Key</h3>
            <p>1. 进入"Token管理"页面</p>
            <p>2. 点击"生成Token"按钮</p>
            <p>3. 填写Token名称，选择格式和模型</p>
            <p>4. 点击确定，复制生成的Key</p>

            <h3>调用 API</h3>
            <p>Base URL: <code>{{ baseUrl }}</code></p>

            <h4>Chat Completions API</h4>
            <pre><code>curl -X POST {{ baseUrl }}/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_KEY" \
  -d '{
    "model": "minimaxai/minimax-m2.5",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'</code></pre>

            <h4>轮询所有模型</h4>
            <p>外部应用调用时，在model字段填写 <code>AUTO</code> 或 <code>POLL_ALL</code>，系统会自动轮询所有可用模型：</p>
            <pre><code>curl -X POST {{ baseUrl }}/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_KEY" \
  -d '{
    "model": "AUTO",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'</code></pre>
            <p><strong>说明：</strong>填写AUTO或POLL_ALL后，系统会忽略Token关联的模型，自动选择负载最低的模型。</p>

            <h4>List Models API</h4>
            <pre><code>curl -X GET {{ baseUrl }}/v1/models \
  -H "Authorization: Bearer YOUR_TOKEN_KEY"</code></pre>

            <h3>支持的模型</h3>
            <p>系统支持多种AI模型，包括:</p>
            <ul>
              <li><strong>文本模型:</strong> GPT系列、GLM系列、MiniMax等</li>
              <li><strong>视觉理解:</strong> GPT-4V、GLM-4V等</li>
              <li><strong>图像生成:</strong> DALL-E、CogView等</li>
              <li><strong>视频生成:</strong> CogVideoX等</li>
            </ul>
            <p>在"模型管理"页面可添加自定义模型，系统会自动根据负载均衡选择可用的渠道和模型。</p>
          </div>
        </el-tab-pane>

        <el-tab-pane label="管理指南" name="guide">
          <div class="doc-content">
            <h2>管理指南</h2>
            
            <h3>渠道管理</h3>
            <p>渠道是对接不同AI服务商的配置，包括:</p>
            <ul>
              <li>API地址 - AI服务商提供的API端点</li>
              <li>API Key - 渠道的身份凭证</li>
              <li>格式 - OpenAI/Anthropic/Azure/Google/智谱Zhipu</li>
            </ul>
            <p><strong>提示:</strong> 选择格式后，API地址会自动填充。</p>

            <h3>模型管理</h3>
            <p>模型是具体可调用的AI模型，关联到渠道:</p>
            <ul>
              <li>每个模型必须属于一个渠道</li>
              <li>模型类型: Chat(对话)、Embedding(向量)、Image(图片)、Video(视频)</li>
              <li>支持批量添加模型</li>
            </ul>
            <p><strong>提示:</strong> 点击"从API获取可用模型"自动获取渠道支持的模型列表。</p>

            <h3>模型调用页面</h3>
            <p>在"模型调用"页面可以:</p>
            <ul>
              <li>查看所有可用模型（按渠道+类型分组）</li>
              <li>快速启用/禁用模型</li>
              <li>了解系统的模型分布情况</li>
            </ul>
            <p><strong>格式+类型分组:</strong> 系统按"渠道名称+类型"分组展示模型，例如：</p>
            <ul>
              <li>OpenAI/Chat: gpt-3.5-turbo, gpt-4</li>
              <li>智谱Zhipu/Chat: GLM-4.7-Flash</li>
              <li>智谱Zhipu/Video: CogVideoX-Flash</li>
            </ul>

            <h3>Token管理</h3>
            <p>Token是分配给用户的API访问凭证:</p>
            <ul>
              <li>每个Token可以绑定特定的格式和模型</li>
              <li>Token Key仅显示一次，请妥善保存</li>
              <li>支持重新生成Key</li>
            </ul>

            <h3>调用日志</h3>
            <p>系统记录最近24小时的API调用日志，包括:</p>
            <ul>
              <li>调用时间、延迟、状态</li>
              <li>使用的Token和模型</li>
              <li>错误信息(如有)</li>
            </ul>
          </div>
        </el-tab-pane>

        <el-tab-pane label="常见问题" name="faq">
          <div class="doc-content">
            <h2>常见问题</h2>
            
            <h3>Q: 调用返回401错误?</h3>
            <p>A: 请检查API Key是否正确，以及Token是否被启用。</p>

            <h3>Q: 调用返回429错误?</h3>
            <p>A: 请求过于频繁，请降低调用频率或升级渠道配额。</p>

            <h3>Q: 调用返回500错误?</h3>
            <p>A: 上游服务商出错，请查看详细错误信息，或稍后重试。</p>

            <h3>Q: 如何添加新的AI渠道?</h3>
            <p>A: 1. 在渠道管理中添加新的渠道配置<br>
               2. 在模型管理中添加该渠道下的模型<br>
               3. 在Token管理中创建访问凭证</p>

            <h3>Q: 如何实现负载均衡?</h3>
            <p>A: 系统默认使用轮询(round-robin)策略，自动在多个渠道间分配请求。</p>

            <h3>Q: 渠道被禁用怎么办?</h3>
            <p>A: 检查渠道配置是否正确，API Key是否有效，渠道额度是否充足。</p>
          </div>
        </el-tab-pane>

        <el-tab-pane label="样本分析" name="sample">
          <div class="doc-content">
            <h2>样本分析功能</h2>
            
            <h3>功能说明</h3>
            <p>样本分析功能用于保存和分析API调用样本，帮助了解模型表现和行为。</p>

            <h3>样本保存规则</h3>
            <ul>
              <li><strong>Token数量要求:</strong> 只保存响应Token数 >= 1000 的请求</li>
              <li><strong>模型去重:</strong> 每个模型只保存一个最新样本</li>
              <li><strong>有效期:</strong> 样本保存7天，到期后自动删除</li>
              <li><strong>异步保存:</strong> 样本保存不影响API响应速度</li>
            </ul>

            <h3>样本组成</h3>
            <p>每个样本包含:</p>
            <ul>
              <li><strong>模型标识:</strong> 格式+类型+模型名称 (如: openai_chat_MiniMax-M2.5)</li>
              <li><strong>请求内容:</strong> 完整的API请求体</li>
              <li><strong>响应内容:</strong> 完整的API响应体</li>
              <li><strong>Token数:</strong> 响应的Token使用量</li>
            </ul>

            <h3>清理机制</h3>
            <ul>
              <li><strong>自动清理:</strong> 每小时自动删除过期样本</li>
              <li><strong>手动清理:</strong> 可通过样本分析页面手动触发清理</li>
            </ul>

            <h3>页面功能</h3>
            <ul>
              <li><strong>统计概览:</strong> 显示总样本数、模型数、平均Token数、过期样本数</li>
              <li><strong>样本列表:</strong> 按模型名称展示样本</li>
              <li><strong>查看详情:</strong> 点击查看完整请求/响应内容</li>
              <li><strong>删除样本:</strong> 手动删除不需要的样本</li>
              <li><strong>分析功能:</strong> 预留按钮（后续开发）</li>
            </ul>

            <h3>数据结构</h3>
            <pre><code>样本 {
  id: number,
  model_key: string,        // 格式_类型_模型名称
  request_content: string,  // 请求JSON
  response_content: string,   // 响应JSON
  token_count: number,       // Token数
  created_at: string,        // 创建时间
  expires_at: string,       // 过期时间
  remaining_minutes: number, // 剩余分钟数
  remaining_days: number     // 剩余天数
}</code></pre>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const activeTab = ref('usage')
const baseUrl = window.location.origin
</script>

<style scoped>
.card-header {
  font-size: 18px;
  font-weight: bold;
}

.doc-content {
  padding: 20px 0;
}

.doc-content h2 {
  margin-bottom: 20px;
  color: #303133;
}

.doc-content h3 {
  margin: 20px 0 10px;
  color: #606266;
}

.doc-content p {
  margin: 8px 0;
  color: #606266;
  line-height: 1.6;
}

.doc-content ul {
  margin: 10px 0;
  padding-left: 20px;
}

.doc-content li {
  margin: 5px 0;
  color: #606266;
}

.doc-content pre {
  background: #f5f7fa;
  padding: 15px;
  border-radius: 4px;
  overflow-x: auto;
  margin: 15px 0;
}

.doc-content code {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: Consolas, Monaco, monospace;
  color: #e6a23c;
}

.doc-content pre code {
  background: transparent;
  padding: 0;
  color: #409eff;
}
</style>
