"""
Holmes Analyze Action for Robusta

This action triggers HolmesGPT analysis for alerts and sends results to Central Hub.
"""

import hashlib
import hmac
import json
import logging
import time
from typing import Dict, Any, Optional, List
from datetime import datetime, timedelta

import requests
from pydantic import BaseModel, Field
from robusta.api import (
    action,
    ActionParams,
    PrometheusKubernetesAlert,
    PodEvent,
    ExecutionBaseEvent,
    MarkdownBlock,
    FileBlock,
    TableBlock,
    RateLimiter,
)


class HolmesAnalyzeParams(ActionParams):
    """Parameters for Holmes analyze action"""
    
    central_hub_url: str = Field(..., description="Central Hub base URL")
    hmac_secret: str = Field(..., description="HMAC secret for request signing")
    cluster_id: str = Field(..., description="Cluster identifier")
    depth: str = Field(default="standard", description="Analysis depth: quick, standard, deep")
    timeout_seconds: int = Field(default=300, description="Analysis timeout in seconds")
    cooldown_minutes: int = Field(default=10, description="Cooldown between analyses for same alert")
    enabled: bool = Field(default=True, description="Enable/disable Holmes analysis")
    auto_trigger_severities: List[str] = Field(
        default=["critical", "high"], 
        description="Severities that auto-trigger analysis"
    )


@action
def holmes_analyze(event: ExecutionBaseEvent, params: HolmesAnalyzeParams):
    """
    Trigger HolmesGPT analysis for alerts
    
    This action can be triggered automatically by high-severity alerts or manually.
    It sends analysis requests to Central Hub which coordinates with HolmesGPT.
    """
    
    if not params.enabled:
        logging.info("Holmes analysis is disabled")
        return
    
    # Extract alert information
    alert_info = _extract_alert_info(event)
    if not alert_info:
        logging.warning("Could not extract alert information from event")
        return
    
    # Check rate limiting
    rate_limit_key = f"{alert_info['fingerprint']}:{params.cluster_id}"
    if not RateLimiter.mark_and_test(
        "holmes_analyze",
        rate_limit_key,
        params.cooldown_minutes * 60,
    ):
        logging.info(f"Holmes analysis rate limited for alert: {alert_info['title']}")
        return
    
    # Check if severity qualifies for auto-trigger
    if hasattr(event, 'prometheus_alert'):
        severity = _get_alert_severity(event.prometheus_alert)
        if severity not in params.auto_trigger_severities:
            logging.info(f"Alert severity '{severity}' does not qualify for auto-trigger")
            return
    
    try:
        # Send analysis request to Central Hub
        analysis_result = _trigger_holmes_analysis(alert_info, params)
        
        if analysis_result:
            # Add analysis result as enrichment
            _add_analysis_enrichment(event, analysis_result, alert_info)
            logging.info(f"Holmes analysis completed for alert: {alert_info['title']}")
        else:
            # Add pending analysis notification
            event.add_enrichment([
                MarkdownBlock("🔍 **Holmes Analysis Triggered**"),
                MarkdownBlock(f"Analysis depth: `{params.depth}`"),
                MarkdownBlock("Results will be available in Central Hub when complete."),
            ])
            logging.info(f"Holmes analysis triggered for alert: {alert_info['title']}")
            
    except Exception as e:
        logging.error(f"Holmes analysis failed: {e}")
        event.add_enrichment([
            MarkdownBlock("❌ **Holmes Analysis Failed**"),
            MarkdownBlock(f"Error: {str(e)}"),
        ])


def _extract_alert_info(event: ExecutionBaseEvent) -> Optional[Dict[str, Any]]:
    """Extract alert information from event"""
    
    alert_info = {
        "title": "Unknown Alert",
        "fingerprint": "unknown",
        "cluster_id": "",
        "labels": {},
        "annotations": {},
        "severity": "medium",
    }
    
    # Handle Prometheus alerts
    if hasattr(event, 'prometheus_alert') and event.prometheus_alert:
        alert = event.prometheus_alert
        alert_info.update({
            "title": alert.alert_name or "Prometheus Alert",
            "fingerprint": _generate_fingerprint(alert),
            "labels": alert.labels or {},
            "annotations": alert.annotations or {},
            "severity": _get_alert_severity(alert),
        })
    
    # Handle Pod events
    elif hasattr(event, 'get_pod') and event.get_pod():
        pod = event.get_pod()
        alert_info.update({
            "title": f"Pod Event: {pod.metadata.name}",
            "fingerprint": f"pod-{pod.metadata.namespace}-{pod.metadata.name}",
            "labels": {
                "pod": pod.metadata.name,
                "namespace": pod.metadata.namespace,
                **dict(pod.metadata.labels or {}),
            },
        })
    
    # Handle generic events
    else:
        # Try to extract basic information
        if hasattr(event, 'obj') and event.obj:
            obj = event.obj
            if hasattr(obj, 'metadata'):
                alert_info.update({
                    "title": f"Kubernetes Event: {obj.metadata.name}",
                    "fingerprint": f"k8s-{obj.metadata.namespace or 'default'}-{obj.metadata.name}",
                    "labels": dict(obj.metadata.labels or {}),
                })
    
    return alert_info


