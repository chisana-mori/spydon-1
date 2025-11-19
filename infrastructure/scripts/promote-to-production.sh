#!/bin/bash
set -e

# Spydon 推广到生产环境脚本
# 用途：将 staging 验证通过的版本推广到 production

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GITOPS_DIR="$SCRIPT_DIR/../gitops"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Spydon Production Promotion${NC}"
echo "================================"

# 1. 检查当前分支
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo -e "${RED}❌ Error: Must be on main branch${NC}"
    exit 1
fi

# 2. 确保工作目录干净
if [ -n "$(git status --porcelain)" ]; then
    echo -e "${RED}❌ Error: Working directory is not clean${NC}"
    git status --short
    exit 1
fi

# 3. 拉取最新代码
echo -e "${YELLOW}📥 Pulling latest changes...${NC}"
git pull origin main

# 4. 获取 staging 版本
cd "$GITOPS_DIR"
STAGING_VERSION=$(grep "newTag:" environments/staging/kustomization.yaml | awk '{print $2}')

if [ -z "$STAGING_VERSION" ]; then
    echo -e "${RED}❌ Error: Could not find staging version${NC}"
    exit 1
fi

echo -e "${GREEN}📦 Staging version: $STAGING_VERSION${NC}"

# 5. 获取当前 production 版本
CURRENT_PROD_VERSION=$(grep "newTag:" environments/production/kustomization.yaml | awk '{print $2}')
echo -e "${YELLOW}📦 Current production version: $CURRENT_PROD_VERSION${NC}"

# 6. 确认推广
echo ""
echo -e "${YELLOW}⚠️  You are about to promote to PRODUCTION:${NC}"
echo "   From: $CURRENT_PROD_VERSION"
echo "   To:   $STAGING_VERSION"
echo ""
read -p "Are you sure? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo -e "${RED}❌ Promotion cancelled${NC}"
    exit 0
fi

# 7. 更新 production 配置
echo -e "${YELLOW}📝 Updating production configuration...${NC}"

# 更新镜像标签
sed -i.bak "s|newTag:.*|newTag: $STAGING_VERSION|g" environments/production/kustomization.yaml

# 复制版本信息
cp environments/staging/version.yaml environments/production/version.yaml
sed -i.bak "s|staging|production|g" environments/production/version.yaml
sed -i.bak "s|promoted_at:.*|promoted_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)|g" environments/production/version.yaml

# 清理备份文件
rm -f environments/production/*.bak

# 8. 验证配置
echo -e "${YELLOW}🔍 Validating configuration...${NC}"
if command -v kustomize &> /dev/null; then
    kustomize build environments/production > /tmp/production-manifests.yaml
    echo -e "${GREEN}✅ Configuration is valid${NC}"
else
    echo -e "${YELLOW}⚠️  kustomize not found, skipping validation${NC}"
fi

# 9. 提交变更
echo -e "${YELLOW}💾 Committing changes...${NC}"
cd "$SCRIPT_DIR/../.."
git add infrastructure/gitops/environments/production/
git commit -m "🚀 Promote $STAGING_VERSION to production

Promoted from staging after successful testing.

Previous version: $CURRENT_PROD_VERSION
New version: $STAGING_VERSION
Promoted by: $(git config user.name)
Promoted at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
"

# 10. 推送到远程
echo -e "${YELLOW}📤 Pushing to remote...${NC}"
git push origin main

echo ""
echo -e "${GREEN}✅ Production promotion completed!${NC}"
echo ""
echo "Next steps:"
echo "1. ArgoCD will detect the changes"
echo "2. Go to ArgoCD UI: https://argocd.example.com"
echo "3. Find 'spydon-production' application"
echo "4. Click 'SYNC' to deploy"
echo ""
echo "Or use ArgoCD CLI:"
echo "  argocd app sync spydon-production"
echo ""

# 11. 可选：自动触发 ArgoCD 同步
if command -v argocd &> /dev/null; then
    read -p "Trigger ArgoCD sync now? (yes/no): " SYNC_NOW

    if [ "$SYNC_NOW" = "yes" ]; then
        echo -e "${YELLOW}🔄 Syncing with ArgoCD...${NC}"
        argocd app sync spydon-production --prune

        echo -e "${YELLOW}⏳ Waiting for sync to complete...${NC}"
        argocd app wait spydon-production --timeout 600

        echo -e "${GREEN}✅ Production deployment completed!${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  argocd CLI not found, please sync manually${NC}"
fi

# 12. 发送通知（可选）
if [ -n "$SLACK_WEBHOOK_URL" ]; then
    curl -X POST -H 'Content-type: application/json' \
        --data "{
            \"text\": \"🚀 Production Promotion\",
            \"attachments\": [
                {
                    \"color\": \"good\",
                    \"fields\": [
                        {\"title\": \"Version\", \"value\": \"$STAGING_VERSION\", \"short\": true},
                        {\"title\": \"Previous\", \"value\": \"$CURRENT_PROD_VERSION\", \"short\": true},
                        {\"title\": \"Promoted by\", \"value\": \"$(git config user.name)\", \"short\": true},
                        {\"title\": \"Status\", \"value\": \"Waiting for ArgoCD sync\", \"short\": true}
                    ]
                }
            ]
        }" \
        "$SLACK_WEBHOOK_URL"
fi

echo -e "${GREEN}🎉 All done!${NC}"
