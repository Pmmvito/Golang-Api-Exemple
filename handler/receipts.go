package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/Pmmvito/Golang-Api-Exemple/service/gemini"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type receiptLLMItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Total       float64 `json:"total"`
}

type receiptLLMResult struct {
	Total      float64          `json:"total"`
	Currency   string           `json:"currency"`
	Confidence float64          `json:"confidence"`
	Date       string           `json:"date"`
	Items      []receiptLLMItem `json:"items"`
	RawText    string           `json:"raw_text"`
	RawTextAlt string           `json:"rawText"`
	Notes      string           `json:"notes"`
}

// ScanReceiptHandler godoc
// @Summary Processar recibo com OCR
// @Description Analisa uma imagem em Base64 usando Gemini e retorna extrações estruturadas
// @Tags Recibos
// @Security Bearer
// @Accept json
// @Produce json
// @Param body body ReceiptScanRequest true "Dados do recibo"
// @Success 200 {object} ReceiptScanResponse
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 500 {object} APIError
// @Router /receipts/scan [post]
func ScanReceiptHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	var request ReceiptScanRequest
	if !bindJSON(ctx, &request) {
		return
	}

	rawImage := strings.TrimSpace(request.ImageBase64)
	if rawImage == "" {
		respondError(ctx, 400, "imagem é obrigatória", nil)
		return
	}

	mimeType, payload := extractMimeAndPayload(rawImage)
	if payload == "" {
		respondError(ctx, 400, "imagem inválida", "payload base64 vazio")
		return
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		respondError(ctx, 400, "imagem inválida", err.Error())
		return
	}

	currency := strings.ToUpper(strings.TrimSpace(request.Currency))
	if currency == "" && user.Config != nil {
		currency = strings.ToUpper(strings.TrimSpace(user.Config.Currency))
	}
	if currency == "" {
		currency = "BRL"
	}

	locale := strings.TrimSpace(request.Locale)
	if locale == "" && user.Config != nil {
		locale = strings.TrimSpace(user.Config.Language)
	}
	if locale == "" {
		locale = "pt-BR"
	}

	client, err := gemini.NewClientFromEnv()
	if err != nil {
		getLogger().ErrorF("erro ao configurar cliente gemini: %v", err)
		fallback := buildFallbackResponse(len(data), currency, request.AmountHint)
		respondSuccess(ctx, "recebido", fallback)
		return
	}

	prompt := buildReceiptPrompt(currency, locale, request.AmountHint)
	model := detectModelName()

	geminiRequest := gemini.GenerateContentRequest{
		Contents: []gemini.Content{
			{
				Role: "user",
				Parts: []gemini.ContentPart{
					gemini.NewTextPart(prompt),
					gemini.NewInlineImagePart(mimeType, payload),
				},
			},
		},
	}

	ctxTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 45*time.Second)
	defer cancel()

	result, err := client.GenerateContent(ctxTimeout, geminiRequest)
	if err != nil {
		getLogger().WarnF("falha ao gerar conteúdo com gemini: %v", err)
		fallback := buildFallbackResponse(len(data), currency, request.AmountHint)
		fallback.Model = model
		respondSuccess(ctx, "recebido", fallback)
		return
	}

	sanitized := gemini.SanitizeJSON(result.Text)
	llmResult, parseErr := parseReceiptLLMResult(sanitized)
	if parseErr != nil {
		getLogger().WarnF("não foi possível interpretar resposta do gemini: %v", parseErr)
	}

	response := buildResponseFromLLM(llmResult, len(data), currency, request.AmountHint)
	response.Model = model
	response.TokensUsed = result.Usage.TotalTokenCount

	metadata := datatypes.JSONMap{
		"currency":      response.Currency,
		"locale":        locale,
		"mimeType":      mimeType,
		"itemsDetected": len(response.Items),
		"returnRaw":     request.ReturnRaw,
		"hasAmountHint": request.AmountHint != nil,
		"model":         model,
	}

	if request.ReturnRaw {
		response.RawModelOutput = sanitized
	}

	savedExpense, persistErr := persistReceiptData(ctx.Request.Context(), user, &response, sanitized)
	if persistErr != nil {
		respondError(ctx, 500, "não foi possível salvar o recibo", persistErr.Error())
		return
	}

	if savedExpense != nil {
		recorded, loadErr := loadExpenseForResponse(ctx.Request.Context(), savedExpense.ID)
		if loadErr != nil {
			getLogger().WarnF("não foi possível carregar despesa salva: %v", loadErr)
		} else {
			response.SavedExpense = toExpenseResponse(recorded)
		}
		metadata["expenseId"] = savedExpense.ID.String()
	}

	recordCtx, cancelRecord := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancelRecord()

	entry, logErr := recordTokenUsage(recordCtx, user.ID, schemas.RequestTypeReceipt, result.Usage, metadata)
	if logErr != nil {
		getLogger().WarnF("não foi possível registrar uso de tokens: %v", logErr)
	} else if entry != nil {
		response.TokenCostCents = entry.CostInCents
	}

	respondSuccess(ctx, "recebido", response)
}

