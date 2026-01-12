
import React, { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';

interface HtmlPreviewProps {
    html: string;
    className?: string;
    style?: React.CSSProperties;
}

export function HtmlPreview({ html, className, style }: HtmlPreviewProps) {
    const iframeRef = useRef<HTMLIFrameElement>(null);
    const [mountNode, setMountNode] = useState<HTMLElement | null>(null);

    // Auto-resize logic
    useEffect(() => {
        const iframe = iframeRef.current;
        if (!iframe) return;

        const handleLoad = () => {
            if (iframe.contentWindow?.document?.body) {
                // Determine height - this can be tricky.
                const height = iframe.contentWindow.document.documentElement.scrollHeight;
                iframe.style.height = `${height}px`;
            }
        };

        // If we use srcDoc, the load event might fire or we might need to observe.
        iframe.addEventListener('load', handleLoad);

        // Trigger manually if already loaded or srcDoc updates quickly
        if (iframe.contentDocument && iframe.contentDocument.readyState === 'complete') {
            handleLoad();
        }

        return () => {
            iframe.removeEventListener('load', handleLoad);
        };
    }, [html]);

    // Update content cleanly
    useEffect(() => {
        const iframe = iframeRef.current;
        if (!iframe) return;

        // We use srcDoc for immediate rendering.
        // But to ensure scripts don't run or to be safe, we might just write to doc.
        // srcDoc is generally okay.
    }, [html]);

    return (
        <iframe
            ref={iframeRef}
            className={className}
            style={{
                width: '100%',
                border: 'none',
                backgroundColor: '#fff', // Email usually expects white bg
                overflow: 'hidden',
                ...style
            }}
            srcDoc={html}
            sandbox="allow-same-origin allow-popups" // Disallow scripts for safety if needed, but emails might use them? usually not.
        />
    );
}
