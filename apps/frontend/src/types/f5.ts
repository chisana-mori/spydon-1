export interface F5Info {
  id: number;
  name: string;
  vip: string;
  port: string;
  appid: string;
  instance_group: string;
  status: string;
  pool_name: string;
  pool_status: string;
  pool_members: string;
  k8s_cluster_id?: number;  // 保持与后端一致
  cluster_name?: string;
  domains: string;
  grafana_params: string;
  ignored: boolean;
  created_at: string;
  updated_at: string;
}

export interface F5InfoQuery {
  page: number;
  size: number;
  name?: string;
  vip?: string;
  port?: string;
  appid?: string;
  instance_group?: string;
  status?: string;
  pool_name?: string;
  cluster_name?: string;
}

export interface F5InfoListResponse {
  list: F5Info[];
  page: number;
  size: number;
  total: number;
}

export interface F5InfoUpdateDTO {
  name: string;
  vip: string;
  port: string;
  appid: string;
  instance_group?: string;
  status?: string;
  pool_name?: string;
  pool_status?: string;
  pool_members?: string;
  k8s_cluster_id?: number;  // 保持与后端一致
  domains?: string;
  grafana_params?: string;
  ignored: boolean;
}
