package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type EmergencyController struct {
	emergencyService *services.EmergencyService
}

func NewEmergencyController(emergencyService *services.EmergencyService) *EmergencyController {
	return &EmergencyController{
		emergencyService: emergencyService,
	}
}

// =================== EMERGENCY ALERTS ===================

// GetEmergencyAlerts gets all emergency alerts
func (ec *EmergencyController) GetEmergencyAlerts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	alerts, err := ec.emergencyService.GetEmergencyAlerts(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency alerts failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency alerts")
		return
	}

	utils.SuccessResponse(c, "Emergency alerts retrieved successfully", alerts)
}

// CreateEmergencyAlert creates a new emergency alert
func (ec *EmergencyController) CreateEmergencyAlert(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreateEmergencyAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	alert, err := ec.emergencyService.CreateEmergencyAlert(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create emergency alert failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to create emergency alert")
		return
	}

	utils.CreatedResponse(c, "Emergency alert created successfully", alert)
}

// GetEmergencyAlert gets a specific emergency alert
func (ec *EmergencyController) GetEmergencyAlert(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	alert, err := ec.emergencyService.GetEmergencyAlert(c.Request.Context(), userID, alertID)
	if err != nil {
		logrus.Errorf("Get emergency alert failed: %v", err)

		switch err.Error() {
		case "alert not found":
			utils.NotFoundResponse(c, "Emergency alert")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this alert")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get emergency alert")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency alert retrieved successfully", alert)
}

// UpdateEmergencyAlert updates an emergency alert
func (ec *EmergencyController) UpdateEmergencyAlert(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")

	var req models.UpdateEmergencyAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	alert, err := ec.emergencyService.UpdateEmergencyAlert(c.Request.Context(), userID, alertID, req)
	if err != nil {
		logrus.Errorf("Update emergency alert failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update emergency alert")
		return
	}

	utils.SuccessResponse(c, "Emergency alert updated successfully", alert)
}

// DeleteEmergencyAlert deletes an emergency alert
func (ec *EmergencyController) DeleteEmergencyAlert(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")

	err := ec.emergencyService.DeleteEmergencyAlert(c.Request.Context(), userID, alertID)
	if err != nil {
		logrus.Errorf("Delete emergency alert failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to delete emergency alert")
		return
	}

	utils.SuccessResponse(c, "Emergency alert deleted successfully", nil)
}

// DismissEmergencyAlert dismisses an emergency alert
func (ec *EmergencyController) DismissEmergencyAlert(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")

	var req models.DismissAlertRequest
	c.ShouldBindJSON(&req)

	err := ec.emergencyService.DismissEmergencyAlert(c.Request.Context(), userID, alertID, req.Reason)
	if err != nil {
		logrus.Errorf("Dismiss emergency alert failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to dismiss emergency alert")
		return
	}

	utils.SuccessResponse(c, "Emergency alert dismissed successfully", nil)
}

// ResolveEmergencyAlert resolves an emergency alert
func (ec *EmergencyController) ResolveEmergencyAlert(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")

	var req models.ResolveAlertRequest
	c.ShouldBindJSON(&req)

	err := ec.emergencyService.ResolveEmergencyAlert(c.Request.Context(), userID, alertID, req.Resolution)
	if err != nil {
		logrus.Errorf("Resolve emergency alert failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to resolve emergency alert")
		return
	}

	utils.SuccessResponse(c, "Emergency alert resolved successfully", nil)
}

// =================== SOS FUNCTIONALITY ===================

// TriggerSOS triggers an SOS alert
func (ec *EmergencyController) TriggerSOS(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.TriggerSOSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	sos, err := ec.emergencyService.TriggerSOS(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Trigger SOS failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to trigger SOS")
		return
	}

	utils.CreatedResponse(c, "SOS triggered successfully", sos)
}

// CancelSOS cancels an active SOS
func (ec *EmergencyController) CancelSOS(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CancelSOSRequest
	c.ShouldBindJSON(&req)

	err := ec.emergencyService.CancelSOS(c.Request.Context(), userID, req.Reason)
	if err != nil {
		logrus.Errorf("Cancel SOS failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to cancel SOS")
		return
	}

	utils.SuccessResponse(c, "SOS cancelled successfully", nil)
}

// GetSOSStatus gets the current SOS status
func (ec *EmergencyController) GetSOSStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	status, err := ec.emergencyService.GetSOSStatus(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get SOS status failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get SOS status")
		return
	}

	utils.SuccessResponse(c, "SOS status retrieved successfully", status)
}

