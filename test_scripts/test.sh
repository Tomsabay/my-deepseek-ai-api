#!/bin/bash

# ==========================================
# AI API 完整流程测试脚本
# 测试内容：
# 1. 服务健康检查
# 2. 单轮对话
# 3. 多轮对话
# 4. 流式对话接口
# ==========================================

BASE_URL="http://localhost:8080"
API_URL="${BASE_URL}/api/v1"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}==========================================${NC}"
echo -e "${YELLOW}       开始执行 AI API 完整接口测试       ${NC}"
echo -e "${YELLOW}==========================================${NC}"

# ==========================================
# 1. 检查服务健康状态
# ==========================================
echo -e "\n${GREEN}[1/4] 测试健康检查接口 (/health)...${NC}"
HEALTH_RES=$(curl -s -m 5 ${BASE_URL}/health)
if [[ $HEALTH_RES == *"running"* ]]; then
  echo -e "✅ 健康检查通过，响应内容: $HEALTH_RES"
else
  echo -e "❌ 服务连接失败，请确认服务已启动。"
  exit 1
fi

# ==========================================
# 2. 基础单轮对话测试
# ==========================================
echo -e "\n${GREEN}[2/4] 测试普通单轮对话 POST /chat ...${NC}"
CHAT_RES=$(curl -s -X POST ${API_URL}/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "你好，请回复一个短句：测试通过"}' )

# 检查是否包含 reply 字段
if [[ $CHAT_RES == *"reply"* ]]; then
  echo -e "✅ 单轮对话正常，响应内容: "
  echo $CHAT_RES
else
  echo -e "❌ 单轮对话异常，响应: $CHAT_RES"
fi

# ==========================================
# 3. 多轮对话测试
# ==========================================
echo -e "\n${GREEN}[3/4] 测试多轮对话 POST /chat ...${NC}"
MULTI_CHAT_RES=$(curl -s -X POST ${API_URL}/chat \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "我的名字是赵少峰"},
      {"role": "assistant", "content": "你好，赵少峰！很高兴认识你。"},
      {"role": "user", "content": "我刚才说我的名字是什么？请只回答名字。"}
    ]
  }' )

if [[ $MULTI_CHAT_RES == *"reply"* ]]; then
  echo -e "✅ 多轮对话正常，响应内容 (期望看到提取的名字): "
  echo $MULTI_CHAT_RES
else
  echo -e "❌ 多轮对话异常，响应: $MULTI_CHAT_RES"
fi

# ==========================================
# 4. 流式测试
# ==========================================
echo -e "\n${GREEN}[4/4] 测试流式接口 /chat/stream... (仅读取部分字节)${NC}"
echo -e "⚠️ 流式响应输出示例 (打字机效果)："
curl -s -N -X POST ${API_URL}/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "请用10个字以内说一句话。"}' | head -n 10

echo -e "\n\n${YELLOW}==========================================${NC}"
echo -e "${GREEN}✨ 测试流程执行完毕！✨${NC}"
echo -e "${YELLOW}==========================================${NC}"
