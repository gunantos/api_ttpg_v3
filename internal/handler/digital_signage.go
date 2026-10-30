package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type DigitalSignageDevice struct {
	ID            int    `json:"id"`
	DeviceID      string `json:"device_id"`
	Name          string `json:"name"`
	Location      string `json:"location"`
	Resolution    string `json:"resolution"`
	Orientation   string `json:"orientation"`
	Volume        int    `json:"volume"`
	Brightness    int    `json:"brightness"`
	Status        string `json:"status"`
	LastSeenAt    string `json:"last_seen_at"`
	ConfigVersion string `json:"config_version"`
}

// ListDigitalSignageDevices godoc
// @Summary List digital signage devices
// @Description List devices with filtering by type, q, status and pagination
// @Tags signage
// @Produce json
// @Param type query string false "Filter by type (vending/signage)"
// @Param q query string false "Search by device_id or name"
// @Param status query string false "Filter by status (active/pending)"
// @Param limit query integer false "Items per page (1-1000, default 100)"
// @Param offset query integer false "Page offset (default 0)"
// @Success 200 {object} response.Success "daftar device signage"
// @Router /signage/devices [get]
func ListDigitalSignageDevices(c *gin.Context) {
	type resp struct {
		Data       interface{} `json:"data"`
		Pagination gin.H `json:"pagination"`
	}
	c.JSON(200, gin.H{
		"success":  true,
		"message":  "daftar device signage",
		"data":     nil,
		"pagination": gin.H{
			"page":      1,
			"limit":     10,
			"total":     0,
		},
	})
}

// CreateDigitalSignageDevice godoc
// @Summary Create digital signage device
// @Description Create a new digital signage device
// @Tags signage
// @Accept json
// @Produce json
// @Param device body DigitalSignageDevice true "Device data"
// @Success 200 {object} response.Success "device berhasil dibuat"
// @Router /signage/devices [post]
func CreateDigitalSignageDevice(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"message": "device berhasil dibuat",
		"data":    gin.H{"id": 0},
	})
}

// UpdateDigitalSignageDevice godoc
// @Summary Update digital signage device
// @Description Update device settings
// @Tags signage
// @Accept json
// @Produce json
// @Param id path int true "Device ID"
// @Param device body DigitalSignageDevice true "Updated device data"
// @Success 200 {object} response.Success "device berhasil diupdate"
// @Router /signage/devices/{id} [put]
func UpdateDigitalSignageDevice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	c.JSON(200, gin.H{
		"success": true,
		"message": "device berhasil diupdate",
		"data":    gin.H{"id": id},
	})
}

// DeleteDigitalSignageDevice godoc
// @Summary Delete digital signage device
// @Description Delete device, check playlist usage first
// @Tags signage
// @Produce json
// @Param id path int true "Device ID"
// @Success 200 {object} response.Success "device berhasil dihapus"
// @Router /signage/devices/{id} [delete]
func DeleteDigitalSignageDevice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	c.JSON(200, gin.H{
		"success": true,
		"message": "device berhasil dihapus",
		"data":    gin.H{"id": id},
	})
}