// UpdateSOSSettings updates SOS settings
func (ec *EmergencyController) UpdateSOSSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.SOSSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := ec.emergencyService.UpdateSOSSettings(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update SOS settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update SOS settings")
		return
	}

	utils.SuccessResponse(c, "SOS settings updated successfully", settings)
}

// GetSOSSettings gets SOS settings
func (ec *EmergencyController) GetSOSSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := ec.emergencyService.GetSOSSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get SOS settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get SOS settings")
		return
	}

	utils.SuccessResponse(c, "SOS settings retrieved successfully", settings)
}

// TestSOS tests SOS functionality
func (ec *EmergencyController) TestSOS(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	result, err := ec.emergencyService.TestSOS(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Test SOS failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to test SOS")
		return
	}

	utils.SuccessResponse(c, "SOS test completed successfully", result)
}

// =================== CRASH DETECTION ===================

// DetectCrash detects a potential crash
func (ec *EmergencyController) DetectCrash(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CrashDetectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	detection, err := ec.emergencyService.DetectCrash(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Detect crash failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to detect crash")
		return
	}

	utils.CreatedResponse(c, "Crash detection processed successfully", detection)
}

// ConfirmCrash confirms a crash detection
func (ec *EmergencyController) ConfirmCrash(c *gin.Context) {
	userID := c.GetString("userID")
	detectionID := c.Param("detectionId")

	var req models.ConfirmCrashRequest
	c.ShouldBindJSON(&req)

	err := ec.emergencyService.ConfirmCrash(c.Request.Context(), userID, detectionID, req)
	if err != nil {
		logrus.Errorf("Confirm crash failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to confirm crash")
		return
	}

	utils.SuccessResponse(c, "Crash confirmed successfully", nil)
}

// MarkFalseAlarm marks a crash detection as false alarm
func (ec *EmergencyController) MarkFalseAlarm(c *gin.Context) {
	userID := c.GetString("userID")
	detectionID := c.Param("detectionId")

	var req models.FalseAlarmRequest
	c.ShouldBindJSON(&req)

	err := ec.emergencyService.MarkFalseAlarm(c.Request.Context(), userID, detectionID, req.Reason)
	if err != nil {
		logrus.Errorf("Mark false alarm failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to mark false alarm")
		return
	}

	utils.SuccessResponse(c, "Marked as false alarm successfully", nil)
}

// GetCrashHistory gets crash detection history
func (ec *EmergencyController) GetCrashHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	history, err := ec.emergencyService.GetCrashHistory(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get crash history failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get crash history")
		return
	}

	utils.SuccessResponse(c, "Crash history retrieved successfully", history)
}

// GetCrashDetectionSettings gets crash detection settings
func (ec *EmergencyController) GetCrashDetectionSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := ec.emergencyService.GetCrashDetectionSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get crash detection settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get crash detection settings")
		return
	}

	utils.SuccessResponse(c, "Crash detection settings retrieved successfully", settings)
}

// UpdateCrashDetectionSettings updates crash detection settings
func (ec *EmergencyController) UpdateCrashDetectionSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CrashDetectionSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := ec.emergencyService.UpdateCrashDetectionSettings(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update crash detection settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update crash detection settings")
		return
	}

	utils.SuccessResponse(c, "Crash detection settings updated successfully", settings)
}

// CalibrateCrashDetection calibrates crash detection
func (ec *EmergencyController) CalibrateCrashDetection(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CalibrateCrashRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := ec.emergencyService.CalibrateCrashDetection(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Calibrate crash detection failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to calibrate crash detection")
		return
	}

	utils.SuccessResponse(c, "Crash detection calibrated successfully", result)
}

// =================== EMERGENCY CONTACTS ===================

// GetEmergencyContacts gets all emergency contacts
func (ec *EmergencyController) GetEmergencyContacts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	contacts, err := ec.emergencyService.GetEmergencyContacts(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency contacts failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency contacts")
		return
	}

	utils.SuccessResponse(c, "Emergency contacts retrieved successfully", contacts)
}