func buildResponseFromLLM(result *receiptLLMResult, imageSize int, currency string, amountHint *float64) ReceiptScanResponse {
	if result == nil {
		fallback := buildFallbackResponse(imageSize, currency, amountHint)
		return fallback
	}

	suggestedAmount := result.Total
	if suggestedAmount <= 0 && amountHint != nil && *amountHint > 0 {
		suggestedAmount = *amountHint
	}
	if suggestedAmount <= 0 {
		suggestedAmount = calculateHeuristicAmount(imageSize)
	}

	confidence := clampConfidence(result.Confidence)
	if confidence <= 0 {
		confidence = calculateHeuristicConfidence(imageSize)
	}

	extractedText := strings.TrimSpace(result.RawText)
	if extractedText == "" {
		extractedText = strings.TrimSpace(result.RawTextAlt)
	}
	if extractedText == "" {
		extractedText = strings.TrimSpace(result.Notes)
	}

	parsedDate := chooseDate(result.Date)

	items := make([]ReceiptItem, 0, len(result.Items))
	for _, item := range result.Items {
		if strings.TrimSpace(item.Description) == "" {
			continue
		}
		items = append(items, ReceiptItem{
			Description: strings.TrimSpace(item.Description),
			Quantity:    roundFloat(item.Quantity),
			UnitPrice:   roundFloat(item.UnitPrice),
			Total:       roundFloat(item.Total),
		})
	}

	detectedCurrency := strings.ToUpper(strings.TrimSpace(result.Currency))
	if detectedCurrency == "" {
		detectedCurrency = currency
	}

	return ReceiptScanResponse{
		SuggestedAmount: roundFloat(suggestedAmount),
		SuggestedDate:   parsedDate,
		Currency:        detectedCurrency,
		ExtractedText:   extractedText,
		Items:           items,
		Confidence:      roundFloat(confidence),
	}
}

func buildFallbackResponse(imageSize int, currency string, amountHint *float64) ReceiptScanResponse {
	fallbackAmount := calculateHeuristicAmount(imageSize)
	if amountHint != nil && *amountHint > 0 {
		fallbackAmount = roundFloat(*amountHint)
	}

	return ReceiptScanResponse{
		SuggestedAmount: fallbackAmount,
		SuggestedDate:   time.Now().Format("2006-01-02"),
		Currency:        currency,
		ExtractedText:   "",
		Items:           []ReceiptItem{},
		Confidence:      roundFloat(calculateHeuristicConfidence(imageSize)),
		TokensUsed:      0,
		TokenCostCents:  0,
	}
}

func parseReceiptLLMResult(raw string) (*receiptLLMResult, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("resposta vazia do modelo")
	}

	var result receiptLLMResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func buildReceiptPrompt(currency, locale string, amountHint *float64) string {
	var builder strings.Builder
	builder.WriteString("Você é um assistente de finanças que extrai dados estruturados de recibos em imagem.\n")
	builder.WriteString("Retorne apenas JSON, sem comentários nem texto adicional.\n")
	builder.WriteString("Formato esperado:\n")
	builder.WriteString("{" +
		"\"total\": number, \"currency\": \"" + currency + "\", \"confidence\": number entre 0 e 1, \"date\": \"YYYY-MM-DD\", \"items\": [ {\"description\": string, \"quantity\": number, \"unitPrice\": number, \"total\": number} ], \"raw_text\": string, \"notes\": string }\n")
	builder.WriteString("Se algum valor não estiver presente, use null ou string vazia.\n")
	builder.WriteString("Use ponto como separador decimal.\n")
	builder.WriteString("Interprete quantias na moeda " + currency + " e utilize o formato de data " + locale + " convertendo para YYYY-MM-DD.\n")
	if amountHint != nil && *amountHint > 0 {
		builder.WriteString(fmt.Sprintf("O total esperado aproximado é %.2f %s. Utilize isso apenas como referência ao validar o valor extraído.\n", *amountHint, currency))
	}
	builder.WriteString("Mantenha a chave currency em letras maiúsculas.\n")
	return builder.String()
}

