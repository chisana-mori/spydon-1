package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// no direct dependency on pkg redis types here
	goredis "github.com/redis/go-redis/v9"
)

// DrainEventType 表示 Drain 事件类型
type DrainEventType string

const (
	DrainStarted   DrainEventType = "started"
	DrainProgress  DrainEventType = "progress"
	DrainCompleted DrainEventType = "completed"
	DrainFailed    DrainEventType = "failed"
	DrainCanceled  DrainEventType = "canceled"
	DrainStatsEvt  DrainEventType = "stats"

	PodSnapshotCreated    DrainEventType = "pod_snapshot_created"
	PodEvictionStarted    DrainEventType = "pod_eviction_started"
	PodEvictionSucceeded  DrainEventType = "pod_eviction_succeeded"
	PodEvictionFailed     DrainEventType = "pod_eviction_failed"
	NewPodDetected        DrainEventType = "new_pod_detected"
	PodMigrationCompleted DrainEventType = "pod_migration_completed"
	PodMigrationFailed    DrainEventType = "pod_migration_failed"
	PodMigrationIgnored   DrainEventType = "pod_migration_ignored"
	PodPending            DrainEventType = "pod_pending"
	PodPhaseUpdated       DrainEventType = "pod_phase_updated"

	PdbEvent DrainEventType = "pdb_event"

	DrainStreamClosing DrainEventType = "stream_closing"
)

const (
	redisHandshakeTimeout = 200 * time.Millisecond
	redisInitialBackoff   = 200 * time.Millisecond
	redisMaxBackoff       = 5 * time.Second
	sseDedupWindow        = 500 * time.Millisecond
)