// AddEmergencyContact adds a new emergency contact
func (ec *EmergencyController) AddEmergencyContact(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.AddEmergencyContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	contact, err := ec.emergencyService.AddEmergencyContact(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Add emergency contact failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to add emergency contact")
		return
	}

	utils.CreatedResponse(c, "Emergency contact added successfully", contact)
}

// GetEmergencyContact gets a specific emergency contact
func (ec *EmergencyController) GetEmergencyContact(c *gin.Context) {
	userID := c.GetString("userID")
	contactID := c.Param("contactId")

	contact, err := ec.emergencyService.GetEmergencyContact(c.Request.Context(), userID, contactID)
	if err != nil {
		logrus.Errorf("Get emergency contact failed: %v", err)

		switch err.Error() {
		case "contact not found":
			utils.NotFoundResponse(c, "Emergency contact")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get emergency contact")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency contact retrieved successfully", contact)
}

// UpdateEmergencyContact updates an emergency contact
func (ec *EmergencyController) UpdateEmergencyContact(c *gin.Context) {
	userID := c.GetString("userID")
	contactID := c.Param("contactId")

	var req models.UpdateEmergencyContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	contact, err := ec.emergencyService.UpdateEmergencyContact(c.Request.Context(), userID, contactID, req)
	if err != nil {
		logrus.Errorf("Update emergency contact failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update emergency contact")
		return
	}

	utils.SuccessResponse(c, "Emergency contact updated successfully", contact)
}

// DeleteEmergencyContact deletes an emergency contact
func (ec *EmergencyController) DeleteEmergencyContact(c *gin.Context) {
	userID := c.GetString("userID")
	contactID := c.Param("contactId")

	err := ec.emergencyService.DeleteEmergencyContact(c.Request.Context(), userID, contactID)
	if err != nil {
		logrus.Errorf("Delete emergency contact failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to delete emergency contact")
		return
	}

	utils.SuccessResponse(c, "Emergency contact deleted successfully", nil)
}

// VerifyEmergencyContact verifies an emergency contact
func (ec *EmergencyController) VerifyEmergencyContact(c *gin.Context) {
	userID := c.GetString("userID")
	contactID := c.Param("contactId")

	var req models.VerifyContactRequest
	c.ShouldBindJSON(&req)

	result, err := ec.emergencyService.VerifyEmergencyContact(c.Request.Context(), userID, contactID, req)
	if err != nil {
		logrus.Errorf("Verify emergency contact failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to verify emergency contact")
		return
	}

	utils.SuccessResponse(c, "Emergency contact verification sent successfully", result)
}

// NotifyEmergencyContact notifies an emergency contact
func (ec *EmergencyController) NotifyEmergencyContact(c *gin.Context) {
	userID := c.GetString("userID")
	contactID := c.Param("contactId")

	var req models.NotifyContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := ec.emergencyService.NotifyEmergencyContact(c.Request.Context(), userID, contactID, req)
	if err != nil {
		logrus.Errorf("Notify emergency contact failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to notify emergency contact")
		return
	}

	utils.SuccessResponse(c, "Emergency contact notified successfully", nil)
}

// GetContactHistory gets contact notification history
func (ec *EmergencyController) GetContactHistory(c *gin.Context) {
	userID := c.GetString("userID")
	contactID := c.Param("contactId")

	history, err := ec.emergencyService.GetContactHistory(c.Request.Context(), userID, contactID)
	if err != nil {
		logrus.Errorf("Get contact history failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get contact history")
		return
	}

	utils.SuccessResponse(c, "Contact history retrieved successfully", history)
}

// =================== EMERGENCY SERVICES ===================

// GetNearbyEmergencyServices gets nearby emergency services
func (ec *EmergencyController) GetNearbyEmergencyServices(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	lat := c.Query("lat")
	lng := c.Query("lng")
	radius := c.Query("radius")

	services, err := ec.emergencyService.GetNearbyEmergencyServices(c.Request.Context(), userID, lat, lng, radius)
	if err != nil {
		logrus.Errorf("Get nearby emergency services failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get nearby emergency services")
		return
	}

	utils.SuccessResponse(c, "Nearby emergency services retrieved successfully", services)
}

// GetNearbyHospitals gets nearby hospitals
func (ec *EmergencyController) GetNearbyHospitals(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	lat := c.Query("lat")
	lng := c.Query("lng")
	radius := c.Query("radius")

	hospitals, err := ec.emergencyService.GetNearbyHospitals(c.Request.Context(), userID, lat, lng, radius)
	if err != nil {
		logrus.Errorf("Get nearby hospitals failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get nearby hospitals")
		return
	}

	utils.SuccessResponse(c, "Nearby hospitals retrieved successfully", hospitals)
}

// GetNearbyPoliceStations gets nearby police stations
func (ec *EmergencyController) GetNearbyPoliceStations(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	lat := c.Query("lat")
	lng := c.Query("lng")
	radius := c.Query("radius")

	stations, err := ec.emergencyService.GetNearbyPoliceStations(c.Request.Context(), userID, lat, lng, radius)
	if err != nil {
		logrus.Errorf("Get nearby police stations failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get nearby police stations")
		return
	}

	utils.SuccessResponse(c, "Nearby police stations retrieved successfully", stations)
}

// GetNearbyFireStations gets nearby fire stations
func (ec *EmergencyController) GetNearbyFireStations(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	lat := c.Query("lat")
	lng := c.Query("lng")
	radius := c.Query("radius")

	stations, err := ec.emergencyService.GetNearbyFireStations(c.Request.Context(), userID, lat, lng, radius)
	if err != nil {
		logrus.Errorf("Get nearby fire stations failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get nearby fire stations")
		return
	}

	utils.SuccessResponse(c, "Nearby fire stations retrieved successfully", stations)
}

// InitiateEmergencyCall initiates a call to emergency services
func (ec *EmergencyController) InitiateEmergencyCall(c *gin.Context) {
	userID := c.GetString("userID")
	serviceType := c.Param("serviceType")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.EmergencyCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := ec.emergencyService.InitiateEmergencyCall(c.Request.Context(), userID, serviceType, req)
	if err != nil {
		logrus.Errorf("Initiate emergency call failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to initiate emergency call")
		return
	}

	utils.SuccessResponse(c, "Emergency call initiated successfully", result)
}

// GetEmergencyNumbers gets emergency phone numbers
func (ec *EmergencyController) GetEmergencyNumbers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	numbers, err := ec.emergencyService.GetEmergencyNumbers(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency numbers failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency numbers")
		return
	}

	utils.SuccessResponse(c, "Emergency numbers retrieved successfully", numbers)
}

