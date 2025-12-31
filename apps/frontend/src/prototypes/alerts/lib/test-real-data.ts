/**
 * 真实数据测试
 * 用于验证解析器能正确处理包含 Python 对象的数据
 */

export const realK8sWarningData = {
  "id": "c67815ae-132e-4cca-a3f5-f4cb4c820b8b",
  "title": "⚠️ Kubernetes Warning: <missing> (default/isolated-web-app-5668b988f.187160ed5c484253)",
  "description": "事件详情:\n- 类型: <missing>\n- 原因: <missing>\n- 对象: default/isolated-web-app-5668b988f.187160ed5c484253\n- 信息: <missing>\n",
  "enrichments": [
    {
      "blocks": [
        {
          "hidden": false,
          "html_class": null,
          "rows": [
            ["FailedCreate", "Warning", 1761295501000.0, "Error creating: pods \"isolated-web-app-5668b988f-tmq6x\" is forbidden: failed quota: test-quota: must specify limits.cpu for: web; limits.memory for: web; requests.cpu for: web; requests.memory for: web"],
            ["FailedCreate", "Warning", 1761294501000.0, "Error creating: pods \"isolated-web-app-5668b988f-tj66l\" is forbidden: failed quota: test-quota: must specify limits.cpu for: web; limits.memory for: web; requests.cpu for: web; requests.memory for: web"],
            ["FailedCreate", "Warning", 1761293501000.0, "Error creating: pods \"isolated-web-app-5668b988f-jwkb6\" is forbidden: failed quota: test-quota: must specify limits.cpu for: web; limits.memory for: web; requests.cpu for: web; requests.memory for: web"],
            ["FailedCreate", "Warning", 1761292501000.0, "Error creating: pods \"isolated-web-app-5668b988f-74pr7\" is forbidden: failed quota: test-quota: must specify limits.cpu for: web; limits.memory for: web; requests.cpu for: web; requests.memory for: web"]
          ],
          "headers": ["reason", "type", "time", "message"],
          "column_renderers": {
            "time": "DATETIME"
          },
          "table_name": "Related Events",
          "column_width": [1, 1, 1, 2],
          "metadata": {},
          "table_format": null,
          "events": [
            {
              "type": "Warning",
              "reason": "FailedCreate",
              "message": "Error creating: pods \"isolated-web-app-5668b988f-tmq6x\" is forbidden: failed quota: test-quota: must specify limits.cpu for: web; limits.memory for: web; requests.cpu for: web; requests.memory for: web",
              "kind": "replicaset",
              "name": "isolated-web-app-5668b988f",
              "namespace": "default",
              "time": "Oct 24, 2025, 08:45:01 AM"
            }
          ]
        }
      ],
      "annotations": {
        "attachment": true
      },
      "enrichment_type": {
        "_value_": "k8s_events",
        "_name_": "k8s_events"
      },
      "title": "Related Events"
    }
  ]
};

// 测试解析
if (typeof window !== 'undefined') {
  console.log('Testing real data parsing...');

  // 动态导入以避免服务端执行
  import('./enrichment-parser').then(({ parseFinding }) => {
    try {
      const parsed = parseFinding(realK8sWarningData);
      console.log('✅ Parsing successful!');
      console.log('Enrichments:', parsed.enrichments);
      console.log('First enrichment type:', parsed.enrichments?.[0]?.enrichment_type);
      console.log('First block:', parsed.enrichments?.[0]?.blocks?.[0]);
    } catch (error) {
      console.error('❌ Parsing failed:', error);
    }
  });
}