// DrainMessage 为简化的事件消息结构
type DrainMessage struct {
	Type      DrainEventType         `json:"type"`
	DrainID   string                 `json:"drainId"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// DrainPodMigrationStatus 表示 Pod 迁移状态
type DrainPodMigrationStatus string

const (
	DrainMigrationPending   DrainPodMigrationStatus = "pending"
	DrainMigrationEvicting  DrainPodMigrationStatus = "evicting"
	DrainMigrationEvicted   DrainPodMigrationStatus = "evicted"
	DrainMigrationCreating  DrainPodMigrationStatus = "creating"
	DrainMigrationCompleted DrainPodMigrationStatus = "completed"
	DrainMigrationFailed    DrainPodMigrationStatus = "failed"
	DrainMigrationTimeout   DrainPodMigrationStatus = "timeout"
	DrainMigrationIgnored   DrainPodMigrationStatus = "ignored"
)

// DrainPodInfo 为 Pod 基本信息
type DrainPodInfo struct {
	Name            string                  `json:"name"`
	Namespace       string                  `json:"namespace"`
	NodeName        string                  `json:"nodeName"`
	Labels          map[string]string       `json:"labels"`
	Annotations     map[string]string       `json:"annotations"`
	UID             string                  `json:"uid"`
	Phase           string                  `json:"phase"`
	PhaseReason     string                  `json:"phaseReason,omitempty"`
	Status          string                  `json:"status,omitempty"`
	Ready           string                  `json:"ready,omitempty"`
	RestartCount    int32                   `json:"restartCount,omitempty"`
	Age             string                  `json:"age,omitempty"`
	CreatedAt       time.Time               `json:"createdAt"`
	OwnerReferences []metav1.OwnerReference `json:"ownerReferences"`
}

// DrainPodMigrationInfo 为 Pod 迁移跟踪信息
type DrainPodMigrationInfo struct {
	DrainID        string                  `json:"drainId"`
	SourcePod      DrainPodInfo            `json:"sourcePod"`
	TargetPod      *DrainPodInfo           `json:"targetPod,omitempty"`
	Status         DrainPodMigrationStatus `json:"status"`
	Progress       int                     `json:"progress"`
	StartTime      time.Time               `json:"startTime"`
	EvictionTime   *time.Time              `json:"evictionTime,omitempty"`
	CompletionTime *time.Time              `json:"completionTime,omitempty"`
	Duration       time.Duration           `json:"duration"`
	ErrorMessage   string                  `json:"errorMessage,omitempty"`
	OwnerReference string                  `json:"ownerReference"`
	MigrationID    string                  `json:"migrationId"`
}

// DrainSnapshotInfo 为 Drain 快照信息
type DrainSnapshotInfo struct {
	DrainID     string         `json:"drainId"`
	NodeName    string         `json:"nodeName"`
	ClusterName string         `json:"clusterName"`
	TotalPods   int            `json:"totalPods"`
	CreatedAt   time.Time      `json:"createdAt"`
	Pods        []DrainPodInfo `json:"pods"`
}

// SimpleDrainEventManager 负责事件订阅与发布
type SimpleDrainEventManager struct {
	subscribers       map[string][]chan DrainMessage
	connectionCancels map[string][]connectionCancel
	connSeq           uint64
	mutex             sync.RWMutex
	redisHandler      sharedservices.RedisClient

	trackerMutex   sync.RWMutex
	messageTracker map[string]time.Time
}

type connectionCancel struct {
	id     string
	cancel context.CancelFunc
	notify chan string
}

// NewSimpleDrainEventManager 创建事件管理器
func NewSimpleDrainEventManager(redisHandler sharedservices.RedisClient) *SimpleDrainEventManager {
	return &SimpleDrainEventManager{
		subscribers:       make(map[string][]chan DrainMessage),
		connectionCancels: make(map[string][]connectionCancel),
		redisHandler:      redisHandler,
		messageTracker:    make(map[string]time.Time),
	}
}

func (sem *SimpleDrainEventManager) eventDedupWindow(eventType DrainEventType) time.Duration {
	switch eventType {
	case DrainProgress, DrainStatsEvt, DrainStarted, DrainCompleted, DrainFailed, DrainCanceled, DrainStreamClosing:
		return 0
	case PodPhaseUpdated:
		return 1500 * time.Millisecond
	case NewPodDetected:
		return 10 * time.Second
	case PodEvictionStarted, PodEvictionSucceeded, PodEvictionFailed,
		PodMigrationCompleted, PodMigrationFailed, PodMigrationIgnored,
		PodSnapshotCreated, PodPending, PdbEvent:
		return 2 * time.Second
	default:
		// 针对通用日志或其他事件，设置适度窗口避免连续重复
		return 2 * time.Second
	}
}

func (sem *SimpleDrainEventManager) shouldSkipEvent(drainID string, msgType DrainEventType, message string) bool {
	window := sem.eventDedupWindow(msgType)
	if window <= 0 {
		return false
	}

	key := fmt.Sprintf("%s:%s:%s", drainID, msgType, message)
	now := time.Now()

	sem.trackerMutex.Lock()
	defer sem.trackerMutex.Unlock()

	if last, exists := sem.messageTracker[key]; exists {
		if now.Sub(last) < window {
			return true
		}
	}

	sem.messageTracker[key] = now
	return false
}

func (sem *SimpleDrainEventManager) registerConnection(drainID string, cancel context.CancelFunc) (string, <-chan string) {
	sem.mutex.Lock()
	sem.connSeq++
	id := fmt.Sprintf("%s-%d", drainID, sem.connSeq)
	notify := make(chan string, 1)
	sem.connectionCancels[drainID] = append(sem.connectionCancels[drainID], connectionCancel{id: id, cancel: cancel, notify: notify})
	sem.mutex.Unlock()
	return id, notify
}

func (sem *SimpleDrainEventManager) unregisterConnection(drainID, id string) {
	sem.mutex.Lock()
	if cancels, ok := sem.connectionCancels[drainID]; ok {
		for i, holder := range cancels {
			if holder.id == id {
				sem.connectionCancels[drainID] = append(cancels[:i], cancels[i+1:]...)
				break
			}
		}
		if len(sem.connectionCancels[drainID]) == 0 {
			delete(sem.connectionCancels, drainID)
		}
	}
	sem.mutex.Unlock()
}

// CloseStreams 主动关闭指定 drain 的 SSE 连接
func (sem *SimpleDrainEventManager) CloseStreams(drainID string, reason string) {
	sem.SendMessage(drainID, DrainStreamClosing, fmt.Sprintf("事件流即将关闭: %s", reason), map[string]interface{}{"reason": reason})
	sem.mutex.Lock()
	cancels := sem.connectionCancels[drainID]
	delete(sem.connectionCancels, drainID)
	sem.mutex.Unlock()
	for _, holder := range cancels {
		select {
		case holder.notify <- reason:
		default:
		}
		if holder.cancel != nil {
			time.AfterFunc(200*time.Millisecond, holder.cancel)
		}
	}
}

func (sem *SimpleDrainEventManager) channelName(drainID string) string {
	return fmt.Sprintf("drain:events:%s", drainID)
}

// Subscribe 订阅指定 DrainID 的事件
func (sem *SimpleDrainEventManager) Subscribe(drainID string) chan DrainMessage {
	sem.mutex.Lock()
	defer sem.mutex.Unlock()

	ch := make(chan DrainMessage, 100)
	sem.subscribers[drainID] = append(sem.subscribers[drainID], ch)
	return ch
}

// Unsubscribe 取消订阅指定 DrainID 的事件
func (sem *SimpleDrainEventManager) Unsubscribe(drainID string, target chan DrainMessage) {
	if target == nil {
		return
	}
	sem.mutex.Lock()
	defer sem.mutex.Unlock()

	if channels, exists := sem.subscribers[drainID]; exists {
		for i, ch := range channels {
			if ch == target {
				sem.subscribers[drainID] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		if len(sem.subscribers[drainID]) == 0 {
			delete(sem.subscribers, drainID)
		}
	}
}

// SendMessage 发送通用事件消息
func (sem *SimpleDrainEventManager) SendMessage(drainID string, msgType DrainEventType, message string, data map[string]interface{}) {
	if sem.shouldSkipEvent(drainID, msgType, message) {
		return
	}

	msg := DrainMessage{
		Type:      msgType,
		DrainID:   drainID,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
	// Prefer Redis Pub/Sub for cross-instance delivery
	if sem.redisHandler != nil {
		if payload, err := json.Marshal(msg); err == nil {
			_ = sem.redisHandler.Pub(sem.channelName(drainID), string(payload))
		}
	}
	// Also deliver to local subscribers (same-instance clients), best-effort
	sem.mutex.RLock()
	if channels, exists := sem.subscribers[drainID]; exists {
		for _, ch := range channels {
			select {
			case ch <- msg:
			default:
			}
		}
	}
	sem.mutex.RUnlock()
}

// SendProgress 发送进度消息
func (sem *SimpleDrainEventManager) SendProgress(drainID, step, message string, progress int) {
	data := map[string]interface{}{
		"step":     step,
		"progress": progress,
		"level":    "INFO",
		"category": "general",
	}
	sem.SendMessage(drainID, DrainProgress, message, data)
}

// newPodEventData 构造标准化 Pod 事件数据
func newPodEventData(step, podName, podNamespace string, progress int, level, category string, err error) map[string]interface{} {
	data := map[string]interface{}{
		"step":         step,
		"progress":     progress,
		"level":        level,
		"category":     category,
		"podName":      podName,
		"podNamespace": podNamespace,
		"resource": map[string]interface{}{
			"kind":      "Pod",
			"namespace": podNamespace,
			"name":      podName,
		},
	}
	if err != nil {
		data["error"] = err.Error()
	}
	return data
}

// SendProgressWithPod 发送包含 Pod 信息的进度消息
func (sem *SimpleDrainEventManager) SendProgressWithPod(drainID, step, message string, progress int, podName, podNamespace string) {
	data := newPodEventData(step, podName, podNamespace, progress, "INFO", "pod", nil)
	if step == "pdb" {
		data["kind"] = "pdb"
		data["category"] = "pdb"
	}
	sem.SendMessage(drainID, DrainProgress, message, data)
}

// SendError 发送错误消息
func (sem *SimpleDrainEventManager) SendError(drainID, message string, err error) {
	data := map[string]interface{}{
		"error":    err.Error(),
		"level":    "ERROR",
		"category": "error",
	}
	sem.SendMessage(drainID, DrainFailed, message, data)
}

// SendErrorWithPod 发送包含 Pod 信息的错误消息
func (sem *SimpleDrainEventManager) SendErrorWithPod(drainID, message string, err error, podName, podNamespace string) {
	data := newPodEventData("", podName, podNamespace, 0, "ERROR", "pod", err)
	sem.SendMessage(drainID, DrainFailed, message, data)
}

// sendMigrationEvent 发送迁移相关事件
func (sem *SimpleDrainEventManager) sendMigrationEvent(drainID string, eventType DrainEventType, message string, migrationInfo *DrainPodMigrationInfo) {
	data := map[string]interface{}{
		"migration": migrationInfo,
	}
	sem.SendMessage(drainID, eventType, message, data)
}

// SendPodSnapshotCreated 发送快照创建事件
func (sem *SimpleDrainEventManager) SendPodSnapshotCreated(drainID string, snapshot *DrainSnapshotInfo) {
	data := map[string]interface{}{
		"snapshot": snapshot,
	}
	message := fmt.Sprintf("Pod快照创建完成，共%d个Pod", snapshot.TotalPods)
	sem.SendMessage(drainID, PodSnapshotCreated, message, data)
}

// SendPodEvictionStarted 发送开始驱逐事件
func (sem *SimpleDrainEventManager) SendPodEvictionStarted(drainID string, migrationInfo *DrainPodMigrationInfo) {
	msg := fmt.Sprintf("Starting eviction for pod: %s/%s", migrationInfo.SourcePod.Namespace, migrationInfo.SourcePod.Name)
	sem.sendMigrationEvent(drainID, PodEvictionStarted, msg, migrationInfo)
}

// SendPodEvictionSucceeded 发送驱逐成功事件
func (sem *SimpleDrainEventManager) SendPodEvictionSucceeded(drainID string, migrationInfo *DrainPodMigrationInfo) {
	msg := fmt.Sprintf("Pod evicted successfully: %s/%s", migrationInfo.SourcePod.Namespace, migrationInfo.SourcePod.Name)
	sem.sendMigrationEvent(drainID, PodEvictionSucceeded, msg, migrationInfo)
}

// SendPodEvictionFailed 发送驱逐失败事件
func (sem *SimpleDrainEventManager) SendPodEvictionFailed(drainID string, migrationInfo *DrainPodMigrationInfo) {
	msg := fmt.Sprintf("Pod eviction failed: %s/%s - %s", migrationInfo.SourcePod.Namespace, migrationInfo.SourcePod.Name, migrationInfo.ErrorMessage)
	sem.sendMigrationEvent(drainID, PodEvictionFailed, msg, migrationInfo)
}

// SendNewPodDetected 发送发现新 Pod 事件
func (sem *SimpleDrainEventManager) SendNewPodDetected(drainID string, migrationInfo *DrainPodMigrationInfo) {
	msg := fmt.Sprintf("Detected new pod: %s/%s", migrationInfo.TargetPod.Namespace, migrationInfo.TargetPod.Name)
	sem.sendMigrationEvent(drainID, NewPodDetected, msg, migrationInfo)
}

// SendPodMigrationCompleted 发送迁移完成事件
func (sem *SimpleDrainEventManager) SendPodMigrationCompleted(drainID string, migrationInfo *DrainPodMigrationInfo) {
	msg := fmt.Sprintf("Pod migration completed: %s/%s -> %s/%s",
		migrationInfo.SourcePod.Namespace, migrationInfo.SourcePod.Name,
		migrationInfo.TargetPod.Namespace, migrationInfo.TargetPod.Name)
	sem.sendMigrationEvent(drainID, PodMigrationCompleted, msg, migrationInfo)
}

// SendPodMigrationFailed 发送迁移失败事件
func (sem *SimpleDrainEventManager) SendPodMigrationFailed(drainID string, migrationInfo *DrainPodMigrationInfo) {
	msg := fmt.Sprintf("Pod migration failed: %s/%s - %s", migrationInfo.SourcePod.Namespace, migrationInfo.SourcePod.Name, migrationInfo.ErrorMessage)
	sem.sendMigrationEvent(drainID, PodMigrationFailed, msg, migrationInfo)
}

// SendPodMigrationIgnored 发送迁移忽略事件
func (sem *SimpleDrainEventManager) SendPodMigrationIgnored(drainID string, migrationInfo *DrainPodMigrationInfo) {
	msg := fmt.Sprintf("Pod migration ignored (non-evictable): %s/%s", migrationInfo.SourcePod.Namespace, migrationInfo.SourcePod.Name)
	sem.sendMigrationEvent(drainID, PodMigrationIgnored, msg, migrationInfo)
}

// SendPodPending 发送Pod等待处理事件
func (sem *SimpleDrainEventManager) SendPodPending(drainID string, migrationInfo *DrainPodMigrationInfo) {
	msg := fmt.Sprintf("Pod waiting for processing: %s/%s", migrationInfo.SourcePod.Namespace, migrationInfo.SourcePod.Name)
	sem.sendMigrationEvent(drainID, PodPending, msg, migrationInfo)
}

// SendPodPhaseUpdated 发送新 Pod 状态变更事件
func (sem *SimpleDrainEventManager) SendPodPhaseUpdated(drainID string, migrationInfo *DrainPodMigrationInfo) {
	if migrationInfo.TargetPod == nil {
		return
	}
	if migrationInfo.TargetPod.PhaseReason != "" {
		msg := fmt.Sprintf("Pod phase updated: %s (%s)", migrationInfo.TargetPod.Phase, migrationInfo.TargetPod.PhaseReason)
		sem.sendMigrationEvent(drainID, PodPhaseUpdated, msg, migrationInfo)
		return
	}
	msg := fmt.Sprintf("Pod phase updated: %s", migrationInfo.TargetPod.Phase)
	sem.sendMigrationEvent(drainID, PodPhaseUpdated, msg, migrationInfo)
}

// HandleSSE 处理 SSE 连接并推送事件
func (sem *SimpleDrainEventManager) HandleSSE(c *gin.Context, drainID string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	origin := c.Request.Header.Get("Origin")
	if origin != "" {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
	} else {
		c.Header("Access-Control-Allow-Origin", "*")
	}

	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	log.Printf("[SSE] connected: drainID=%s, remote=%s", drainID, c.ClientIP())

	welcome := DrainMessage{Type: DrainStarted, DrainID: drainID, Message: "Connected to drain events", Timestamp: time.Now()}
	if data, err := json.Marshal(welcome); err == nil {
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			// client might have disconnected; stop streaming
			return
		}
		flusher.Flush()
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	connID, notifyCh := sem.registerConnection(drainID, cancel)
	defer sem.unregisterConnection(drainID, connID)

	localChan := sem.Subscribe(drainID)
	defer sem.Unsubscribe(drainID, localChan)

	redisChan := sem.startRedisSubscriber(ctx, drainID)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	recentPayloads := make(map[string]time.Time)

	defer func() { log.Printf("[SSE] disconnected: drainID=%s, remote=%s", drainID, c.ClientIP()) }()

	for {
		select {
		case <-ticker.C:
			cleanupRecentPayloads(recentPayloads, time.Now())
			if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case reason := <-notifyCh:
			closing := DrainMessage{Type: DrainStreamClosing, DrainID: drainID, Message: "Drain已完成，连接将关闭", Data: map[string]interface{}{"reason": reason}, Timestamp: time.Now()}
			if data, err := json.Marshal(closing); err == nil {
				if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
					return
				}
				flusher.Flush()
			}
			return
		case <-ctx.Done():
			return
		case msg, ok := <-redisChan:
			if !ok {
				redisChan = nil
				continue
			}
			if msg == nil || msg.Payload == "" {
				continue
			}
			payload := msg.Payload
			if !shouldEmitPayload(recentPayloads, payload, time.Now()) {
				continue
			}
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		case event, ok := <-localChan:
			if !ok {
				localChan = nil
				continue
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			payload := string(data)
			if !shouldEmitPayload(recentPayloads, payload, time.Now()) {
				continue
			}
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (sem *SimpleDrainEventManager) startRedisSubscriber(ctx context.Context, drainID string) <-chan *goredis.Message {
	if sem.redisHandler == nil {
		return nil
	}

	out := make(chan *goredis.Message, 256)
	go sem.redisSubscribeLoop(ctx, drainID, out)
	return out
}

func (sem *SimpleDrainEventManager) redisSubscribeLoop(ctx context.Context, drainID string, out chan<- *goredis.Message) {
	defer close(out)

	if sem.redisHandler == nil {
		return
	}

	backoff := redisInitialBackoff

outer:
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ps := sem.redisHandler.Subscribe(sem.channelName(drainID))
		if ps == nil {
			log.Printf("[SSE] redis subscribe returned nil: drainID=%s", drainID)
			if !sleepWithContext(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		setupCtx, cancel := context.WithTimeout(ctx, redisHandshakeTimeout)
		_, err := ps.Receive(setupCtx)
		cancel()
		if err != nil {
			_ = ps.Close()
			log.Printf("[SSE] redis handshake failed for drain %s: %v", drainID, err)
			if !sleepWithContext(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		backoff = redisInitialBackoff
		ch := ps.ChannelSize(1000)

		for {
			select {
			case <-ctx.Done():
				_ = ps.Close()
				return
			case msg, ok := <-ch:
				if !ok {
					_ = ps.Close()
					if !sleepWithContext(ctx, backoff) {
						return
					}
					backoff = nextBackoff(backoff)
					continue outer
				}
				if msg == nil || msg.Payload == "" {
					continue
				}
				select {
				case out <- msg:
				default:
					// drop to avoid blocking when client is slow
				}
			}
		}
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > redisMaxBackoff {
		next = redisMaxBackoff
	}
	return next
}

func shouldEmitPayload(cache map[string]time.Time, payload string, now time.Time) bool {
	if ts, ok := cache[payload]; ok && now.Sub(ts) < sseDedupWindow {
		return false
	}
	cache[payload] = now
	return true
}

func cleanupRecentPayloads(cache map[string]time.Time, now time.Time) {
	threshold := sseDedupWindow * 2
	for key, ts := range cache {
		if now.Sub(ts) > threshold {
			delete(cache, key)
		}
	}
}

// Cleanup 清理指定 Drain 的订阅者
func (sem *SimpleDrainEventManager) Cleanup(drainID string) {
	sem.mutex.Lock()
	defer sem.mutex.Unlock()

	if channels, exists := sem.subscribers[drainID]; exists {
		for _, ch := range channels {
			close(ch)
		}
		delete(sem.subscribers, drainID)
	}
	delete(sem.connectionCancels, drainID)

	sem.trackerMutex.Lock()
	for key := range sem.messageTracker {
		if strings.HasPrefix(key, drainID+":") {
			delete(sem.messageTracker, key)
		}
	}
	sem.trackerMutex.Unlock()
}

// DrainOperationManager 管理 Drain 统计和重试信息
type DrainOperationManager struct {
	eventManager *SimpleDrainEventManager
	statistics   map[string]*DrainStats
	retryStates  map[string]*RetryInfo
	mutex        sync.RWMutex
	redisHandler sharedservices.RedisClient
}

// DrainStats 为 Drain 简化统计信息
type DrainStats struct {
	DrainID       string    `json:"drainId"`
	StartTime     time.Time `json:"startTime"`
	TotalPods     int       `json:"totalPods"`
	ProcessedPods int       `json:"processedPods"`
	FailedPods    int       `json:"failedPods"`
	Status        string    `json:"status"`
}

// RetryInfo 为 Drain 重试信息
type RetryInfo struct {
	DrainID     string    `json:"drainId"`
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"maxAttempts"`
	LastError   string    `json:"lastError"`
	NextRetry   time.Time `json:"nextRetry"`
	CanRetry    bool      `json:"canRetry"`
}

