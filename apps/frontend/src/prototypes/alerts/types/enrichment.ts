/**
 * Enrichment 数据类型定义
 * 基于设计文档 docs/rawplay.md
 */

export type EnrichmentType =
  | "graph"
  | "ai_analysis"
  | "node_info"
  | "container_info"
  | "k8s_events"
  | "alert_labels"
  | "diff"
  | "text_file"
  | "crash_info"
  | "image_pull_backoff_info"
  | "pending_pod_info"
  | string;

export interface Link {
  url: string;
  name?: string;
  type?: string;
}

export interface FindingSubject {
  name?: string;
  subject_type?: string;
  namespace?: string;
  node?: string;
  container?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

export interface BaseBlock {
  id?: string;
  type: string;
  raw?: any;
}

export interface MarkdownBlock extends BaseBlock {
  type: "markdown";
  text: string;
}

export interface HeaderBlock extends BaseBlock {
  type: "header";
  text: string;
  level?: number;
}

export interface JsonBlock extends BaseBlock {
  type: "json";
  data: any;
  json_str?: string;
}

export interface K8sDiffItem {
  path: string[];
  other_value?: any;
  value?: any;
  op?: "add" | "remove" | "replace";
}

export interface K8sDiffBlock extends BaseBlock {
  type: "k8s_diff" | "diff";
  diffs: K8sDiffItem[];
}

export interface ListBlock extends BaseBlock {
  type: "list";
  items: (string | any)[];
  ordered?: boolean;
}

export interface TableBlock extends BaseBlock {
  type: "table";
  headers: string[];
  rows: (string | number | any)[][];
  table_name?: string;
}

export interface LinksBlock extends BaseBlock {
  type: "links";
  links: Link[];
}

export interface TextFileBlock extends BaseBlock {
  type: "text_file" | "file";
  filename?: string;
  contents?: string;
}

export interface GraphBlock extends BaseBlock {
  type: "graph";
  filename?: string;
  contents?: string;
  hidden?: boolean;
  html_class?: string | null;
}

export interface GenericBlock extends BaseBlock {
  type: "generic";
  raw: any;
}

export type Block =
  | MarkdownBlock
  | HeaderBlock
  | JsonBlock
  | K8sDiffBlock
  | ListBlock
  | TableBlock
  | LinksBlock
  | TextFileBlock
  | GraphBlock
  | GenericBlock;

export interface Enrichment {
  title?: string;
  enrichment_type?: EnrichmentType;
  annotations?: Record<string, string>;
  blocks: Block[];
}

export interface Finding {
  id?: string;
  title?: string;
  description?: string;
  severity?: string;
  aggregation_key?: string;
  subject?: FindingSubject;
  links?: Link[];
  enrichments?: Enrichment[];
  fingerprint?: string;
  creation_date?: string;
  starts_at?: string;
  ends_at?: string;
  raw?: any;
}