def _generate_fingerprint(prometheus_alert) -> str:
    """Generate fingerprint for Prometheus alert"""
    labels = prometheus_alert.labels or {}
    
    # Use standard Prometheus fingerprint labels
    fingerprint_labels = ["alertname", "instance", "job"]
    fingerprint_parts = []
    
    for label in fingerprint_labels:
        if label in labels:
            fingerprint_parts.append(f"{label}={labels[label]}")
    
    # Add other labels for uniqueness
    for key, value in sorted(labels.items()):
        if key not in fingerprint_labels:
            fingerprint_parts.append(f"{key}={value}")
    
    fingerprint_data = "|".join(fingerprint_parts)
    return hashlib.md5(fingerprint_data.encode()).hexdigest()


def _get_alert_severity(prometheus_alert) -> str:
    """Extract severity from Prometheus alert"""
    labels = prometheus_alert.labels or {}
    annotations = prometheus_alert.annotations or {}
    
    # Check common severity fields
    for field in ["severity", "priority", "level"]:
        if field in labels:
            return labels[field].lower()
        if field in annotations:
            return annotations[field].lower()
    
    # Default severity
    return "medium"


def _trigger_holmes_analysis(alert_info: Dict[str, Any], params: HolmesAnalyzeParams) -> Optional[Dict[str, Any]]:
    """Trigger Holmes analysis via Central Hub"""
    
    # Prepare analysis request
    analysis_request = {
        "alert_fingerprint": alert_info["fingerprint"],
        "cluster_id": params.cluster_id,
        "depth": params.depth,
        "timeout_seconds": params.timeout_seconds,
        "context": {
            "title": alert_info["title"],
            "labels": alert_info["labels"],
            "annotations": alert_info["annotations"],
            "severity": alert_info["severity"],
        }
    }
    
    # Send request to Central Hub
    url = f"{params.central_hub_url.rstrip('/')}/api/v1/rca/trigger"
    
    payload = json.dumps(analysis_request, sort_keys=True)
    timestamp = str(int(time.time()))
    
    # Generate HMAC signature
    signature_data = f"{timestamp}.{payload}"
    signature = hmac.new(
        params.hmac_secret.encode(),
        signature_data.encode(),
        hashlib.sha256
    ).hexdigest()
    
    headers = {
        "Content-Type": "application/json",
        "X-Robusta-Signature": f"sha256={signature}",
        "X-Robusta-Timestamp": timestamp,
        "X-Robusta-Cluster-ID": params.cluster_id,
        "User-Agent": "Robusta-Holmes-Action/1.0",
    }
    
    try:
        response = requests.post(
            url,
            data=payload,
            headers=headers,
            timeout=30  # Short timeout for trigger request
        )
        
        if response.status_code in [200, 201, 202]:
            result = response.json()
            logging.info(f"Holmes analysis triggered successfully: {result}")
            return result.get("data")
        else:
            raise Exception(f"Central Hub returned status {response.status_code}: {response.text}")
            
    except Exception as e:
        logging.error(f"Failed to trigger Holmes analysis: {e}")
        raise


def _add_analysis_enrichment(event: ExecutionBaseEvent, analysis_result: Dict[str, Any], alert_info: Dict[str, Any]):
    """Add Holmes analysis results as enrichment"""
    
    enrichments = [
        MarkdownBlock("🔍 **Holmes Analysis Results**"),
    ]
    
    # Add analysis summary
    if analysis_result.get("summary"):
        enrichments.append(MarkdownBlock(f"**Summary:** {analysis_result['summary']}"))
    
    # Add suspects
    if analysis_result.get("suspects"):
        suspects = analysis_result["suspects"]
        if isinstance(suspects, dict):
            suspect_rows = []
            for category, details in suspects.items():
                suspect_rows.append([category, str(details)])
            
            enrichments.append(MarkdownBlock("**Potential Root Causes:**"))
            enrichments.append(TableBlock(suspect_rows, ["Category", "Details"]))
    
    # Add recommendations
    if analysis_result.get("recommendations"):
        recommendations = analysis_result["recommendations"]
        if isinstance(recommendations, dict):
            rec_text = []
            for action, details in recommendations.items():
                rec_text.append(f"• **{action}**: {details}")
            
            enrichments.append(MarkdownBlock("**Recommendations:**"))
            enrichments.append(MarkdownBlock("\n".join(rec_text)))
    
    # Add analysis metadata
    metadata_rows = [
        ["Analysis ID", analysis_result.get("id", "N/A")],
        ["Depth", analysis_result.get("depth", "standard")],
        ["Status", analysis_result.get("status", "completed")],
    ]
    
    if analysis_result.get("started_at"):
        metadata_rows.append(["Started", analysis_result["started_at"]])
    if analysis_result.get("completed_at"):
        metadata_rows.append(["Completed", analysis_result["completed_at"]])
    
    enrichments.append(MarkdownBlock("**Analysis Details:**"))
    enrichments.append(TableBlock(metadata_rows, ["Field", "Value"]))
    
    # Add link to Central Hub
    central_hub_base = analysis_result.get("central_hub_url", "")
    if central_hub_base and analysis_result.get("id"):
        hub_link = f"{central_hub_base}/alerts/{alert_info['fingerprint']}"
        enrichments.append(MarkdownBlock(f"[View in Central Hub]({hub_link})"))
    
    event.add_enrichment(enrichments)
