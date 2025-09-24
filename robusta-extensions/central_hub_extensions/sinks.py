"""
Central Hub Webhook Sink for Robusta

This sink sends enriched alerts to the Central Hub for centralized management.
"""

import hashlib
import hmac
import json
import logging
import time
from typing import Dict, Any, Optional

import requests
from pydantic import BaseModel, Field
from robusta.api import (
    SinkBaseParams,
    SinkConfigBase,
    SinkBase,
    Finding,
    FindingSource,
    ExecutionBaseEvent,
    RobustaSink,
)


class CentralWebhookSinkParams(SinkBaseParams):
    """Parameters for Central Hub webhook sink"""
    
    url: str = Field(..., description="Central Hub webhook URL")
    cluster_id: str = Field(..., description="Unique identifier for this cluster")
    hmac_secret: str = Field(..., description="HMAC secret for request signing")
    timeout: int = Field(default=30, description="Request timeout in seconds")
    retry_attempts: int = Field(default=3, description="Number of retry attempts")
    batch_size: int = Field(default=10, description="Maximum alerts per batch")


class CentralWebhookSinkConfigWrapper(SinkConfigBase):
    """Configuration wrapper for Central Hub webhook sink"""
    
    central_webhook: CentralWebhookSinkParams

    def get_params(self) -> SinkBaseParams:
        return self.central_webhook


@RobustaSink
class CentralWebhookSink(SinkBase):
    """
    Central Hub webhook sink implementation
    
    Sends enriched alerts to Central Hub with HMAC signature verification.
    """

    def __init__(self, sink_config: CentralWebhookSinkConfigWrapper, registry):
        super().__init__(sink_config.central_webhook, registry)
        self.params: CentralWebhookSinkParams = sink_config.central_webhook
        self.session = requests.Session()
        
        # Configure session with retries
        from requests.adapters import HTTPAdapter
        from urllib3.util.retry import Retry
        
        retry_strategy = Retry(
            total=self.params.retry_attempts,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
        )
        adapter = HTTPAdapter(max_retries=retry_strategy)
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)

    def write_finding(self, finding: Finding, platform_enabled: bool) -> None:
        """Send a single finding to Central Hub"""
        try:
            # Convert finding to Central Hub format
            alert_data = self._convert_finding_to_alert(finding)
            
            # Send to Central Hub
            self._send_to_central_hub(alert_data)
            
            logging.info(f"Successfully sent alert to Central Hub: {finding.title}")
            
        except Exception as e:
            logging.error(f"Failed to send alert to Central Hub: {e}")
            # Don't raise exception to avoid breaking other sinks

    def _convert_finding_to_alert(self, finding: Finding) -> Dict[str, Any]:
        """Convert Robusta Finding to Central Hub alert format"""
        
        # Extract labels and annotations from finding
        labels = {}
        annotations = {}
        
        # Get source information
        source = finding.source
        if source and hasattr(source, 'prometheus_alert'):
            prometheus_alert = source.prometheus_alert
            if prometheus_alert:
                labels.update(prometheus_alert.labels or {})
                annotations.update(prometheus_alert.annotations or {})
        
        # Add Robusta-specific labels
        labels.update({
            "robusta_cluster": self.params.cluster_id,
            "robusta_source": "robusta",
            "robusta_finding_id": finding.id,
        })
        
        # Add enrichment data to annotations
        if finding.enrichments:
            enrichment_summary = []
            for enrichment in finding.enrichments:
                if hasattr(enrichment, 'text'):
                    enrichment_summary.append(enrichment.text)
                elif hasattr(enrichment, 'content'):
                    enrichment_summary.append(str(enrichment.content))
            
            if enrichment_summary:
                annotations["robusta_enrichments"] = "\n".join(enrichment_summary)
        
        # Generate fingerprint for deduplication
        fingerprint_data = f"{finding.title}:{self.params.cluster_id}:{finding.aggregation_key or ''}"
        fingerprint = hashlib.md5(fingerprint_data.encode()).hexdigest()
        
        # Determine severity mapping
        severity_mapping = {
            "INFO": "low",
            "LOW": "low", 
            "MEDIUM": "medium",
            "HIGH": "high",
            "CRITICAL": "critical"
        }
        severity = severity_mapping.get(finding.severity.name, "medium")
        
        # Build alert payload
        alert_payload = {
            "fingerprint": fingerprint,
            "cluster_id": self.params.cluster_id,
            "title": finding.title,
            "description": finding.description or "",
            "severity": severity,
            "status": "firing",  # Robusta findings are always firing
            "labels": labels,
            "annotations": annotations,
            "starts_at": finding.creation_time.isoformat() if finding.creation_time else None,
        }
        
        return alert_payload

    def _send_to_central_hub(self, alert_data: Dict[str, Any]) -> None:
        """Send alert data to Central Hub with HMAC signature"""
        
        # Prepare request
        payload = json.dumps(alert_data, sort_keys=True)
        timestamp = str(int(time.time()))
        
        # Generate HMAC signature
        signature_data = f"{timestamp}.{payload}"
        signature = hmac.new(
            self.params.hmac_secret.encode(),
            signature_data.encode(),
            hashlib.sha256
        ).hexdigest()
        
        headers = {
            "Content-Type": "application/json",
            "X-Robusta-Signature": f"sha256={signature}",
            "X-Robusta-Timestamp": timestamp,
            "X-Robusta-Cluster-ID": self.params.cluster_id,
            "User-Agent": "Robusta-Central-Hub-Sink/1.0",
        }
        
        # Send request
        response = self.session.post(
            self.params.url,
            data=payload,
            headers=headers,
            timeout=self.params.timeout
        )
        
        # Check response
        if response.status_code not in [200, 201, 202]:
            raise Exception(
                f"Central Hub returned status {response.status_code}: {response.text}"
            )
        
        logging.debug(f"Central Hub response: {response.status_code} - {response.text}")

    def set_cluster_active(self, active: bool):
        """Send cluster heartbeat to Central Hub"""
        try:
            heartbeat_data = {
                "cluster_id": self.params.cluster_id,
                "name": self.params.cluster_id,  # Can be overridden in config
                "status": "active" if active else "inactive",
                "description": f"Robusta cluster {self.params.cluster_id}",
            }
            
            # Use heartbeat endpoint
            heartbeat_url = self.params.url.replace("/ingest/alert", "/clusters/heartbeat")
            
            payload = json.dumps(heartbeat_data, sort_keys=True)
            timestamp = str(int(time.time()))
            
            # Generate HMAC signature
            signature_data = f"{timestamp}.{payload}"
            signature = hmac.new(
                self.params.hmac_secret.encode(),
                signature_data.encode(),
                hashlib.sha256
            ).hexdigest()
            
            headers = {
                "Content-Type": "application/json",
                "X-Robusta-Signature": f"sha256={signature}",
                "X-Robusta-Timestamp": timestamp,
                "X-Robusta-Cluster-ID": self.params.cluster_id,
                "User-Agent": "Robusta-Central-Hub-Sink/1.0",
            }
            
            response = self.session.post(
                heartbeat_url,
                data=payload,
                headers=headers,
                timeout=self.params.timeout
            )
            
            if response.status_code in [200, 201, 202]:
                logging.info(f"Cluster heartbeat sent successfully: {active}")
            else:
                logging.warning(f"Cluster heartbeat failed: {response.status_code}")
                
        except Exception as e:
            logging.error(f"Failed to send cluster heartbeat: {e}")