// UpdateEmergencyNumbers updates emergency phone numbers
func (ec *EmergencyController) UpdateEmergencyNumbers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateEmergencyNumbersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	numbers, err := ec.emergencyService.UpdateEmergencyNumbers(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update emergency numbers failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update emergency numbers")
		return
	}

	utils.SuccessResponse(c, "Emergency numbers updated successfully", numbers)
}

// =================== LOCATION SHARING ===================

// ShareEmergencyLocation shares location during emergency
func (ec *EmergencyController) ShareEmergencyLocation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.ShareLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	share, err := ec.emergencyService.ShareEmergencyLocation(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Share emergency location failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to share emergency location")
		return
	}

	utils.CreatedResponse(c, "Emergency location shared successfully", share)
}

// GetSharedEmergencyLocations gets shared emergency locations
func (ec *EmergencyController) GetSharedEmergencyLocations(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	locations, err := ec.emergencyService.GetSharedEmergencyLocations(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get shared emergency locations failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get shared emergency locations")
		return
	}

	utils.SuccessResponse(c, "Shared emergency locations retrieved successfully", locations)
}

// UpdateLocationShare updates location sharing settings
func (ec *EmergencyController) UpdateLocationShare(c *gin.Context) {
	userID := c.GetString("userID")
	shareID := c.Param("shareId")

	var req models.UpdateLocationShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	share, err := ec.emergencyService.UpdateLocationShare(c.Request.Context(), userID, shareID, req)
	if err != nil {
		logrus.Errorf("Update location share failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update location share")
		return
	}

	utils.SuccessResponse(c, "Location share updated successfully", share)
}

// StopLocationShare stops location sharing
func (ec *EmergencyController) StopLocationShare(c *gin.Context) {
	userID := c.GetString("userID")
	shareID := c.Param("shareId")

	err := ec.emergencyService.StopLocationShare(c.Request.Context(), userID, shareID)
	if err != nil {
		logrus.Errorf("Stop location share failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to stop location share")
		return
	}

	utils.SuccessResponse(c, "Location sharing stopped successfully", nil)
}

