package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Vadim-Karpenko/golang-json-sync-service/handlers"
	"github.com/Vadim-Karpenko/golang-json-sync-service/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var rdb *redis.Client

func setupRouter() *gin.Engine {
	r := gin.Default()
	// Initialize Redis
	rdb := utils.InitializeRedis()

	// Routes
	r.POST("/upload", func(c *gin.Context) {
		handlers.UploadJSON(c, rdb)
	})
	r.GET("/ws/:uuid", func(c *gin.Context) {
		handlers.HandleWebSocket(c, rdb)
	})
	r.GET("/json/:uuid", func(c *gin.Context) {
		handlers.GetJSON(c, rdb)
	})

	return r
}

func TestUploadAndGetJSON(t *testing.T) {
	router := setupRouter()

	// Test JSON data
	testData := map[string]interface{}{
		"character": map[string]interface{}{
			"name": "Aragorn",
			"age":  87.0,
		},
	}
	jsonData, _ := json.Marshal(testData)

	// Upload the JSON
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", bytes.NewBuffer(jsonData))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	uuid := response["uuid"]

	// Retrieve the JSON using the UUID
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/json/"+uuid, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var retrievedData map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &retrievedData)
	assert.Equal(t, testData, retrievedData)
}

func TestWebSocketSync(t *testing.T) {
	router := setupRouter()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r)
	}))
	defer server.Close()

	// Start by uploading a JSON
	testData := map[string]interface{}{
		"character": map[string]interface{}{
			"name": "Frodo",
			"age":  50.0,
		},
	}
	jsonData, _ := json.Marshal(testData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", bytes.NewBuffer(jsonData))
	router.ServeHTTP(w, req)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	uuid := response["uuid"]

	// Establish a WebSocket connection
	wsURL := "ws" + server.URL[4:] + "/ws/" + uuid
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer ws.Close()

	// Function to send updates and verify results
	sendAndVerifyUpdate := func(updateMessage handlers.Message, expectedData map[string]interface{}) {
		// Send an update via WebSocket
		updateData, _ := json.Marshal(updateMessage)
		ws.WriteMessage(websocket.TextMessage, updateData)

		// Wait a moment for the update to propagate
		time.Sleep(500 * time.Millisecond)

		// Check if the update was applied by retrieving the JSON
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/json/"+uuid, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var updatedData map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &updatedData)

		assert.Equal(t, expectedData, updatedData)
	}

	// Test the first update
	sendAndVerifyUpdate(
		handlers.Message{
			Path:  "character.age",
			Value: 51,
		},
		map[string]interface{}{
			"character": map[string]interface{}{
				"name": "Frodo",
				"age":  51.0,
			},
		},
	)

	// Update the JSON with list data
	testDataList := map[string]interface{}{
		"character": map[string]interface{}{
			"name": "Frodo",
			"age":  51.0,
			"items": []interface{}{
				"ring",
				"cloak",
			},
		},
	}
	jsonDataList, _ := json.Marshal(testDataList)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/upload", bytes.NewBuffer(jsonDataList))
	router.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &response)
	uuid = response["uuid"]

	// Reuse the WebSocket connection for the new UUID
	wsURL = "ws" + server.URL[4:] + "/ws/" + uuid
	ws, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer ws.Close()

	// Test the second update with list modification
	sendAndVerifyUpdate(
		handlers.Message{
			Path:  "character.items.1",
			Value: "sword",
		},
		map[string]interface{}{
			"character": map[string]interface{}{
				"name": "Frodo",
				"age":  51.0,
				"items": []interface{}{
					"ring",
					"sword",
				},
			},
		},
	)
}

func TestInvalidUUID(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/json/invalid-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWebSocketInvalidUUID(t *testing.T) {
	router := setupRouter()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r)
	}))
	defer server.Close()

	// Attempt to establish a WebSocket connection with an invalid UUID
	wsURL := "ws" + server.URL[4:] + "/ws/invalid-uuid"
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.Error(t, err)
}
