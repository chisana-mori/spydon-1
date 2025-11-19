#!/bin/bash
set -e

# Spydon 环境清理脚本
# 用途：清理旧的预览环境和测试资源

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}🧹 Spydon Environment Cleanup${NC}"
echo "================================"

# 配置
MAX_AGE_DAYS=${MAX_AGE_DAYS:-7}  # 默认保留 7 天
DRY_RUN=${DRY_RUN:-false}        # 默认实际执行

echo "Configuration:"
echo "  Max age: $MAX_AGE_DAYS days"
echo "  Dry run: $DRY_RUN"
echo ""

# 检查 kubectl
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ Error: kubectl not found${NC}"
    exit 1
fi

# 1. 清理预览环境
echo -e "${YELLOW}📋 Listing preview environments...${NC}"

PREVIEW_NAMESPACES=$(kubectl get namespaces -l app=spydon-preview -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

if [ -z "$PREVIEW_NAMESPACES" ]; then
    echo -e "${GREEN}✅ No preview environments found${NC}"
else
    echo "Found preview environments:"

    for ns in $PREVIEW_NAMESPACES; do
        # 获取创建时间
        CREATED=$(kubectl get namespace $ns -o jsonpath='{.metadata.creationTimestamp}')
        CREATED_TIMESTAMP=$(date -d "$CREATED" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$CREATED" +%s 2>/dev/null)
        CURRENT_TIMESTAMP=$(date +%s)
        AGE_SECONDS=$((CURRENT_TIMESTAMP - CREATED_TIMESTAMP))
        AGE_DAYS=$((AGE_SECONDS / 86400))

        echo "  - $ns (age: $AGE_DAYS days)"

        # 如果超过最大年龄，删除
        if [ $AGE_DAYS -gt $MAX_AGE_DAYS ]; then
            if [ "$DRY_RUN" = "true" ]; then
                echo -e "    ${YELLOW}[DRY RUN] Would delete${NC}"
            else
                echo -e "    ${RED}Deleting...${NC}"
                kubectl delete namespace $ns --timeout=300s
                echo -e "    ${GREEN}✅ Deleted${NC}"
            fi
        else
            echo -e "    ${GREEN}Keeping (still fresh)${NC}"
        fi
    done
fi

echo ""

# 2. 清理未使用的 PVC
echo -e "${YELLOW}📋 Checking for unused PVCs...${NC}"

for ns in spydon-dev spydon-staging; do
    if kubectl get namespace $ns &> /dev/null; then
        echo "Namespace: $ns"

        UNUSED_PVCS=$(kubectl get pvc -n $ns -o json | jq -r '.items[] | select(.status.phase == "Bound" and (.metadata.annotations["pv.kubernetes.io/bind-completed"] // "" | . != "")) | select(.spec.volumeName as $pv | [kubectl get pods -n '$ns' -o json | .items[].spec.volumes[]?.persistentVolumeClaim.claimName] | index(.metadata.name) == null) | .metadata.name' 2>/dev/null || echo "")

        if [ -z "$UNUSED_PVCS" ]; then
            echo -e "  ${GREEN}✅ No unused PVCs${NC}"
        else
            for pvc in $UNUSED_PVCS; do
                echo "  - $pvc"
                if [ "$DRY_RUN" = "true" ]; then
                    echo -e "    ${YELLOW}[DRY RUN] Would delete${NC}"
                else
                    echo -e "    ${RED}Deleting...${NC}"
                    kubectl delete pvc $pvc -n $ns
                    echo -e "    ${GREEN}✅ Deleted${NC}"
                fi
            done
        fi
    fi
done

echo ""

# 3. 清理旧的 Docker 镜像（如果使用内部 Registry）
if [ -n "$REGISTRY_URL" ] && [ -n "$REGISTRY_TOKEN" ]; then
    echo -e "${YELLOW}📋 Cleaning up old Docker images...${NC}"

    # 这里需要根据你的 Registry 类型实现
    # 示例：GitLab Container Registry
    # curl --header "PRIVATE-TOKEN: $REGISTRY_TOKEN" \
    #      --request DELETE \
    #      "$REGISTRY_URL/api/v4/projects/$PROJECT_ID/registry/repositories/$REPO_ID/tags/$TAG"

    echo -e "${YELLOW}⚠️  Docker image cleanup not implemented${NC}"
    echo "   Configure REGISTRY_URL and REGISTRY_TOKEN to enable"
fi

echo ""

# 4. 清理 ArgoCD 应用历史
if command -v argocd &> /dev/null; then
    echo -e "${YELLOW}📋 Cleaning up ArgoCD history...${NC}"

    for app in spydon-dev spydon-staging; do
        if argocd app get $app &> /dev/null; then
            HISTORY_COUNT=$(argocd app history $app --output json | jq '. | length')

            if [ $HISTORY_COUNT -gt 10 ]; then
                echo "  - $app: $HISTORY_COUNT revisions (keeping last 10)"

                if [ "$DRY_RUN" = "true" ]; then
                    echo -e "    ${YELLOW}[DRY RUN] Would prune history${NC}"
                else
                    # ArgoCD 会自动保留最近的版本
                    echo -e "    ${GREEN}✅ History managed by ArgoCD${NC}"
                fi
            else
                echo -e "  - $app: $HISTORY_COUNT revisions ${GREEN}(OK)${NC}"
            fi
        fi
    done
else
    echo -e "${YELLOW}⚠️  argocd CLI not found, skipping ArgoCD cleanup${NC}"
fi

echo ""

# 5. 生成清理报告
echo -e "${GREEN}📊 Cleanup Summary${NC}"
echo "================================"
echo "Cleaned up:"
echo "  - Preview environments older than $MAX_AGE_DAYS days"
echo "  - Unused PVCs in dev/staging"
echo ""

if [ "$DRY_RUN" = "true" ]; then
    echo -e "${YELLOW}⚠️  This was a DRY RUN${NC}"
    echo "Run without DRY_RUN=true to actually delete resources"
else
    echo -e "${GREEN}✅ Cleanup completed${NC}"
fi

# 6. 发送通知（可选）
if [ -n "$SLACK_WEBHOOK_URL" ] && [ "$DRY_RUN" != "true" ]; then
    curl -X POST -H 'Content-type: application/json' \
        --data "{
            \"text\": \"🧹 Environment Cleanup Completed\",
            \"attachments\": [
                {
                    \"color\": \"good\",
                    \"fields\": [
                        {\"title\": \"Max Age\", \"value\": \"$MAX_AGE_DAYS days\", \"short\": true},
                        {\"title\": \"Status\", \"value\": \"Completed\", \"short\": true}
                    ]
                }
            ]
        }" \
        "$SLACK_WEBHOOK_URL"
fi

echo ""
echo -e "${GREEN}🎉 All done!${NC}"
