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

            <h4>渠道限制设置</h4>
            <p>可以为渠道设置以下限制:</p>
            <ul>
              <li><strong>总Token限制</strong> - 渠道可用Token总数上限（0表示无限制）</li>
              <li><strong>API有效期</strong> - 渠道API密钥的过期时间（不设置则永不过期）</li>
              <li><strong>调用频率限制</strong> - 支持多种时间窗口：
                <ul>
                  <li>调用次数限制（每分钟/小时/天/周/月/年）</li>
                  <li>Token数量限制（每分钟/小时/天/周/月/年）</li>
                </ul>
              </li>
            </ul>
            <p><strong>重叠限制:</strong> 可以同时设置多个限制规则，例如：每5小时最多1500次调用且每天最多3000次。</p>

            <h3>模型管理</h3>
            <p>模型是具体可调用的AI模型，关联到渠道:</p>
            <ul>
              <li>每个模型必须属于一个渠道</li>
              <li>模型类型: Chat(对话)、Embedding(向量)、Image(图片)、Video(视频)</li>
              <li>支持批量添加模型</li>
            </ul>
            <p><strong>提示:</strong> 点击"从API获取可用模型"自动获取渠道支持的模型列表。</p>

            <h4>模型限制设置</h4>
            <p>可以为模型设置以下限制:</p>
            <ul>
              <li><strong>总Token限制</strong> - 模型可用Token总数上限（0表示无限制）</li>
              <li><strong>模型有效期</strong> - 模型API的过期时间（不设置则永不过期）</li>
              <li><strong>调用频率限制</strong> - 支持多种时间窗口：
                <ul>
                  <li>调用次数限制（每分钟/小时/天/周/月/年）</li>
                  <li>Token数量限制（每分钟/小时/天/周/月/年）</li>
                </ul>
              </li>
              <li><strong>每Token费用</strong> - 设置模型的调用成本</li>
              <li><strong>货币单位</strong> - 人民币(CNY)或美元(USD)</li>
            </ul>
            <p><strong>重叠限制:</strong> 可以同时设置多个限制规则，例如：每5小时最多1500次调用且每天最多3000次。</p>

            <h4>汇率设置</h4>
            <p>在"模型管理"页面点击"汇率设置"按钮可以配置:</p>
            <ul>
              <li><strong>USD兑换CNY汇率</strong> - 用于统一计算成本</li>
              <li><strong>默认货币</strong> - 成本计算的默认货币单位</li>
            </ul>
            <p><strong>成本计算公式:</strong> 总成本 = Token数 × 每Token成本 × 汇率</p>

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
            <p>A: 请求过于频繁，可能原因：</p>
            <ul>
              <li>渠道设置了调用频率限制（如每分钟/小时限制）</li>
              <li>渠道设置了Token总量限制</li>
              <li>渠道API已过期</li>
            </ul>
            <p>请检查渠道的限制设置，或升级渠道配额。</p>

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
            <p>样本分析功能用于保存和分析API调用样本，帮助了解模型表现和行为，特别适用于Agent场景下的模型评估。</p>

            <h3>样本保存规则</h3>
            <ul>
              <li><strong>Token数量要求:</strong> 只保存响应Token数 >= 500 的请求</li>
              <li><strong>模型去重:</strong> 每个模型只保存一个最新样本</li>
              <li><strong>有效期:</strong> 样本保存7天，到期后自动删除</li>
              <li><strong>异步保存:</strong> 样本保存不影响API响应速度</li>
            </ul>

            <h3>页面结构（4个板块）</h3>
            <ul>
              <li><strong>样本展示:</strong> 分页展示所有样本，可查看详情和删除</li>
              <li><strong>样本评分:</strong> 展示模型分析评分结果，可修改评分</li>
              <li><strong>分析日志:</strong> 展示样本分析过程的实时日志</li>
              <li><strong>LLM设置:</strong> 配置用于分析的大模型API</li>
            </ul>

            <h3>分析机制</h3>
            <ul>
              <li><strong>分析频率:</strong> 每2小时自动分析最多20个样本</li>
              <li><strong>分析顺序:</strong> 按剩余时间升序（即将过期的优先分析）</li>
              <li><strong>分析后处理:</strong> 分析完成后自动删除样本</li>
              <li><strong>评分有效期:</strong> 评分保存7天，过期后自动清理</li>
              <li><strong>重试机制:</strong> 分析失败时自动重试最多3次，每次间隔2秒递增</li>
            </ul>

            <h3>评分体系（1-100分）</h3>
            <ul>
              <li><strong>工具调用 (30%):</strong> 是否正确识别并调用工具，参数是否正确</li>
              <li><strong>完整性 (25%):</strong> 是否完整回复用户请求，无遗漏</li>
              <li><strong>上下文理解 (20%):</strong> 是否正确理解对话上下文</li>
              <li><strong>错误处理 (15%):</strong> 错误处理和模糊请求处理能力</li>
              <li><strong>回复质量 (10%):</strong> 回复的清晰度和格式</li>
            </ul>

            <h3>模型评分权重（整合到模型评分页面）</h3>
            <ul>
              <li><strong>成功率 (28%):</strong> 成功请求占总请求的比例</li>
              <li><strong>延迟分数 (21%):</strong> 基于平均延迟计算，延迟越低分数越高</li>
              <li><strong>稳定性 (21%):</strong> 基于样本量计算，样本越多评分越可靠</li>
              <li><strong>用户评分 (15%):</strong> 用户对模型的评分（1-100）</li>
              <li><strong>样本分析 (15%):</strong> 基于样本分析评分（Agent能力评估）</li>
            </ul>

            <h3>数据结构</h3>
            <pre><code>样本 {
  id: number,
  model_key: string,        // 格式_类型_模型名称
  request_content: string,  // 请求JSON
  response_content: string, // 响应JSON
  token_count: number,       // Token数
  created_at: string,        // 创建时间
  expires_at: string,       // 过期时间
  remaining_minutes: number, // 剩余分钟数
  remaining_days: number     // 剩余天数
}

