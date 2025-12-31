/**
 * Enrichment 解析器
 * 将原始 payload 转换为结构化的 Enrichment 数据
 */

import {
  Block,
  Enrichment,
  Finding,
  MarkdownBlock,
  HeaderBlock,
  JsonBlock,
  K8sDiffBlock,
  ListBlock,
  TableBlock,
  LinksBlock,
  TextFileBlock,
  GenericBlock,
  KubernetesFieldsBlock,
  CallbackBlock,
  KubernetesFieldDescriptor,
  CallbackChoice,
} from '../types/enrichment';

type BlockParser = (raw: any) => Block;

class BlockParserRegistry {
  private parsers = new Map<string, BlockParser>();

  register(typeName: string, parser: BlockParser) {
    this.parsers.set(typeName, parser);
  }

  get(typeName: string): BlockParser | undefined {
    return this.parsers.get(typeName);
  }

  has(typeName: string): boolean {
    return this.parsers.has(typeName);
  }
}

const registry = new BlockParserRegistry();

// Markdown Block Parser
function parseMarkdown(raw: any): MarkdownBlock {
  return {
    id: raw.id || crypto.randomUUID(),
    type: 'markdown',
    text: raw.text || raw.markdown || '',
    raw,
  };
}

// Header Block Parser
function parseHeader(raw: any): HeaderBlock {
  return {
    id: raw.id || crypto.randomUUID(),
    type: 'header',
    text: raw.text || raw.header || '',
    level: raw.level || 3,
    raw,
  };
}

// JSON Block Parser
function parseJsonBlock(raw: any): JsonBlock {
  let data = raw.data || raw.json || raw;

  if (raw.json_str) {
    try {
      data = JSON.parse(raw.json_str);
    } catch (e) {
      data = raw.json_str;
    }
  }

  return {
    id: raw.id || crypto.randomUUID(),
    type: 'json',
    data,
    json_str: raw.json_str,
    raw,
  };
}

// K8s Diff Block Parser
function parseK8sDiff(raw: any): K8sDiffBlock {
  const diffs = raw.diffs || raw.changes || [];

  return {
    id: raw.id || crypto.randomUUID(),
    type: raw.type || 'k8s_diff',
    diffs: diffs.map((d: any) => ({
      path: d.path || [],
      other_value: d.other_value,
      value: d.value,
      op: d.op || 'replace',
    })),
    raw,
  };
}

// List Block Parser
function parseListBlock(raw: any): ListBlock {
  return {
    id: raw.id || crypto.randomUUID(),
    type: 'list',
    items: raw.items || [],
    ordered: raw.ordered || false,
    raw,
  };
}

// Table Block Parser
function parseTableBlock(raw: any): TableBlock {
  return {
    id: raw.id || crypto.randomUUID(),
    type: 'table',
    headers: raw.headers || [],
    rows: raw.rows || [],
    table_name: raw.table_name || raw.name,
    raw,
  };
}

// Links Block Parser
function parseLinksBlock(raw: any): LinksBlock {
  return {
    id: raw.id || crypto.randomUUID(),
    type: 'links',
    links: raw.links || [],
    raw,
  };
}

// Text File Block Parser
function parseTextFileBlock(raw: any): TextFileBlock {
  return {
    id: raw.id || crypto.randomUUID(),
    type: raw.type || 'text_file',
    filename: raw.filename || raw.name,
    contents: raw.contents || raw.content,
    raw,
  };
}

// Graph Block Parser
function parseGraphBlock(raw: any): Block {
  return {
    id: raw.id || crypto.randomUUID(),
    type: 'graph',
    filename: raw.filename || raw.name,
    contents: raw.contents || raw.content,
    raw,
  } as any;
}

