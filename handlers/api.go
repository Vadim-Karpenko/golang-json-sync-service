package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Message struct {
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

type Room struct {
	UUID string `json:"uuid"`
}

func UploadJSON(c *gin.Context, rdb *redis.Client) {
	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// Generate UUID
	id := uuid.New().String()

	// Store JSON in Redis
	jsonStr, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JSON marshalling failed"})
		return
	}

	// Store for 30 days
	err = rdb.Set(context.Background(), id, jsonStr, 30*24*3600*time.Second).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
		return
	}

	// Return UUID to the user
	c.JSON(http.StatusOK, gin.H{"uuid": id})
}

func GetJSON(c *gin.Context, rdb *redis.Client) {
	uuid := c.Param("uuid")

	// Retrieve JSON from Redis
	val, err := rdb.Get(context.Background(), uuid).Result()
	if err == redis.Nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "UUID not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
		return
	}

	c.Data(http.StatusOK, "application/json", []byte(val))
}
