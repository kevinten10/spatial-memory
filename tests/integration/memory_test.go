//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCRUD(t *testing.T) {
	defer suite.CleanupTestData(t)

	_, token := suite.CreateTestUser(t, "+8613800138100")

	t.Run("Create Memory", func(t *testing.T) {
		body := map[string]interface{}{
			"title":       "Test Memory",
			"content":     "This is a test memory",
			"location":    map[string]float64{"lat": 39.9042, "lng": 116.4074},
			"address":     "Beijing, China",
			"visibility":  2, // Public
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/memories", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "Test Memory")
	})

	t.Run("Create Memory - Invalid Location", func(t *testing.T) {
		body := map[string]interface{}{
			"title":    "Invalid Memory",
			"location": map[string]float64{"lat": 999, "lng": 999},
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/memories", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("List My Memories", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/memories/mine", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "items")
	})

	t.Run("Get Memory by ID", func(t *testing.T) {
		// First create a memory
		body := map[string]interface{}{
			"title":      "Memory to Get",
			"content":    "Content",
			"location":   map[string]float64{"lat": 39.9042, "lng": 116.4074},
			"visibility": 2,
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/memories", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		// Extract memory ID from response
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok)
		memoryID := int64(data["id"].(float64))

		// Get the memory
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/memories/%d", memoryID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Memory to Get")
	})

	t.Run("Update Memory", func(t *testing.T) {
		// Create a memory first
		body := map[string]interface{}{
			"title":      "Memory to Update",
			"content":    "Original content",
			"location":   map[string]float64{"lat": 39.9042, "lng": 116.4074},
			"visibility": 2,
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/memories", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		data := response["data"].(map[string]interface{})
		memoryID := int64(data["id"].(float64))

		// Update the memory
		updateBody := map[string]string{"title": "Updated Title"}
		jsonBody, _ = json.Marshal(updateBody)

		w = httptest.NewRecorder()
		req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/memories/%d", memoryID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Updated Title")
	})

	t.Run("Delete Memory", func(t *testing.T) {
		// Create a memory first
		body := map[string]interface{}{
			"title":      "Memory to Delete",
			"content":    "Content",
			"location":   map[string]float64{"lat": 39.9042, "lng": 116.4074},
			"visibility": 2,
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/memories", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		data := response["data"].(map[string]interface{})
		memoryID := int64(data["id"].(float64))

		// Delete the memory
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/v1/memories/%d", memoryID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "deleted")
	})
}

func TestNearbyQuery(t *testing.T) {
	defer suite.CleanupTestData(t)

	_, token := suite.CreateTestUser(t, "+8613800138200")

	// Create multiple memories at different locations
	locations := []struct {
		lat   float64
		lng   float64
		title string
	}{
		{39.9042, 116.4074, "Memory at Beijing Center"},
		{39.9142, 116.4174, "Memory nearby 1"},
		{39.9242, 116.4274, "Memory nearby 2"},
		{39.8042, 116.3074, "Memory far away"},
	}

	for _, loc := range locations {
		body := map[string]interface{}{
			"title":      loc.title,
			"content":    "Test content",
			"location":   map[string]float64{"lat": loc.lat, "lng": loc.lng},
			"visibility": 2, // Public
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/memories", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
	}

	t.Run("Nearby Query - Default Radius", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/memories/nearby?lat=39.9042&lng=116.4074", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Should find nearby memories
		assert.Contains(t, w.Body.String(), "Memory")
	})

	t.Run("Nearby Query - Custom Radius", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/memories/nearby?lat=39.9042&lng=116.4074&radius=5000", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Nearby Query - Sort by Recent", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/memories/nearby?lat=39.9042&lng=116.4074&sort=recent", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Nearby Query - Invalid Coordinates", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/memories/nearby?lat=999&lng=999", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Nearby Query - Missing Coordinates", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/memories/nearby", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestMemoryInteractions(t *testing.T) {
	defer suite.CleanupTestData(t)

	_, token := suite.CreateTestUser(t, "+8613800138300")

	// Create a memory
	body := map[string]interface{}{
		"title":      "Memory for Interactions",
		"content":    "Test content",
		"location":   map[string]float64{"lat": 39.9042, "lng": 116.4074},
		"visibility": 2,
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/memories", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	memoryID := int64(data["id"].(float64))

	t.Run("Like Memory", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/memories/%d/like", memoryID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "liked")
	})

	t.Run("Unlike Memory", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/memories/%d/like", memoryID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Bookmark Memory", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/memories/%d/bookmark", memoryID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "bookmarked")
	})

	t.Run("Unbookmark Memory", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/memories/%d/bookmark", memoryID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Report Memory", func(t *testing.T) {
		reportBody := map[string]string{"reason": "Inappropriate content"}
		jsonBody, _ := json.Marshal(reportBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/memories/%d/report", memoryID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
