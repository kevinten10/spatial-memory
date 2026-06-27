//go:build ark_smoke

package moderation

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/spatial-memory/spatial-memory/internal/config"
)

func TestArkModerationSmoke(t *testing.T) {
	apiKey := os.Getenv("SPATIAL_ARK_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ARK_API_KEY")
	}
	if apiKey == "" {
		t.Skip("SPATIAL_ARK_API_KEY or ARK_API_KEY is not configured")
	}

	cfg := config.ArkConfig{
		APIKey:      apiKey,
		BaseURL:     envOrDefault("SPATIAL_ARK_BASE_URL", defaultBaseURL),
		ChatModel:   envOrDefault("SPATIAL_ARK_CHAT_MODEL", defaultChatModel),
		VisionModel: envOrDefault("SPATIAL_ARK_VISION_MODEL", defaultVisionModel),
		Timeout:     60 * time.Second,
	}

	client := NewClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	textResult, err := client.ModerateText(ctx, "今天和朋友在公园散步，天气很好。")
	if err != nil {
		t.Fatalf("ModerateText failed: %v", err)
	}
	if textResult.Confidence <= 0 {
		t.Fatalf("ModerateText returned invalid confidence: %+v", textResult)
	}

	imageResult, err := client.ModerateImage(ctx, smokePNGDataURL(t))
	if err != nil {
		t.Fatalf("ModerateImage failed: %v", err)
	}
	if imageResult.Confidence <= 0 {
		t.Fatalf("ModerateImage returned invalid confidence: %+v", imageResult)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func smokePNGDataURL(t *testing.T) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 128, G: 180, B: 220, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode smoke png: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}
