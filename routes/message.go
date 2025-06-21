// routes/message.go
package routes

import (
	"ftrack/controllers"
	"ftrack/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// SetupMessageRoutes configures messaging and chat related routes
func SetupMessageRoutes(router *gin.RouterGroup, messageController *controllers.MessageController, redis *redis.Client) {
	messages := router.Group("/messages")

	// Basic message operations
	messages.GET("/circle/:circleId", messageController.GetMessages)
	messages.POST("/", messageController.SendMessage)
	messages.GET("/:messageId", messageController.GetMessage)
	messages.PUT("/:messageId", messageController.UpdateMessage)
	messages.DELETE("/:messageId", messageController.DeleteMessage)

	// Message threading and replies
	threading := messages.Group("/:messageId/replies")
	{
		threading.GET("/", messageController.GetReplies)
		threading.POST("/", messageController.ReplyToMessage)
		threading.GET("/:replyId", messageController.GetReply)
		threading.PUT("/:replyId", messageController.UpdateReply)
		threading.DELETE("/:replyId", messageController.DeleteReply)
	}

	// Message reactions and emojis
	reactions := messages.Group("/:messageId/reactions")
	{
		reactions.GET("/", messageController.GetReactions)
		reactions.POST("/:emoji", messageController.AddReaction)
		reactions.DELETE("/:emoji", messageController.RemoveReaction)
		reactions.GET("/users/:emoji", messageController.GetReactionUsers)
	}

	// Media handling
	media := messages.Group("/media")
	media.Use(middleware.UploadRateLimit(redis))
	{
		media.POST("/upload", messageController.UploadMedia)
		media.GET("/:mediaId", messageController.GetMedia)
		media.DELETE("/:mediaId", messageController.DeleteMedia)
		media.GET("/:mediaId/thumbnail", messageController.GetMediaThumbnail)
		media.POST("/:mediaId/compress", messageController.CompressMedia)
	}

	// Message search and filtering
	search := messages.Group("/search")
	{
		search.GET("/", messageController.SearchMessages)
		search.GET("/circle/:circleId", messageController.SearchInCircle)
		search.GET("/media", messageController.SearchMedia)
		search.GET("/mentions", messageController.SearchMentions)
		search.GET("/links", messageController.SearchLinks)
		search.GET("/files", messageController.SearchFiles)
	}

	// Message status and delivery
	status := messages.Group("/status")
	{
		status.PUT("/:messageId/read", messageController.MarkAsRead)
		status.PUT("/:messageId/unread", messageController.MarkAsUnread)
		status.PUT("/bulk/read", messageController.BulkMarkAsRead)
		status.GET("/:messageId/delivery", messageController.GetDeliveryStatus)
		status.GET("/:messageId/read-receipts", messageController.GetReadReceipts)
	}

	// Message forwarding
	forwarding := messages.Group("/:messageId/forward")
	{
		forwarding.POST("/circle/:circleId", messageController.ForwardToCircle)
		forwarding.POST("/user/:userId", messageController.ForwardToUser)
		forwarding.GET("/history", messageController.GetForwardHistory)
	}

	// Message scheduling
	scheduling := messages.Group("/schedule")
	{
		scheduling.POST("/", messageController.ScheduleMessage)
		scheduling.GET("/", messageController.GetScheduledMessages)
		scheduling.GET("/:scheduleId", messageController.GetScheduledMessage)
		scheduling.PUT("/:scheduleId", messageController.UpdateScheduledMessage)
		scheduling.DELETE("/:scheduleId", messageController.CancelScheduledMessage)
	}

	// Message templates and quick replies
	templates := messages.Group("/templates")
	{
		templates.GET("/", messageController.GetMessageTemplates)
		templates.POST("/", messageController.CreateMessageTemplate)
		templates.GET("/:templateId", messageController.GetMessageTemplate)
		templates.PUT("/:templateId", messageController.UpdateMessageTemplate)
		templates.DELETE("/:templateId", messageController.DeleteMessageTemplate)
		templates.POST("/:templateId/use", messageController.UseMessageTemplate)
	}

	// Message backup and export
	backup := messages.Group("/backup")
	{
		backup.POST("/export/:circleId", messageController.ExportCircleMessages)
		backup.GET("/export/status", messageController.GetExportStatus)
		backup.GET("/export/:exportId/download", messageController.DownloadMessageExport)
		backup.POST("/import", messageController.ImportMessages)
	}

	// Message moderation
	moderation := messages.Group("/moderation")
	{
		moderation.POST("/:messageId/report", messageController.ReportMessage)
		moderation.PUT("/:messageId/flag", messageController.FlagMessage)
		moderation.DELETE("/:messageId/admin", messageController.AdminDeleteMessage)
		moderation.GET("/reports", messageController.GetMessageReports)
		moderation.PUT("/reports/:reportId", messageController.HandleMessageReport)
	}

	// Typing indicators
	typing := messages.Group("/typing")
	{
		typing.POST("/start/:circleId", messageController.StartTyping)
		typing.POST("/stop/:circleId", messageController.StopTyping)
		typing.GET("/status/:circleId", messageController.GetTypingStatus)
	}

	// Message analytics
	analytics := messages.Group("/analytics")
	{
		analytics.GET("/stats", messageController.GetMessageStats)
		analytics.GET("/circle/:circleId/stats", messageController.GetCircleMessageStats)
		analytics.GET("/activity", messageController.GetMessageActivity)
		analytics.GET("/trends", messageController.GetMessageTrends)
		analytics.GET("/popular", messageController.GetPopularMessages)
	}

	// Message automation and bots
	automation := messages.Group("/automation")
	{
		automation.GET("/rules", messageController.GetAutomationRules)
		automation.POST("/rules", messageController.CreateAutomationRule)
		automation.PUT("/rules/:ruleId", messageController.UpdateAutomationRule)
		automation.DELETE("/rules/:ruleId", messageController.DeleteAutomationRule)
		automation.POST("/rules/:ruleId/test", messageController.TestAutomationRule)
	}

	// Message drafts
	drafts := messages.Group("/drafts")
	{
		drafts.GET("/", messageController.GetDrafts)
		drafts.POST("/", messageController.SaveDraft)
		drafts.GET("/:draftId", messageController.GetDraft)
		drafts.PUT("/:draftId", messageController.UpdateDraft)
		drafts.DELETE("/:draftId", messageController.DeleteDraft)
		drafts.POST("/:draftId/send", messageController.SendDraft)
	}
}
