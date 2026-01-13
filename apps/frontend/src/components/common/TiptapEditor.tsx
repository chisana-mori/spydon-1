'use client';

import { useEditor, EditorContent, Editor } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import Link from '@tiptap/extension-link';
import Image from '@tiptap/extension-image';
import { Table } from '@tiptap/extension-table';
import { TableRow } from '@tiptap/extension-table-row';
import { TableCell } from '@tiptap/extension-table-cell';
import { TableHeader } from '@tiptap/extension-table-header';
import Underline from '@tiptap/extension-underline';
import TextAlign from '@tiptap/extension-text-align';
import { TextStyle } from '@tiptap/extension-text-style';
import { Color } from '@tiptap/extension-color';
import { Node, mergeAttributes } from '@tiptap/core';
import Paragraph from '@tiptap/extension-paragraph';
import Heading from '@tiptap/extension-heading';
import { Button } from '@/components/ui/button';
import {
    Bold, Italic, Underline as UnderlineIcon, Strikethrough,
    List, ListOrdered,
    AlignLeft, AlignCenter, AlignRight,
    Heading1, Heading2, Table as TableIcon,
    Undo, Redo
} from 'lucide-react';
import { useEffect } from 'react';
import { cn } from '@/lib/utils';

interface TiptapEditorProps {
    value: string;
    onChange: (content: string) => void;
}

const Toolbar = ({ editor }: { editor: Editor | null }) => {
    if (!editor) return null;

    const ToolbarButton = ({
        isActive,
        onClick,
        children
    }: {
        isActive?: boolean;
        onClick: () => void;
        children: React.ReactNode;
    }) => (
        <Button
            variant="ghost"
            size="sm"
            onClick={onClick}
            className={cn(
                "h-8 w-8 p-0",
                isActive ? "bg-muted text-foreground" : "text-muted-foreground hover:text-foreground"
            )}
        >
            {children}
        </Button>
    );

    return (
        <div className="border-b p-2 flex flex-wrap gap-1 bg-muted/20 items-center">
            <ToolbarButton
                isActive={editor.isActive('heading', { level: 1 })}
                onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
            >
                <Heading1 className="h-4 w-4" />
            </ToolbarButton>
            <ToolbarButton
                isActive={editor.isActive('heading', { level: 2 })}
                onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
            >
                <Heading2 className="h-4 w-4" />
            </ToolbarButton>

            <div className="w-px h-6 bg-border mx-1" />

            <ToolbarButton
                isActive={editor.isActive('bold')}
                onClick={() => editor.chain().focus().toggleBold().run()}
            >
                <Bold className="h-4 w-4" />
            </ToolbarButton>
            <ToolbarButton
                isActive={editor.isActive('italic')}
                onClick={() => editor.chain().focus().toggleItalic().run()}
            >
                <Italic className="h-4 w-4" />
            </ToolbarButton>
            <ToolbarButton
                isActive={editor.isActive('underline')}
                onClick={() => editor.chain().focus().toggleUnderline().run()}
            >
                <UnderlineIcon className="h-4 w-4" />
            </ToolbarButton>
            <ToolbarButton
                isActive={editor.isActive('strike')}
                onClick={() => editor.chain().focus().toggleStrike().run()}
            >
                <Strikethrough className="h-4 w-4" />
            </ToolbarButton>

            <div className="w-px h-6 bg-border mx-1" />

            <ToolbarButton
                isActive={editor.isActive({ textAlign: 'left' })}
                onClick={() => editor.chain().focus().setTextAlign('left').run()}
            >
                <AlignLeft className="h-4 w-4" />
            </ToolbarButton>
            <ToolbarButton
                isActive={editor.isActive({ textAlign: 'center' })}
                onClick={() => editor.chain().focus().setTextAlign('center').run()}
            >
                <AlignCenter className="h-4 w-4" />
            </ToolbarButton>
            <ToolbarButton
                isActive={editor.isActive({ textAlign: 'right' })}
                onClick={() => editor.chain().focus().setTextAlign('right').run()}
            >
                <AlignRight className="h-4 w-4" />
            </ToolbarButton>

            <div className="w-px h-6 bg-border mx-1" />

            <ToolbarButton
                isActive={editor.isActive('bulletList')}
                onClick={() => editor.chain().focus().toggleBulletList().run()}
            >
                <List className="h-4 w-4" />
            </ToolbarButton>
            <ToolbarButton
                isActive={editor.isActive('orderedList')}
                onClick={() => editor.chain().focus().toggleOrderedList().run()}
            >
                <ListOrdered className="h-4 w-4" />
            </ToolbarButton>

            <div className="w-px h-6 bg-border mx-1" />

            {/* Table Controls */}
            <ToolbarButton
                onClick={() => editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()}
            >
                <TableIcon className="h-4 w-4" />
            </ToolbarButton>

            <div className="ml-auto flex items-center gap-1">
                <ToolbarButton onClick={() => editor.chain().focus().undo().run()}>
                    <Undo className="h-4 w-4" />
                </ToolbarButton>
                <ToolbarButton onClick={() => editor.chain().focus().redo().run()}>
                    <Redo className="h-4 w-4" />
                </ToolbarButton>
            </div>
        </div>
    );
};

