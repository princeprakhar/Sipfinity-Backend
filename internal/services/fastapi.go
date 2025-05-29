package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"github.com/princeprakhar/ecommerce-backend/internal/config"
)

type FastAPIService struct {
	config *config.Config
}

type FastAPIResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ExcelPath   string `json:"excel_path"`
	ProductData []ProductData `json:"product_data"`
}

type ProductData struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	Category    string   `json:"category"`
	Brand       string   `json:"brand"`
	SKU         string   `json:"sku"`
	Images      []string `json:"images"`
}

func NewFastAPIService(config *config.Config) *FastAPIService {
	return &FastAPIService{config: config}
}

func (s *FastAPIService) ProcessImages(images []string) (*FastAPIResponse, error) {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add images to form
	for _, imagePath := range images {
		file, err := os.Open(imagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open image %s: %v", imagePath, err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile(fmt.Sprintf("images"), filepath.Base(imagePath))
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %v", err)
		}

		_, err = io.Copy(part, file)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file content: %v", err)
		}
	}

	writer.Close()

	// Create request
	url := fmt.Sprintf("%s/upload/images", s.config.FastAPIURL)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Internal-API-Key", s.config.FastAPIKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var fastAPIResp FastAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&fastAPIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FastAPI error: %s", fastAPIResp.Message)
	}

	return &fastAPIResp, nil
}