package services

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type ValidationService struct {
    emailAPIKey string
    phoneAPIKey string
    client      *http.Client
}

// Email validation response struct matching the actual API response
type EmailValidationResponse struct {
    Email           string                `json:"email"`
    Autocorrect     string                `json:"autocorrect"`
    Deliverability  string                `json:"deliverability"`
    QualityScore    string                `json:"quality_score"`
    IsValidFormat   EmailValidationDetail `json:"is_valid_format"`
    IsFreeEmail     EmailValidationDetail `json:"is_free_email"`
    IsDisposable    EmailValidationDetail `json:"is_disposable_email"`
    IsRoleEmail     EmailValidationDetail `json:"is_role_email"`
    IsCatchall      EmailValidationDetail `json:"is_catchall_email"`
    IsMxFound       EmailValidationDetail `json:"is_mx_found"`
    IsSmtpValid     EmailValidationDetail `json:"is_smtp_valid"`
}

type EmailValidationDetail struct {
    Value bool   `json:"value"`
    Text  string `json:"text"`
}

// Phone validation response struct matching the actual API response
type PhoneValidationResponse struct {
    Phone    string       `json:"phone"`
    Valid    bool         `json:"valid"`
    Format   PhoneFormat  `json:"format"`
    Country  PhoneCountry `json:"country"`
    Location string       `json:"location"` // This is a string, not an object
    Type     string       `json:"type"`
    Carrier  string       `json:"carrier"`
}

type PhoneFormat struct {
    International string `json:"international"`
    Local         string `json:"local"`
}

type PhoneCountry struct {
    Code   string `json:"code"`
    Name   string `json:"name"`
    Prefix string `json:"prefix"`
}

func NewValidationService(emailAPIKey, phoneAPIKey string) *ValidationService {
    return &ValidationService{
        emailAPIKey: emailAPIKey,
        phoneAPIKey: phoneAPIKey,
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (v *ValidationService) ValidateEmail(email string) (*EmailValidationResponse, error) {
    url := fmt.Sprintf("https://emailvalidation.abstractapi.com/v1/?api_key=%s&email=%s", 
        v.emailAPIKey, email)
    
    resp, err := v.client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to make email validation request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("email validation API returned status: %d", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read email validation response: %w", err)
    }

    var result EmailValidationResponse
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("failed to parse email validation response: %w", err)
    }

    return &result, nil
}

func (v *ValidationService) ValidatePhone(phone string) (*PhoneValidationResponse, error) {
    url := fmt.Sprintf("https://phonevalidation.abstractapi.com/v1/?api_key=%s&phone=%s", 
        v.phoneAPIKey, phone)
    
    resp, err := v.client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to make phone validation request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("phone validation API returned status: %d", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read phone validation response: %w", err)
    }

    var result PhoneValidationResponse
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("failed to parse phone validation response: %w", err)
    }

    return &result, nil
}

func (v *ValidationService) IsEmailValid(email string) (bool, error) {
    result, err := v.ValidateEmail(email)
    if err != nil {
        return false, err
    }

    // Validation logic using the correct field names and structure
    isValid := result.IsValidFormat.Value &&      // Must have valid format
               !result.IsDisposable.Value &&      // No disposable emails
               !result.IsRoleEmail.Value &&       // No role-based emails  
               result.IsMxFound.Value &&          // MX record must exist
               result.IsSmtpValid.Value &&        // SMTP must be valid
               result.Deliverability == "DELIVERABLE" // Must be deliverable

    return isValid, nil
}

func (v *ValidationService) IsPhoneValid(phone string) (bool, error) {
    result, err := v.ValidatePhone(phone)
    if err != nil {
        return false, err
    }

    return result.Valid, nil
}

// Optional: Add helper methods to get detailed validation info
func (v *ValidationService) GetEmailValidationDetails(email string) (*EmailValidationResponse, error) {
    return v.ValidateEmail(email)
}

func (v *ValidationService) GetPhoneValidationDetails(phone string) (*PhoneValidationResponse, error) {
    return v.ValidatePhone(phone)
}