func extractMimeAndPayload(raw string) (string, string) {
	mimeType := "image/jpeg"
	payload := raw

	if idx := strings.Index(raw, ","); idx != -1 {
		prefix := raw[:idx]
		payload = raw[idx+1:]

		if strings.HasPrefix(prefix, "data:") {
			if semi := strings.Index(prefix, ";"); semi != -1 {
				mimeType = strings.TrimSpace(prefix[len("data:"):semi])
			} else {
				mimeType = strings.TrimSpace(prefix[len("data:"):])
			}
		}
	}

	return mimeType, strings.TrimSpace(payload)
}

func calculateHeuristicAmount(imageSize int) float64 {
	factor := math.Max(float64(imageSize)/1000.0, 1)
	return roundFloat(19.75 * factor)
}

func calculateHeuristicConfidence(imageSize int) float64 {
	confidence := 0.7 + math.Min(float64(imageSize)/12000.0, 0.2)
	if confidence > 0.95 {
		confidence = 0.95
	}
	return confidence
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func chooseDate(value string) string {
	layouts := []string{"2006-01-02", time.RFC3339, "02/01/2006", "02-01-2006"}
	trimmed := strings.TrimSpace(value)
	for _, layout := range layouts {
		if trimmed == "" {
			break
		}
		if t, err := time.Parse(layout, trimmed); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return time.Now().Format("2006-01-02")
}

func detectModelName() string {
	if model := os.Getenv("GEMINI_MODEL"); model != "" {
		return model
	}
	if model := os.Getenv("EXPO_PUBLIC_GEMINI_MODEL"); model != "" {
		return model
	}
	return "gemini-2.5-flash-preview-05-20"
}

const defaultOcrCategoryName = "Compras OCR"

func persistReceiptData(ctx context.Context, user *schemas.User, payload *ReceiptScanResponse, rawModel string) (*schemas.Expense, error) {
	if user == nil || payload == nil {
		return nil, fmt.Errorf("dados insuficientes para persistir recibo")
	}

	var savedExpense schemas.Expense

	err := getDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		category, err := ensureOcrCategory(ctx, tx, user)
		if err != nil {
			return err
		}

		parsedDate, err := time.Parse("2006-01-02", payload.SuggestedDate)
		if err != nil {
			parsedDate = time.Now()
		}

		description := fmt.Sprintf("Compra no mercado (%s)", parsedDate.Format("02/01"))
		amount := payload.SuggestedAmount
		if amount < 0 {
			amount = 0
		}

		expense := schemas.Expense{
			UserID:      user.ID,
			CategoryID:  category.ID,
			Description: description,
			Amount:      roundFloat(amount),
			Date:        parsedDate,
			Origin:      schemas.ExpenseOriginOCR,
		}

		if err := tx.Create(&expense).Error; err != nil {
			return err
		}

		for _, item := range payload.Items {
			name := strings.TrimSpace(item.Description)
			if name == "" {
				continue
			}
			i := schemas.ExpenseItem{
				ExpenseID:  expense.ID,
				Name:       name,
				Quantity:   roundFloat(item.Quantity),
				UnitPrice:  roundFloat(item.UnitPrice),
				TotalPrice: roundFloat(item.Total),
			}
			if err := tx.Create(&i).Error; err != nil {
				return err
			}
		}

		receiptText := strings.TrimSpace(payload.ExtractedText)
		if receiptText == "" {
			receiptText = strings.TrimSpace(rawModel)
		}

		receipt := schemas.Receipt{
			ExpenseID:     expense.ID,
			ExtractedText: receiptText,
			OcrConfidence: payload.Confidence,
		}
		if err := tx.Create(&receipt).Error; err != nil {
			return err
		}

		savedExpense = expense
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &savedExpense, nil
}

func ensureOcrCategory(ctx context.Context, tx *gorm.DB, user *schemas.User) (*schemas.Category, error) {
	category := schemas.Category{}
	if err := tx.WithContext(ctx).
		Where("user_id = ? AND name = ?", user.ID, defaultOcrCategoryName).
		First(&category).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}

		category = schemas.Category{
			UserID:   user.ID,
			Name:     defaultOcrCategoryName,
			Icon:     "shopping_cart",
			ColorHex: "#2E7D32",
			Type:     schemas.CategoryTypeVariable,
			Order:    999,
			Active:   true,
		}
		if err := tx.Create(&category).Error; err != nil {
			return nil, err
		}
	}
	return &category, nil
}

func loadExpenseForResponse(ctx context.Context, expenseID uuid.UUID) (*schemas.Expense, error) {
	var expense schemas.Expense
	if err := getDB().WithContext(ctx).
		Preload("Category").
		Preload("Receipt").
		Preload("Items").
		First(&expense, "id = ?", expenseID).Error; err != nil {
		return nil, err
	}
	return &expense, nil
}
