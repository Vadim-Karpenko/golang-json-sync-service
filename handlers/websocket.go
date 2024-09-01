package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Store active clients for each room
var roomClients = make(map[string]map[*websocket.Conn]bool)
var mu sync.Mutex

func HandleWebSocket(c *gin.Context, rdb *redis.Client) {
	uuid := c.Param("uuid")

	// Check if UUID exists in Redis
	_, err := rdb.Get(context.Background(), uuid).Result()
	if err == redis.Nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "UUID not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	// Add client to the room
	mu.Lock()
	if roomClients[uuid] == nil {
		roomClients[uuid] = make(map[*websocket.Conn]bool)
	}
	roomClients[uuid][conn] = true
	mu.Unlock()

	defer func() {
		// Remove client from the room on disconnect
		mu.Lock()
		delete(roomClients[uuid], conn)
		mu.Unlock()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read failed:", err)
			break
		}

		// Parse the received message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("Invalid message format:", err)
			continue
		}

		// Apply the update to the JSON in Redis
		applyJSONUpdate(rdb, uuid, msg)

		// Broadcast the update to all clients in the room
		broadcastUpdate(uuid, message)
	}
}

func applyJSONUpdate(rdb *redis.Client, uuid string, msg Message) {
	// Get the current JSON data from Redis
	val, err := rdb.Get(context.Background(), uuid).Result()
	if err != nil && err != redis.Nil {
		log.Println("Error retrieving JSON from Redis:", err)
		return
	}

	var jsonData interface{}
	if err := json.Unmarshal([]byte(val), &jsonData); err != nil {
		log.Println("Error unmarshalling JSON:", err)
		return
	}

	// Update the JSON data according to the path in the Message
	updateJSON(&jsonData, msg.Path, msg.Value)

	// Save the updated JSON back to Redis
	jsonStr, err := json.Marshal(jsonData)
	if err != nil {
		log.Println("Error marshalling JSON:", err)
		return
	}

	if err := rdb.Set(context.Background(), uuid, jsonStr, 30*24*3600*time.Second).Err(); err != nil {
		log.Println("Error saving JSON to Redis:", err)
	}
}

func updateJSON(data *interface{}, path string, value interface{}) {
	keys := strings.Split(path, ".")
	lastKey := keys[len(keys)-1]

	// Navigate the JSON structure according to the keys
	for _, key := range keys[:len(keys)-1] {
		if index, err := strconv.Atoi(key); err == nil {
			// Handle list index
			if list, ok := (*data).([]interface{}); ok {
				if index >= 0 && index < len(list) {
					data = &list[index]
				} else {
					log.Println("List index out of range:", key)
					return
				}
			} else {
				log.Println("Expected list at key:", key)
				return
			}
		} else {
			// Handle map key
			if nestedMap, ok := (*data).(map[string]interface{}); ok {
				if nestedData, ok := nestedMap[key]; ok {
					data = &nestedData
				} else {
					log.Println("Key not found:", key)
					return
				}
			} else {
				log.Println("Invalid data structure at key:", key)
				return
			}
		}
	}

	// Set the final value
	if index, err := strconv.Atoi(lastKey); err == nil {
		// Handle list index for the final key
		if list, ok := (*data).([]interface{}); ok {
			if index >= 0 && index < len(list) {
				list[index] = value
			} else {
				log.Println("List index out of range:", lastKey)
			}
		} else {
			log.Println("Expected list at final key:", lastKey)
		}
	} else {
		// Handle map key for the final key
		if nestedMap, ok := (*data).(map[string]interface{}); ok {
			nestedMap[lastKey] = value
		} else {
			log.Println("Invalid data structure for final key:", lastKey)
		}
	}
}

func broadcastUpdate(uuid string, message []byte) {
	mu.Lock()
	defer mu.Unlock()

	for client := range roomClients[uuid] {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Println("Error broadcasting message:", err)
			client.Close()
			delete(roomClients[uuid], client)
		}
	}
}