// NewDrainOperationManager 创建 DrainOperationManager
func NewDrainOperationManager(eventManager *SimpleDrainEventManager, redisHandler sharedservices.RedisClient) *DrainOperationManager {
	return &DrainOperationManager{
		eventManager: eventManager,
		statistics:   make(map[string]*DrainStats),
		retryStates:  make(map[string]*RetryInfo),
		redisHandler: redisHandler,
	}
}

var domStatsTTL = 24 * time.Hour

func (dom *DrainOperationManager) statsKey(drainID string) string {
	return fmt.Sprintf("drain:stats:%s", drainID)
}
func (dom *DrainOperationManager) retryKey(drainID string) string {
	return fmt.Sprintf("drain:retry:%s", drainID)
}

// StartDrain 启动 Drain 的统计
func (dom *DrainOperationManager) StartDrain(drainID string, totalPods int) {
	dom.mutex.Lock()
	defer dom.mutex.Unlock()

	dom.statistics[drainID] = &DrainStats{
		DrainID:   drainID,
		StartTime: time.Now(),
		TotalPods: totalPods,
		Status:    "running",
	}
	if dom.redisHandler != nil {
		if b, err := json.Marshal(dom.statistics[drainID]); err == nil {
			dom.redisHandler.SetWithExpireTime(dom.statsKey(drainID), string(b), domStatsTTL)
		}
	}

	dom.eventManager.SendMessage(drainID, DrainStarted,
		fmt.Sprintf("开始drain操作，共%d个Pod", totalPods),
		map[string]interface{}{"totalPods": totalPods})
}

