package services

import (
	"ftrack/models"
	"ftrack/websocket"
)

type WebSocketService struct {
	hub *websocket.Hub
}

func NewWebSocketService(hub *websocket.Hub) *WebSocketService {
	return &WebSocketService{
		hub: hub,
	}
}

func (ws *WebSocketService) BroadcastLocationUpdate(userID string, circleIDs []string, location models.Location) {
	ws.hub.BroadcastLocationUpdate(userID, circleIDs, location)
}

func (ws *WebSocketService) BroadcastPlaceEvent(userID string, circleIDs []string, placeEvent models.WSPlaceEvent) {
	ws.hub.BroadcastPlaceEvent(userID, circleIDs, placeEvent)
}

func (ws *WebSocketService) BroadcastEmergencyAlert(circleIDs []string, alert models.WSEmergencyAlert) {
	ws.hub.BroadcastEmergencyAlert(circleIDs, alert)
}

func (ws *WebSocketService) SendNotificationToUser(userID string, notification interface{}) {
	ws.hub.SendNotificationToUser(userID, notification)
}

func (ws *WebSocketService) GetConnectedUsers() []string {
	return ws.hub.GetConnectedUsers()
}

func (ws *WebSocketService) IsUserOnline(userID string) bool {
	return ws.hub.IsUserOnline(userID)
}

func (ws *WebSocketService) GetCircleMembers(circleID string) []*websocket.Client {
	return ws.hub.GetCircleMembers(circleID)
}