// TrackEmergencyLocation tracks emergency location
func (ec *EmergencyController) TrackEmergencyLocation(c *gin.Context) {
	userID := c.GetString("userID")
	shareID := c.Param("shareId")

	location, err := ec.emergencyService.TrackEmergencyLocation(c.Request.Context(), userID, shareID)
	if err != nil {
		logrus.Errorf("Track emergency location failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to track emergency location")
		return
	}

	utils.SuccessResponse(c, "Emergency location tracked successfully", location)
}

// =================== EMERGENCY RESPONSE ===================

// RespondToEmergency responds to an emergency
func (ec *EmergencyController) RespondToEmergency(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")

	var req models.EmergencyResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	response, err := ec.emergencyService.RespondToEmergency(c.Request.Context(), userID, alertID, req)
	if err != nil {
		logrus.Errorf("Respond to emergency failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to respond to emergency")
		return
	}

	utils.CreatedResponse(c, "Emergency response recorded successfully", response)
}

// GetEmergencyResponses gets responses to an emergency
func (ec *EmergencyController) GetEmergencyResponses(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")

	responses, err := ec.emergencyService.GetEmergencyResponses(c.Request.Context(), userID, alertID)
	if err != nil {
		logrus.Errorf("Get emergency responses failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency responses")
		return
	}

	utils.SuccessResponse(c, "Emergency responses retrieved successfully", responses)
}

// UpdateEmergencyResponse updates an emergency response
func (ec *EmergencyController) UpdateEmergencyResponse(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")
	responseID := c.Param("responseId")

	var req models.UpdateEmergencyResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	response, err := ec.emergencyService.UpdateEmergencyResponse(c.Request.Context(), userID, alertID, responseID, req)
	if err != nil {
		logrus.Errorf("Update emergency response failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update emergency response")
		return
	}

	utils.SuccessResponse(c, "Emergency response updated successfully", response)
}

// RequestHelp requests help for an emergency
func (ec *EmergencyController) RequestHelp(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")

	var req models.RequestHelpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	request, err := ec.emergencyService.RequestHelp(c.Request.Context(), userID, alertID, req)
	if err != nil {
		logrus.Errorf("Request help failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to request help")
		return
	}

	utils.CreatedResponse(c, "Help request sent successfully", request)
}

// OfferHelp offers help for an emergency
func (ec *EmergencyController) OfferHelp(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")

	var req models.OfferHelpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	offer, err := ec.emergencyService.OfferHelp(c.Request.Context(), userID, alertID, req)
	if err != nil {
		logrus.Errorf("Offer help failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to offer help")
		return
	}

	utils.CreatedResponse(c, "Help offer sent successfully", offer)
}

// =================== CHECK-IN SAFETY ===================

// CheckInSafe marks user as safe
func (ec *EmergencyController) CheckInSafe(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.SafeCheckInRequest
	c.ShouldBindJSON(&req)

	checkIn, err := ec.emergencyService.CheckInSafe(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Check in safe failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to check in safe")
		return
	}

	utils.SuccessResponse(c, "Checked in as safe successfully", checkIn)
}

// CheckInNotSafe marks user as not safe
func (ec *EmergencyController) CheckInNotSafe(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.NotSafeCheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	checkIn, err := ec.emergencyService.CheckInNotSafe(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Check in not safe failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to check in not safe")
		return
	}

	utils.CreatedResponse(c, "Checked in as not safe - emergency protocols activated", checkIn)
}

// GetCheckInStatus gets check-in status
func (ec *EmergencyController) GetCheckInStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	status, err := ec.emergencyService.GetCheckInStatus(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get check-in status failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get check-in status")
		return
	}

	utils.SuccessResponse(c, "Check-in status retrieved successfully", status)
}

// UpdateCheckInSettings updates check-in settings
func (ec *EmergencyController) UpdateCheckInSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CheckInSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := ec.emergencyService.UpdateCheckInSettings(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update check-in settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update check-in settings")
		return
	}

	utils.SuccessResponse(c, "Check-in settings updated successfully", settings)
}

// GetCheckInSettings gets check-in settings
func (ec *EmergencyController) GetCheckInSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := ec.emergencyService.GetCheckInSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get check-in settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get check-in settings")
		return
	}

	utils.SuccessResponse(c, "Check-in settings retrieved successfully", settings)
}

