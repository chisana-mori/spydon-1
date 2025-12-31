/**
 * 示例 Enrichment 数据
 * 用于测试和开发
 */

export const sampleFinding = {
  id: "finding-123",
  title: "Pod CrashLoopBackOff Detected",
  description: "A pod in the production namespace is experiencing repeated crashes",
  severity: "high",
  enrichments: [
    {
      title: "AI Analysis",
      enrichment_type: "ai_analysis",
      annotations: {
        confidence: "high",
        model: "gpt-4"
      },
      blocks: [
        {
          type: "markdown",
          text: "The pod is crashing due to an out-of-memory error. The container is requesting 512Mi but the actual usage peaks at 800Mi during startup."
        },
        {
          type: "header",
          text: "Recommended Actions",
          level: 3
        },
        {
          type: "list",
          items: [
            "Increase memory limit to 1Gi",
            "Add memory request of 768Mi",
            "Review application memory usage patterns",
            "Consider implementing memory profiling"
          ]
        }
      ]
    },
    {
      title: "Container Information",
      enrichment_type: "container_info",
      blocks: [
        {
          type: "table",
          table_name: "Container Resources",
          headers: ["Resource", "Request", "Limit", "Current Usage"],
          rows: [
            ["CPU", "100m", "500m", "250m"],
            ["Memory", "512Mi", "512Mi", "800Mi"],
            ["Ephemeral Storage", "1Gi", "2Gi", "500Mi"]
          ]
        },
        {
          type: "json",
          data: {
            name: "api-server",
            image: "myapp/api:v1.2.3",
            restartCount: 15,
            state: "waiting",
            lastState: {
              terminated: {
                exitCode: 137,
                reason: "OOMKilled",
                startedAt: "2024-01-15T10:30:00Z",
                finishedAt: "2024-01-15T10:31:00Z"
              }
            }
          }
        }
      ]
    },
    {
      title: "Configuration Changes",
      enrichment_type: "diff",
      blocks: [
        {
          type: "k8s_diff",
          diffs: [
            {
              path: ["spec", "containers", "0", "resources", "limits", "memory"],
              other_value: "512Mi",
              value: "1Gi",
              op: "replace"
            },
            {
              path: ["spec", "containers", "0", "resources", "requests", "memory"],
              other_value: "256Mi",
              value: "768Mi",
              op: "replace"
            },
            {
              path: ["spec", "containers", "0", "env"],
              value: {
                name: "JAVA_OPTS",
                value: "-Xmx768m"
              },
              op: "add"
            }
          ]
        }
      ]
    },
    {
      title: "Recent Events",
      enrichment_type: "k8s_events",
      blocks: [
        {
          type: "table",
          headers: ["Time", "Type", "Reason", "Message"],
          rows: [
            ["2m ago", "Warning", "BackOff", "Back-off restarting failed container"],
            ["3m ago", "Warning", "Failed", "Error: OOMKilled"],
            ["5m ago", "Normal", "Pulled", "Container image already present on machine"],
            ["5m ago", "Normal", "Created", "Created container api-server"],
            ["5m ago", "Normal", "Started", "Started container api-server"]
          ]
        }
      ]
    },
    {
      title: "Pod Logs",
      enrichment_type: "text_file",
      blocks: [
        {
          type: "text_file",
          filename: "api-server.log",
          contents: `2024-01-15 10:30:15 INFO  Starting application...
2024-01-15 10:30:20 INFO  Loading configuration from /etc/config
2024-01-15 10:30:25 INFO  Connecting to database...
2024-01-15 10:30:30 INFO  Database connection established
2024-01-15 10:30:35 WARN  High memory usage detected: 650Mi
2024-01-15 10:30:40 ERROR Out of memory error
2024-01-15 10:30:40 ERROR java.lang.OutOfMemoryError: Java heap space
2024-01-15 10:30:40 ERROR   at com.example.api.DataProcessor.process(DataProcessor.java:45)
2024-01-15 10:30:40 ERROR   at com.example.api.Controller.handleRequest(Controller.java:123)
2024-01-15 10:30:41 FATAL Application terminated`
        }
      ]
    },
    {
      title: "Related Resources",
      enrichment_type: "links",
      blocks: [
        {
          type: "links",
          links: [
            {
              url: "https://prometheus.example.com/graph?g0.expr=container_memory_usage_bytes",
              name: "Memory Usage Graph",
              type: "prometheus"
            },
            {
              url: "https://grafana.example.com/d/pod-dashboard",
              name: "Pod Dashboard",
              type: "grafana"
            },
            {
              url: "https://docs.example.com/troubleshooting/oom",
              name: "OOM Troubleshooting Guide",
              type: "documentation"
            }
          ]
        }
      ]
    }
  ]
};

export const sampleEnrichmentWithUnknownBlock = {
  title: "Test Enrichment",
  enrichment_type: "test",
  blocks: [
    {
      type: "unknown_type",
      someField: "some value",
      nestedData: {
        key: "value"
      }
    }
  ]
};