export function TiptapEditor({ value, onChange }: TiptapEditorProps) {
    const editor = useEditor({
        extensions: [
            StarterKit.configure({
                paragraph: false,
                heading: false,
            }),
            // Custom Div Node to preserve divs
            Node.create({
                name: 'div',
                group: 'block',
                content: 'block+',
                addAttributes() {
                    return {
                        style: {
                            default: null,
                            parseHTML: element => element.getAttribute('style'),
                            renderHTML: attributes => {
                                if (!attributes.style) return {};
                                return { style: attributes.style };
                            },
                        },
                        class: {
                            default: null,
                            parseHTML: element => element.getAttribute('class'),
                            renderHTML: attributes => {
                                if (!attributes.class) return {};
                                return { class: attributes.class };
                            },
                        }
                    };
                },
                parseHTML() {
                    return [{ tag: 'div' }];
                },
                renderHTML({ HTMLAttributes }) {
                    return ['div', mergeAttributes(HTMLAttributes), 0];
                },
            }),
            // Extended Paragraph to preserve style and class
            Paragraph.extend({
                addAttributes() {
                    return {
                        style: {
                            default: null,
                            parseHTML: element => element.getAttribute('style'),
                            renderHTML: attributes => {
                                if (!attributes.style) return {};
                                return { style: attributes.style };
                            },
                        },
                        class: {
                            default: null,
                            parseHTML: element => element.getAttribute('class'),
                            renderHTML: attributes => {
                                if (!attributes.class) return {};
                                return { class: attributes.class };
                            },
                        }
                    };
                }
            }),
            // Extended Heading to preserve style
            Heading.extend({
                addAttributes() {
                    return {
                        level: {
                            default: 1,
                        },
                        style: {
                            default: null,
                            parseHTML: element => element.getAttribute('style'),
                            renderHTML: attributes => {
                                if (!attributes.style) return {};
                                return { style: attributes.style };
                            },
                        }
                    };
                }
            }),
            TextStyle,
            Color,
            Link.configure({ openOnClick: false }),
            Image.configure({
                inline: true,
                allowBase64: true,
                HTMLAttributes: {
                    style: 'max-width: 100%; height: auto;',
                },
            }),
            Table.extend({
                addAttributes() {
                    return {
                        style: {
                            default: null,
                            parseHTML: element => element.getAttribute('style') || element.style.cssText,
                            renderHTML: attributes => {
                                if (!attributes.style) return {};
                                return { style: attributes.style };
                            },
                        },
                        width: {
                            default: null,
                            parseHTML: element => {
                                const widthAttr = element.getAttribute('width');
                                if (widthAttr) return widthAttr;
                                return element.style.width || null;
                            },
                            renderHTML: attributes => {
                                if (!attributes.width) return {};
                                return { width: attributes.width };
                            },
                        },
                        border: {
                            default: null,
                            parseHTML: element => element.getAttribute('border'),
                            renderHTML: attributes => {
                                if (!attributes.border) return {};
                                return { border: attributes.border };
                            },
                        },
                        cellpadding: {
                            default: null,
                            parseHTML: element => element.getAttribute('cellpadding'),
                            renderHTML: attributes => {
                                if (!attributes.cellpadding) return {};
                                return { cellpadding: attributes.cellpadding };
                            },
                        },
                        cellspacing: {
                            default: null,
                            parseHTML: element => element.getAttribute('cellspacing'),
                            renderHTML: attributes => {
                                if (!attributes.cellspacing) return {};
                                return { cellspacing: attributes.cellspacing };
                            },
                        }
                    };
                }
            }).configure({ resizable: false }),
            TableRow.extend({
                addAttributes() {
                    return {
                        style: {
                            default: null,
                            parseHTML: element => element.getAttribute('style') || element.style.cssText,
                            renderHTML: attributes => {
                                if (!attributes.style) return {};
                                return { style: attributes.style };
                            },
                        }
                    };
                }
            }),
            TableHeader.extend({
                addAttributes() {
                    return {
                        style: {
                            default: null,
                            parseHTML: element => element.getAttribute('style') || element.style.cssText,
                            renderHTML: attributes => {
                                if (!attributes.style) return {};
                                return { style: attributes.style };
                            },
                        },
                        width: {
                            default: null,
                            parseHTML: element => element.getAttribute('width') || element.style.width,
                            renderHTML: attributes => {
                                if (!attributes.width) return {};
                                return { width: attributes.width };
                            },
                        },
                        align: {
                            default: null,
                            parseHTML: element => element.getAttribute('align'),
                            renderHTML: attributes => {
                                if (!attributes.align) return {};
                                return { align: attributes.align };
                            },
                        }
                    };
                }
            }),
            TableCell.extend({
                addAttributes() {
                    return {
                        style: {
                            default: null,
                            parseHTML: element => element.getAttribute('style') || element.style.cssText,
                            renderHTML: attributes => {
                                if (!attributes.style) return {};
                                return { style: attributes.style };
                            },
                        },
                        width: {
                            default: null,
                            parseHTML: element => element.getAttribute('width') || element.style.width,
                            renderHTML: attributes => {
                                if (!attributes.width) return {};
                                return { width: attributes.width };
                            },
                        },
                        align: {
                            default: null,
                            parseHTML: element => element.getAttribute('align'),
                            renderHTML: attributes => {
                                if (!attributes.align) return {};
                                return { align: attributes.align };
                            },
                        }
                    };
                }
            }),
            Underline,
            TextAlign.configure({ types: ['heading', 'paragraph', 'div'] }),
        ],
        content: value,
        immediatelyRender: false,
        editorProps: {
            attributes: {
                class: 'focus:outline-none min-h-[350px] p-4 bg-white text-black',
            },
        },
        onUpdate: ({ editor }) => {
            onChange(editor.getHTML());
        },
    });

    useEffect(() => {
        if (editor && value) {
            // Only update if content is drastically different to prevent cursor jumps
            // A simple check: if editor is empty but value is not, set it.
            // Or if we are switching templates/modes.
            // For now, if editor content is completely different from value (e.g. loaded new template)
            // But editor.getHTML() might differ slightly from value due to parsing.
            // We'll leave this manual update for "initial load" scenarios mostly.

            // Actually, for "Preview Mode -> Edit Mode", the editor is remounted, so 'content: value' in useEditor handles it.
            // This useEffect handles updates while editor is alive.
            // Given the parent logic: switching mode unmounts editor? 
            // Yes: {isEditing ? <Editor> : <Preview>}
            // So we rely on `content: value` in useEditor.
            // We don't strictly need this useEffect for the current use case.
        }
    }, [editor, value]);

    return (
        <div className="border rounded-lg overflow-hidden flex flex-col bg-background shadow-sm">
            <Toolbar editor={editor} />
            <div className="flex-1 bg-white text-black overflow-y-auto max-h-[600px]">
                <EditorContent editor={editor} />
            </div>
        </div>
    );
}