// Kubernetes Fields Block Parser
function parseKubernetesFieldsBlock(raw: any): KubernetesFieldsBlock {
  const fieldsRaw = raw.fields || raw.field_paths || raw.field_list || raw.paths || [];
  const k8sObject = raw.k8s_obj || raw.kubernetes_object || raw.k8s_object || raw.object;
  const explanations = raw.explanations || raw.field_explanations || raw.metadata?.explanations;

  const normalizeField = (entry: any): KubernetesFieldDescriptor => {
    if (!entry) {
      return { path: '', label: '' };
    }

    if (typeof entry === 'string') {
      return {
        path: entry,
        label: entry,
      };
    }

    if (Array.isArray(entry)) {
      return {
        path: entry.join('.'),
        label: entry.join('.'),
      };
    }

    if (typeof entry === 'object') {
      const path =
        entry.path ||
        entry.field_path ||
        entry.key ||
        (Array.isArray(entry.field) ? entry.field.join('.') : entry.field) ||
        entry.name ||
        '';

      return {
        path,
        label: entry.label || entry.display_name || entry.title || entry.field_name || path,
        value: entry.value,
      };
    }

    return {
      path: String(entry),
      label: String(entry),
    };
  };

  let fields: KubernetesFieldDescriptor[] = [];

  if (Array.isArray(fieldsRaw)) {
    fields = fieldsRaw.map(normalizeField).filter((f) => f.path);
  } else if (typeof fieldsRaw === 'object' && fieldsRaw !== null) {
    fields = Object.entries(fieldsRaw).map(([key, value]) => ({
      path: key,
      label: key,
      value,
    }));
  }

  if (fields.length === 0 && raw.field_path && raw.value !== undefined) {
    fields = [
      {
        path: raw.field_path,
        label: raw.field_path,
        value: raw.value,
      },
    ];
  }

  return {
    id: raw.id || crypto.randomUUID(),
    type: raw.type || 'kubernetes_fields',
    title: raw.title || raw.name,
    fields,
    k8s_obj: k8sObject,
    explanations,
    raw,
  };
}

// Callback Block Parser
function parseCallbackBlock(raw: any): CallbackBlock {
  const meta = raw.metadata || raw.meta || {};
  const choicesRaw = raw.choices || meta.choices || meta.options || raw.options || meta.buttons || {};

  const normalizeChoice = (entryKey: string | number, entryValue: any): CallbackChoice => {
    if (typeof entryValue === 'string') {
      return {
        label: entryKey ? String(entryKey) : entryValue,
        value: entryValue,
      };
    }

    if (typeof entryValue === 'object' && entryValue !== null) {
      return {
        label: entryValue.label || entryValue.name || entryValue.title || String(entryKey),
        value: entryValue.value || entryValue.action || entryValue.id || String(entryKey),
        description: entryValue.description || entryValue.help_text,
        danger: Boolean(entryValue.danger || entryValue.is_dangerous),
      };
    }

    return {
      label: String(entryKey),
      value: String(entryValue),
    };
  };

  let choices: CallbackChoice[] = [];

  if (Array.isArray(choicesRaw)) {
    choices = choicesRaw.map((item, index) => {
      if (typeof item === 'string') {
        return {
          label: item,
          value: item,
        };
      }
      if (typeof item === 'object' && item !== null) {
        return normalizeChoice(index, item);
      }
      return {
        label: String(item),
        value: String(item),
      };
    });
  } else if (typeof choicesRaw === 'object' && choicesRaw !== null) {
    choices = Object.entries(choicesRaw).map(([key, value]) => normalizeChoice(key, value));
  }

  const rawVerb = (raw.verb || meta.verb || meta.method || 'POST')?.toString().toUpperCase();
  const allowedVerbs: Array<NonNullable<CallbackBlock['verb']>> = ['GET', 'POST', 'PUT', 'DELETE'];
  const normalizedVerb = allowedVerbs.includes(rawVerb as any) ? (rawVerb as CallbackBlock['verb']) : 'POST';

  return {
    id: raw.id || crypto.randomUUID(),
    type: 'callback',
    title: raw.title || raw.name || meta.title,
    description: raw.description || meta.description || meta.prompt,
    callback_url: raw.callback_url || meta.callback_url || meta.webhook_url || raw.webhook_url,
    webhook_url: meta.webhook_url || raw.webhook_url || raw.endpoint,
    verb: normalizedVerb,
    choices,
    extra: {
      ...meta,
      ...(raw.extra || {}),
    },
    raw,
  };
}

// Generic Block Parser
function parseGenericBlock(raw: any): GenericBlock {
  return {
    id: raw.id || crypto.randomUUID(),
    type: 'generic',
    raw,
  };
}

/**
 * 清理 Python 对象表示，提取实际值
 */
function cleanPythonObject(obj: any): any {
  if (obj === null || obj === undefined) {
    return obj;
  }

  // 如果是 Python Enum 对象，提取 _value_
  if (typeof obj === 'object' && '_value_' in obj) {
    return obj._value_;
  }

  // 如果是数组，递归清理每个元素
  if (Array.isArray(obj)) {
    return obj.map(cleanPythonObject);
  }

  // 如果是对象，递归清理每个属性
  if (typeof obj === 'object') {
    const cleaned: any = {};
    for (const key in obj) {
      // 跳过 Python 内部属性
      if (key.startsWith('_') && key.endsWith('_')) {
        continue;
      }
      cleaned[key] = cleanPythonObject(obj[key]);
    }
    return cleaned;
  }

  return obj;
}