样本评分 {
  model_key: string,
  score: number,                    // 总分 1-100
  tool_calling_score: number,       // 工具调用评分
  completeness_score: number,        // 完整性评分
  context_understanding_score: number, // 上下文理解评分
  error_handling_score: number,     // 错误处理评分
  response_quality_score: number,   // 回复质量评分
  analyzed_at: string,              // 分析时间
  expires_at: string                // 过期时间
}

分析日志 {
  id: number,
  model_key: string,        // 模型名称
  analysis_time: string,     // 分析时间
  delete_time: string,      // 删除时间
  success: number,          // 是否成功 0/1
  error_message: string,    // 错误信息
  score: number             // 得分
}</code></pre>

            <h3>API端点</h3>
            <ul>
              <li><strong>GET /api/sample-analysis/config:</strong> 获取LLM配置</li>
              <li><strong>POST /api/sample-analysis/config:</strong> 保存LLM配置</li>
              <li><strong>POST /api/sample-analysis/config/test:</strong> 测试LLM连接</li>
              <li><strong>POST /api/sample-analysis/run:</strong> 手动运行分析</li>
              <li><strong>GET /api/sample-analysis/logs:</strong> 获取分析日志</li>
              <li><strong>GET /api/sample-analysis/logs/stats:</strong> 获取日志统计</li>
              <li><strong>GET /api/sample-analysis/ratings:</strong> 获取评分列表</li>
              <li><strong>PUT /api/sample-analysis/ratings:</strong> 更新评分</li>
            </ul>

            <h3>LLM设置说明</h3>
            <p>在"LLM设置"板块中配置用于分析样本的大模型:</p>
            <ul>
              <li><strong>API格式:</strong> OpenAI/Anthropic/Google/Zhipu</li>
              <li><strong>API地址:</strong> 模型的API端点</li>
              <li><strong>API Key:</strong> 访问密钥</li>
              <li><strong>模型名称:</strong> 用于分析的具体模型（如 gpt-4, claude-3-opus）</li>
              <li><strong>启用开关:</strong> 控制是否启用自动分析</li>
            </ul>
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
