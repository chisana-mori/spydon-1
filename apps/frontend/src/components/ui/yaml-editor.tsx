'use client'

import { useCallback } from 'react'
import SimpleEditor from 'react-simple-code-editor'
import { Highlight, themes } from 'prism-react-renderer'
import { useThemeStore } from '@/stores/themeStore'

// 自定义 YAML 语法高亮规则
const yamlLanguage = {
    comment: /#.*/,
    key: {
        pattern: /^[ \t]*[^\s:]+(?=:)/m,
        alias: 'property',
    },
    string: {
        pattern: /(["'])(?:(?!\1)[^\\\r\n]|\\.)*\1/,
        greedy: true,
    },
    boolean: /\b(?:true|false|yes|no)\b/i,
    number: /\b-?(?:0x[\da-f]+|\d+(?:\.\d+)?(?:e[+-]?\d+)?)\b/i,
    punctuation: /[:\-[\]{}]/,
}

interface YamlEditorProps {
    value: string
    onChange: (value: string) => void
    placeholder?: string
    height?: string
    disabled?: boolean
}

export function YamlEditor({
    value,
    onChange,
    placeholder = '',
    height = '400px',
    disabled = false,
}: YamlEditorProps) {
    const { theme } = useThemeStore()
    const isDark = theme === 'dark'

    const highlightCode = useCallback((code: string) => (
        <Highlight
            theme={isDark ? themes.vsDark : themes.vsLight}
            code={code}
            language="yaml"
        >
            {({ tokens, getLineProps, getTokenProps }) => (
                <>
                    {tokens.map((line, i) => (
                        <div key={i} {...getLineProps({ line })}>
                            {line.map((token, key) => (
                                <span key={key} {...getTokenProps({ token })} />
                            ))}
                        </div>
                    ))}
                </>
            )}
        </Highlight>
    ), [isDark])

    return (
        <div
            className="yaml-editor-wrapper"
            style={{
                height,
                position: 'relative',
                overflow: 'auto',
                borderRadius: '6px',
                border: '1px solid var(--border)',
                backgroundColor: isDark ? '#1e1e1e' : '#ffffff',
            }}
        >
            <SimpleEditor
                value={value}
                onValueChange={onChange}
                highlight={highlightCode}
                disabled={disabled}
                placeholder={placeholder}
                padding={12}
                style={{
                    fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace',
                    fontSize: 13,
                    lineHeight: 1.5,
                    minHeight: '100%',
                    backgroundColor: 'transparent',
                    color: isDark ? '#d4d4d4' : '#1f2937',
                }}
                textareaClassName="yaml-editor-textarea"
                preClassName="yaml-editor-pre"
            />
            <style jsx global>{`
                .yaml-editor-wrapper textarea,
                .yaml-editor-wrapper pre {
                    min-height: 100% !important;
                    outline: none !important;
                }
                .yaml-editor-wrapper textarea::placeholder {
                    color: ${isDark ? '#6b7280' : '#9ca3af'};
                }
                .yaml-editor-wrapper textarea:focus {
                    outline: none !important;
                    box-shadow: none !important;
                }
            `}</style>
        </div>
    )
}