// UpdateProgress 更新 Drain 进度
func (dom *DrainOperationManager) UpdateProgress(drainID string, processedPods, failedPods int, message string) {
	dom.mutex.Lock()
	stats, exists := dom.statistics[drainID]
	if !exists {
		// 尝试从 Redis 加载
		if dom.redisHandler != nil {
			if s := dom.redisHandler.Get(dom.statsKey(drainID)); s != "" {
				var loaded DrainStats
				if json.Unmarshal([]byte(s), &loaded) == nil {
					stats = &loaded
					dom.statistics[drainID] = stats
				}
			}
		}
		if stats == nil {
			dom.mutex.Unlock()
			return
		}
	}

	stats.ProcessedPods = processedPods
	stats.FailedPods = failedPods

	var progress int
	if stats.TotalPods > 0 {
		progress = int(float64(processedPods) / float64(stats.TotalPods) * 100)
	}
	if dom.redisHandler != nil {
		if b, err := json.Marshal(stats); err == nil {
			dom.redisHandler.SetWithExpireTime(dom.statsKey(drainID), string(b), domStatsTTL)
		}
	}
	dom.mutex.Unlock()
	dom.eventManager.SendProgress(drainID, "evicting", message, progress)
}

// CompleteDrain 结束 Drain 并发送最终状态
func (dom *DrainOperationManager) CompleteDrain(drainID string, success bool, message string) {
	dom.mutex.Lock()
	stats, exists := dom.statistics[drainID]
	if !exists && dom.redisHandler != nil {
		if s := dom.redisHandler.Get(dom.statsKey(drainID)); s != "" {
			var loaded DrainStats
			if json.Unmarshal([]byte(s), &loaded) == nil {
				stats = &loaded
				dom.statistics[drainID] = stats
			}
		}
	}
	// 即使统计信息缺失，也不应阻断最终事件的发送（多实例或Redis短暂不一致情况下）
	if stats == nil {
		dom.mutex.Unlock()
		if success {
			dom.eventManager.SendMessage(drainID, DrainCompleted, message, nil)
		} else {
			dom.eventManager.SendMessage(drainID, DrainFailed, message, nil)
		}
		return
	}

	if success {
		stats.Status = "completed"
	} else {
		stats.Status = "failed"
	}
	if dom.redisHandler != nil {
		if b, err := json.Marshal(stats); err == nil {
			dom.redisHandler.SetWithExpireTime(dom.statsKey(drainID), string(b), domStatsTTL)
		}
	}
	dom.mutex.Unlock()
	if success {
		dom.eventManager.SendMessage(drainID, DrainCompleted, message, nil)
	} else {
		dom.eventManager.SendMessage(drainID, DrainFailed, message, nil)
	}
}

