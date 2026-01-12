'use client';

import { useEffect, useRef, useState } from 'react';

interface HtmlPreviewProps {
    html: string;
    className?: string;
    onLoad?: () => void;
}

/**
 * HTML 预览组件
 * 使用 iframe 渲染 HTML 内容，支持完整的浏览器原生渲染
 */
export function HtmlPreview({ html, className, onLoad }: HtmlPreviewProps) {
    const iframeRef = useRef<HTMLIFrameElement>(null);
    const [isReady, setIsReady] = useState(false);

    useEffect(() => {
        const iframe = iframeRef.current;
        if (!iframe || !html) return;

        // 等待 iframe 加载完成
        const handleLoad = () => {
            setIsReady(true);
            onLoad?.();
        };

        iframe.addEventListener('load', handleLoad);

        // 写入 HTML 内容
        const doc = iframe.contentDocument || iframe.contentWindow?.document;
        if (doc) {
            doc.open();
            doc.write(html);
            doc.close();

            // 注入基础样式以确保更好的渲染效果
            const style = doc.createElement('style');
            style.textContent = `
                /* 确保图片自适应 */
                img {
                    max-width: 100%;
                    height: auto;
                }
                /* 确保链接有默认样式 */
                a {
                    color: #0066cc;
                    text-decoration: underline;
                }
                /* 防止内容溢出 */
                body {
                    margin: 0;
                    padding: 16px;
                    word-wrap: break-word;
                }
                /* 确保表格不溢出 */
                table {
                    max-width: 100%;
                    border-collapse: collapse;
                }
            `;
            doc.head?.appendChild(style);
        }

        return () => {
            iframe.removeEventListener('load', handleLoad);
            setIsReady(false);
        };
    }, [html, onLoad]);

    return (
        <div className={className}>
            {!isReady && (
                <div className="flex items-center justify-center h-[500px] bg-muted/20">
                    <div className="flex items-center gap-2 text-muted-foreground">
                        <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                        <span className="text-sm">加载预览中...</span>
                    </div>
                </div>
            )}
            <iframe
                ref={iframeRef}
                title="邮件预览"
                className="w-full min-h-[500px]"
                sandbox="allow-same-origin allow-scripts allow-forms allow-popups"
                style={{
                    border: 'none',
                    backgroundColor: '#fff',
                    display: isReady ? 'block' : 'none',
                }}
                // 允许同源策略，以便加载内联资源
                allow="accelerometer; camera; geolocation; microphone"
            />
        </div>
    );
}
