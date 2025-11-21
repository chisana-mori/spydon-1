export const sreSkillTemplate = {
    type: "doc",
    content: [
        {
            type: "heading",
            attrs: { level: 1 },
            content: [{ type: "text", text: "🧠 排查增强技巧" }],
        },
        {
            type: "blockquote",
            content: [
                {
                    type: "paragraph",
                    content: [
                        {
                            type: "text",
                            text: "💡 提示：此内容将作为补充上下文提供给 AI，用于增强其分析能力。请重点描述特定于该告警的非通用排查逻辑、业务背景或已知坑点，避免与通用排查步骤冲突。",
                        },
                    ],
                },
            ],
        },
        {
            type: "heading",
            attrs: { level: 2 },
            content: [{ type: "text", text: "🎯 关键检查点" }],
        },
        {
            type: "bulletList",
            content: [
                {
                    type: "listItem",
                    content: [
                        {
                            type: "paragraph",
                            content: [
                                {
                                    type: "text",
                                    text: "特定依赖服务状态（例如：是否依赖外部支付网关？）",
                                },
                            ],
                        },
                    ],
                },
                {
                    type: "listItem",
                    content: [
                        {
                            type: "paragraph",
                            content: [
                                {
                                    type: "text",
                                    text: "特殊配置项检查（例如：JVM 参数、连接池配置）",
                                },
                            ],
                        },
                    ],
                },
            ],
        },
        {
            type: "heading",
            attrs: { level: 2 },
            content: [{ type: "text", text: "📝 历史经验 / 文档粘贴" }],
        },
        {
            type: "paragraph",
            content: [
                {
                    type: "text",
                    text: "（请在此处直接粘贴 Markdown 格式的历史排查记录、Wiki 文档或 Postmortem。编辑器会自动将其渲染为富文本格式。）",
                },
            ],
        },
        {
            type: "paragraph",
            content: [],
        },
    ],
}