// 注册默认解析器
registry.register('markdown', parseMarkdown);
registry.register('header', parseHeader);
registry.register('json', parseJsonBlock);
registry.register('k8s_diff', parseK8sDiff);
registry.register('diff', parseK8sDiff);
registry.register('list', parseListBlock);
registry.register('table', parseTableBlock);
registry.register('links', parseLinksBlock);
registry.register('text_file', parseTextFileBlock);
registry.register('file', parseTextFileBlock);
registry.register('graph', parseGraphBlock);
registry.register('kubernetes_fields', parseKubernetesFieldsBlock);
registry.register('kubernetes_fields_block', parseKubernetesFieldsBlock);
registry.register('fields', parseKubernetesFieldsBlock);
registry.register('callback', parseCallbackBlock);

/**
 * 解析单个 Block
 */
export function parseBlock(raw: any): Block {
  if (!raw) return parseGenericBlock(raw);

  // 清理 Python 对象
  const cleaned = cleanPythonObject(raw);
  const type = cleaned?.type;

  // 使用注册的解析器
  if (type && registry.has(type)) {
    return registry.get(type)!(cleaned);
  }

  // 启发式识别
  if (cleaned.diffs || cleaned.changes) {
    return parseK8sDiff(cleaned);
  }
  if (cleaned.data || cleaned.json || cleaned.json_str) {
    return parseJsonBlock(cleaned);
  }
  if (cleaned.items && Array.isArray(cleaned.items)) {
    return parseListBlock(cleaned);
  }
  if (cleaned.headers && cleaned.rows) {
    return parseTableBlock(cleaned);
  }
  if (cleaned.links && Array.isArray(cleaned.links)) {
    return parseLinksBlock(cleaned);
  }
  if (cleaned.text && !cleaned.headers) {
    return parseMarkdown(cleaned);
  }
  if (cleaned.fields && Array.isArray(cleaned.fields)) {
    return parseKubernetesFieldsBlock(cleaned);
  }
  if (cleaned.metadata && cleaned.metadata.choices) {
    return parseCallbackBlock(cleaned);
  }
  if (cleaned.filename || cleaned.contents) {
    return parseTextFileBlock(cleaned);
  }

  // 未知类型
  return parseGenericBlock(cleaned);
}

/**
 * 解析 Enrichment
 */
export function parseEnrichment(raw: any): Enrichment {
  // 清理 Python 对象
  const cleaned = cleanPythonObject(raw);

  console.log('[parseEnrichment] Raw:', raw);
  console.log('[parseEnrichment] Cleaned:', cleaned);
  console.log('[parseEnrichment] Blocks:', cleaned.blocks);

  const blocks = (cleaned.blocks || []).map(parseBlock);

  console.log('[parseEnrichment] Parsed blocks:', blocks);

  // 提取 enrichment_type 的实际值
  let enrichmentType = cleaned.enrichment_type || cleaned.type;
  if (typeof enrichmentType === 'object' && enrichmentType._value_) {
    enrichmentType = enrichmentType._value_;
  }

  const result = {
    title: cleaned.title,
    enrichment_type: enrichmentType,
    annotations: cleaned.annotations,
    blocks,
  };

  console.log('[parseEnrichment] Result:', result);

  return result;
}

/**
 * 解析 Finding
 */
export function parseFinding(raw: any): Finding {
  // 清理 Python 对象
  const cleaned = cleanPythonObject(raw);

  const enrichments = (cleaned.enrichments || []).map(parseEnrichment);

  return {
    id: cleaned.id,
    title: cleaned.title,
    description: cleaned.description,
    severity: cleaned.severity,
    aggregation_key: cleaned.aggregation_key,
    subject: cleaned.subject,
    links: cleaned.links,
    enrichments,
    fingerprint: cleaned.fingerprint,
    creation_date: cleaned.creation_date,
    starts_at: cleaned.starts_at,
    ends_at: cleaned.ends_at,
    raw: cleaned,
  };
}

/**
 * 注册自定义 Block 解析器
 */
export function registerBlockParser(typeName: string, parser: BlockParser) {
  registry.register(typeName, parser);
}