// RequestCheckIn requests check-in from a user
func (ec *EmergencyController) RequestCheckIn(c *gin.Context) {
	userID := c.GetString("userID")
	targetUserID := c.Param("userId")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.RequestCheckInRequest
	c.ShouldBindJSON(&req)

	request, err := ec.emergencyService.RequestCheckIn(c.Request.Context(), userID, targetUserID, req)
	if err != nil {
		logrus.Errorf("Request check-in failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to request check-in")
		return
	}

	utils.CreatedResponse(c, "Check-in request sent successfully", request)
}

// GetCheckInRequests gets check-in requests
func (ec *EmergencyController) GetCheckInRequests(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	requests, err := ec.emergencyService.GetCheckInRequests(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get check-in requests failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get check-in requests")
		return
	}

	utils.SuccessResponse(c, "Check-in requests retrieved successfully", requests)
}

// =================== EMERGENCY HISTORY ===================

// GetEmergencyHistory gets emergency history
func (ec *EmergencyController) GetEmergencyHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	history, err := ec.emergencyService.GetEmergencyHistory(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency history failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency history")
		return
	}

	utils.SuccessResponse(c, "Emergency history retrieved successfully", history)
}

// GetEmergencyTimeline gets emergency timeline
func (ec *EmergencyController) GetEmergencyTimeline(c *gin.Context) {
	userID := c.GetString("userID")
	alertID := c.Param("alertId")

	timeline, err := ec.emergencyService.GetEmergencyTimeline(c.Request.Context(), userID, alertID)
	if err != nil {
		logrus.Errorf("Get emergency timeline failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency timeline")
		return
	}

	utils.SuccessResponse(c, "Emergency timeline retrieved successfully", timeline)
}

// GetEmergencyStats gets emergency statistics
func (ec *EmergencyController) GetEmergencyStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	stats, err := ec.emergencyService.GetEmergencyStats(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency stats")
		return
	}

	utils.SuccessResponse(c, "Emergency stats retrieved successfully", stats)
}

// ExportEmergencyHistory exports emergency history
func (ec *EmergencyController) ExportEmergencyHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.ExportHistoryRequest
	c.ShouldBindJSON(&req)

	exportJob, err := ec.emergencyService.ExportEmergencyHistory(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Export emergency history failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to export emergency history")
		return
	}

	utils.CreatedResponse(c, "Emergency history export started successfully", exportJob)
}

// DownloadEmergencyExport downloads emergency export
func (ec *EmergencyController) DownloadEmergencyExport(c *gin.Context) {
	userID := c.GetString("userID")
	exportID := c.Param("exportId")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	file, err := ec.emergencyService.DownloadEmergencyExport(c.Request.Context(), userID, exportID)
	if err != nil {
		logrus.Errorf("Download emergency export failed: %v", err)

		switch err.Error() {
		case "export not found":
			utils.NotFoundResponse(c, "Export")
		case "export not ready":
			utils.BadRequestResponse(c, "Export is still processing")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this export")
		default:
			utils.InternalServerErrorResponse(c, "Failed to download export")
		}
		return
	}

	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Disposition", "attachment; filename="+file.Filename)
	c.Data(200, file.ContentType, file.Data)
}

// =================== EMERGENCY SETTINGS ===================

// GetEmergencySettings gets emergency settings
func (ec *EmergencyController) GetEmergencySettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := ec.emergencyService.GetEmergencySettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency settings")
		return
	}

	utils.SuccessResponse(c, "Emergency settings retrieved successfully", settings)
}

// UpdateEmergencySettings updates emergency settings
func (ec *EmergencyController) UpdateEmergencySettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.EmergencySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := ec.emergencyService.UpdateEmergencySettings(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update emergency settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update emergency settings")
		return
	}

	utils.SuccessResponse(c, "Emergency settings updated successfully", settings)
}

// GetEmergencyNotificationSettings gets notification settings
func (ec *EmergencyController) GetEmergencyNotificationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := ec.emergencyService.GetEmergencyNotificationSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency notification settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency notification settings")
		return
	}

	utils.SuccessResponse(c, "Emergency notification settings retrieved successfully", settings)
}

// UpdateEmergencyNotificationSettings updates notification settings
func (ec *EmergencyController) UpdateEmergencyNotificationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.EmergencyNotificationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := ec.emergencyService.UpdateEmergencyNotificationSettings(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update emergency notification settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update emergency notification settings")
		return
	}

	utils.SuccessResponse(c, "Emergency notification settings updated successfully", settings)
}

