import { api } from '@/lib/api';

// Types
export interface LabelManagement {
    id: number;
    name: string;
    key: string;
    source: number;
    status: number;
    values?: string[];
}

export interface TaintManagement {
    id: number;
    key: string;
    value: string;
    effect: string;
    description: string;
    type: string;
    status: number;
}

export interface DeviceApp {
    id: number;
    app_id: string;
    type: number;
    name: string;
    owner: string;
    feature: string;
    description: string;
    status: number;
}

// Labels API
export const getLabels = async (params?: any) => {
    const res = await api.get('/configuration/labels', { params });
    return res.data?.list || [];
};

export const createLabel = async (data: Partial<LabelManagement>) => {
    return (await api.post('/configuration/labels', data)).data;
};

export const updateLabel = async (id: number, data: Partial<LabelManagement>) => {
    return (await api.put(`/configuration/labels/${id}`, data)).data;
};

export const deleteLabel = async (id: number) => {
    return api.delete(`/configuration/labels/${id}`);
};

// Taints API
export const getTaints = async (params?: any) => {
    const res = await api.get('/configuration/taints', { params });
    return res.data?.list || [];
};

export const createTaint = async (data: Partial<TaintManagement>) => {
    return (await api.post('/configuration/taints', data)).data;
};

export const updateTaint = async (id: number, data: Partial<TaintManagement>) => {
    return (await api.put(`/configuration/taints/${id}`, data)).data;
};

export const deleteTaint = async (id: number) => {
    return api.delete(`/configuration/taints/${id}`);
};

// Device Apps API
export const getDeviceApps = async (params?: any) => {
    const res = await api.get('/configuration/device-apps', { params });
    return res.data?.list || [];
};

export const createDeviceApp = async (data: Partial<DeviceApp>) => {
    return (await api.post('/configuration/device-apps', data)).data;
};

export const updateDeviceApp = async (id: number, data: Partial<DeviceApp>) => {
    return (await api.put(`/configuration/device-apps/${id}`, data)).data;
};

export const deleteDeviceApp = async (id: number) => {
    return api.delete(`/configuration/device-apps/${id}`);
};
