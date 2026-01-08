package services

import (
	"fmt"

	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"

	"go.uber.org/zap"
)

// Subscribe 订阅RCA分析流
func (s *RCAService) Subscribe(alertID string) (chan string, func()) {
	ch := make(chan string, 10)
	s.clientsMux.Lock()
	s.clients[alertID] = append(s.clients[alertID], ch)
	s.clientsMux.Unlock()

	return ch, func() {
		s.clientsMux.Lock()
		defer s.clientsMux.Unlock()
		clients := s.clients[alertID]
		for i, client := range clients {
			if client == ch {
				s.clients[alertID] = append(clients[:i], clients[i+1:]...)
				close(ch)
				break
			}
		}
		if len(s.clients[alertID]) == 0 {
			delete(s.clients, alertID)
		}
	}
}

// Broadcast 广播消息给订阅者
func (s *RCAService) Broadcast(alertID string, message string) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()
	for _, ch := range s.clients[alertID] {
		select {
		case ch <- message:
		default:
		}
	}
}

func (s *RCAService) broadcastStatus(alertID string, status models.RCAStatus, runID ...string) {
	msg := fmt.Sprintf(`{"type":"status","status":"%s"`, status)
	if len(runID) > 0 && runID[0] != "" {
		msg += fmt.Sprintf(`,"run_id":"%s"`, runID[0])
	}
	msg += "}"

	s.clientsMux.RLock()
	clientCount := len(s.clients[alertID])
	s.clientsMux.RUnlock()

	logger.L().Debug("广播RCA状态更新",
		zap.String("alert_id", alertID),
		zap.String("status", string(status)),
		zap.Int("client_count", clientCount),
		zap.String("message", msg),
	)

	s.Broadcast(alertID, msg)
}