// GetEmergencyAutomationSettings gets automation settings
func (ec *EmergencyController) GetEmergencyAutomationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := ec.emergencyService.GetEmergencyAutomationSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency automation settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency automation settings")
		return
	}

	utils.SuccessResponse(c, "Emergency automation settings retrieved successfully", settings)
}

// UpdateEmergencyAutomationSettings updates automation settings
func (ec *EmergencyController) UpdateEmergencyAutomationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.EmergencyAutomationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := ec.emergencyService.UpdateEmergencyAutomationSettings(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update emergency automation settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update emergency automation settings")
		return
	}

	utils.SuccessResponse(c, "Emergency automation settings updated successfully", settings)
}

// =================== EMERGENCY DRILLS ===================

// GetEmergencyDrills gets emergency drills
func (ec *EmergencyController) GetEmergencyDrills(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	drills, err := ec.emergencyService.GetEmergencyDrills(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency drills failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency drills")
		return
	}

	utils.SuccessResponse(c, "Emergency drills retrieved successfully", drills)
}

// CreateEmergencyDrill creates a new emergency drill
func (ec *EmergencyController) CreateEmergencyDrill(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreateEmergencyDrillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	drill, err := ec.emergencyService.CreateEmergencyDrill(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create emergency drill failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to create emergency drill")
		return
	}

	utils.CreatedResponse(c, "Emergency drill created successfully", drill)
}

// StartEmergencyDrill starts an emergency drill
func (ec *EmergencyController) StartEmergencyDrill(c *gin.Context) {
	userID := c.GetString("userID")
	drillID := c.Param("drillId")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	drill, err := ec.emergencyService.StartEmergencyDrill(c.Request.Context(), userID, drillID)
	if err != nil {
		logrus.Errorf("Start emergency drill failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to start emergency drill")
		return
	}

	utils.SuccessResponse(c, "Emergency drill started successfully", drill)
}

// CompleteEmergencyDrill completes an emergency drill
func (ec *EmergencyController) CompleteEmergencyDrill(c *gin.Context) {
	userID := c.GetString("userID")
	drillID := c.Param("drillId")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CompleteEmergencyDrillRequest
	c.ShouldBindJSON(&req)

	drill, err := ec.emergencyService.CompleteEmergencyDrill(c.Request.Context(), userID, drillID, req)
	if err != nil {
		logrus.Errorf("Complete emergency drill failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to complete emergency drill")
		return
	}

	utils.SuccessResponse(c, "Emergency drill completed successfully", drill)
}

// GetDrillResults gets drill results
func (ec *EmergencyController) GetDrillResults(c *gin.Context) {
	userID := c.GetString("userID")
	drillID := c.Param("drillId")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	results, err := ec.emergencyService.GetDrillResults(c.Request.Context(), userID, drillID)
	if err != nil {
		logrus.Errorf("Get drill results failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get drill results")
		return
	}

	utils.SuccessResponse(c, "Drill results retrieved successfully", results)
}

// DeleteEmergencyDrill deletes an emergency drill
func (ec *EmergencyController) DeleteEmergencyDrill(c *gin.Context) {
	userID := c.GetString("userID")
	drillID := c.Param("drillId")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	err := ec.emergencyService.DeleteEmergencyDrill(c.Request.Context(), userID, drillID)
	if err != nil {
		logrus.Errorf("Delete emergency drill failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to delete emergency drill")
		return
	}

	utils.SuccessResponse(c, "Emergency drill deleted successfully", nil)
}

// =================== MEDICAL INFORMATION ===================

// GetMedicalInformation gets medical information
func (ec *EmergencyController) GetMedicalInformation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	info, err := ec.emergencyService.GetMedicalInformation(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get medical information failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get medical information")
		return
	}

	utils.SuccessResponse(c, "Medical information retrieved successfully", info)
}

// UpdateMedicalInformation updates medical information
func (ec *EmergencyController) UpdateMedicalInformation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.MedicalInformationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	info, err := ec.emergencyService.UpdateMedicalInformation(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update medical information failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update medical information")
		return
	}

	utils.SuccessResponse(c, "Medical information updated successfully", info)
}