// GetStats 获取 Drain 统计
func (dom *DrainOperationManager) GetStats(drainID string) *DrainStats {
	// 优先从 Redis 取
	if dom.redisHandler != nil {
		if s := dom.redisHandler.Get(dom.statsKey(drainID)); s != "" {
			var stats DrainStats
			if json.Unmarshal([]byte(s), &stats) == nil {
				return &stats
			}
		}
	}
	dom.mutex.RLock()
	defer dom.mutex.RUnlock()
	if stats, exists := dom.statistics[drainID]; exists {
		statCopy := *stats
		return &statCopy
	}
	return nil
}

// SetRetryInfo 设置重试信息
func (dom *DrainOperationManager) SetRetryInfo(drainID string, attempts, maxAttempts int, lastError string, canRetry bool) {
	info := &RetryInfo{
		DrainID:     drainID,
		Attempts:    attempts,
		MaxAttempts: maxAttempts,
		LastError:   lastError,
		NextRetry:   time.Now().Add(time.Minute * time.Duration(attempts)),
		CanRetry:    canRetry,
	}
	if dom.redisHandler != nil {
		if b, err := json.Marshal(info); err == nil {
			dom.redisHandler.SetWithExpireTime(dom.retryKey(drainID), string(b), domStatsTTL)
		}
	}
	dom.mutex.Lock()
	dom.retryStates[drainID] = info
	dom.mutex.Unlock()
}

// GetRetryInfo 获取重试信息
func (dom *DrainOperationManager) GetRetryInfo(drainID string) *RetryInfo {
	if dom.redisHandler != nil {
		if s := dom.redisHandler.Get(dom.retryKey(drainID)); s != "" {
			var info RetryInfo
			if json.Unmarshal([]byte(s), &info) == nil {
				return &info
			}
		}
	}
	dom.mutex.RLock()
	defer dom.mutex.RUnlock()
	if retry, exists := dom.retryStates[drainID]; exists {
		retryCopy := *retry
		return &retryCopy
	}
	return nil
}

// Cleanup 清理指定 Drain 的统计与订阅
func (dom *DrainOperationManager) Cleanup(drainID string) {
	dom.mutex.Lock()
	defer dom.mutex.Unlock()

	delete(dom.statistics, drainID)
	delete(dom.retryStates, drainID)
	dom.eventManager.Cleanup(drainID)
	if dom.redisHandler != nil {
		dom.redisHandler.Delete(dom.statsKey(drainID))
		dom.redisHandler.Delete(dom.retryKey(drainID))
	}
}