// GetAllergies gets allergies
func (ec *EmergencyController) GetAllergies(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	allergies, err := ec.emergencyService.GetAllergies(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get allergies failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get allergies")
		return
	}

	utils.SuccessResponse(c, "Allergies retrieved successfully", allergies)
}

// UpdateAllergies updates allergies
func (ec *EmergencyController) UpdateAllergies(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.AllergiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	allergies, err := ec.emergencyService.UpdateAllergies(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update allergies failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update allergies")
		return
	}

	utils.SuccessResponse(c, "Allergies updated successfully", allergies)
}

// GetMedications gets medications
func (ec *EmergencyController) GetMedications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	medications, err := ec.emergencyService.GetMedications(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get medications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get medications")
		return
	}

	utils.SuccessResponse(c, "Medications retrieved successfully", medications)
}

// UpdateMedications updates medications
func (ec *EmergencyController) UpdateMedications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.MedicationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	medications, err := ec.emergencyService.UpdateMedications(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update medications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update medications")
		return
	}

	utils.SuccessResponse(c, "Medications updated successfully", medications)
}

// GetMedicalConditions gets medical conditions
func (ec *EmergencyController) GetMedicalConditions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	conditions, err := ec.emergencyService.GetMedicalConditions(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get medical conditions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get medical conditions")
		return
	}

	utils.SuccessResponse(c, "Medical conditions retrieved successfully", conditions)
}

// UpdateMedicalConditions updates medical conditions
func (ec *EmergencyController) UpdateMedicalConditions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.MedicalConditionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	conditions, err := ec.emergencyService.UpdateMedicalConditions(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update medical conditions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update medical conditions")
		return
	}

	utils.SuccessResponse(c, "Medical conditions updated successfully", conditions)
}

// =================== EMERGENCY BROADCAST ===================

// BroadcastEmergency broadcasts an emergency
func (ec *EmergencyController) BroadcastEmergency(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.BroadcastEmergencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	broadcast, err := ec.emergencyService.BroadcastEmergency(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Broadcast emergency failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to broadcast emergency")
		return
	}

	utils.CreatedResponse(c, "Emergency broadcast sent successfully", broadcast)
}

// GetEmergencyBroadcasts gets emergency broadcasts
func (ec *EmergencyController) GetEmergencyBroadcasts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	broadcasts, err := ec.emergencyService.GetEmergencyBroadcasts(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency broadcasts failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency broadcasts")
		return
	}

	utils.SuccessResponse(c, "Emergency broadcasts retrieved successfully", broadcasts)
}

// UpdateEmergencyBroadcast updates an emergency broadcast
func (ec *EmergencyController) UpdateEmergencyBroadcast(c *gin.Context) {
	userID := c.GetString("userID")
	broadcastID := c.Param("broadcastId")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateBroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	broadcast, err := ec.emergencyService.UpdateEmergencyBroadcast(c.Request.Context(), userID, broadcastID, req)
	if err != nil {
		logrus.Errorf("Update emergency broadcast failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update emergency broadcast")
		return
	}

	utils.SuccessResponse(c, "Emergency broadcast updated successfully", broadcast)
}

// DeleteEmergencyBroadcast deletes an emergency broadcast
func (ec *EmergencyController) DeleteEmergencyBroadcast(c *gin.Context) {
	userID := c.GetString("userID")
	broadcastID := c.Param("broadcastId")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	err := ec.emergencyService.DeleteEmergencyBroadcast(c.Request.Context(), userID, broadcastID)
	if err != nil {
		logrus.Errorf("Delete emergency broadcast failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to delete emergency broadcast")
		return
	}

	utils.SuccessResponse(c, "Emergency broadcast deleted successfully", nil)
}

// AcknowledgeBroadcast acknowledges an emergency broadcast
func (ec *EmergencyController) AcknowledgeBroadcast(c *gin.Context) {
	userID := c.GetString("userID")
	broadcastID := c.Param("broadcastId")

	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.AcknowledgeBroadcastRequest
	c.ShouldBindJSON(&req)

	err := ec.emergencyService.AcknowledgeBroadcast(c.Request.Context(), userID, broadcastID, req)
	if err != nil {
		logrus.Errorf("Acknowledge broadcast failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to acknowledge broadcast")
		return
	}

	utils.SuccessResponse(c, "Broadcast acknowledged successfully", nil)
}
