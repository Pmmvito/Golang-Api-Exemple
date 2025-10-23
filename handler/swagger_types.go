package handler

// AuthSuccessResponse representa o envelope padrão de sucesso para autenticação.
type AuthSuccessResponse struct {
	Message string       `json:"message"`
	Data    AuthResponse `json:"data"`
}

// UserProfileSuccess representa o retorno de sucesso do endpoint /auth/me.
type UserProfileSuccess struct {
	Message string       `json:"message"`
	Data    UserResponse `json:"data"`
}

// CategoryListSuccess representa a listagem de categorias embrulhada no padrão APISuccess.
type CategoryListSuccess struct {
	Message string             `json:"message"`
	Data    []CategoryResponse `json:"data"`
}

// CategoryItemSuccess representa respostas com uma única categoria.
type CategoryItemSuccess struct {
	Message string           `json:"message"`
	Data    CategoryResponse `json:"data"`
}

// ExpenseItemSuccess representa respostas com uma única despesa.
type ExpenseItemSuccess struct {
	Message string          `json:"message"`
	Data    ExpenseResponse `json:"data"`
}

// ExpenseListSuccess representa a listagem de despesas com resumo agregado.
type ExpenseListSuccess struct {
	Message string               `json:"message"`
	Data    ExpensesListResponse `json:"data"`
}

// DashboardSummarySuccess representa o retorno do resumo do dashboard.
type DashboardSummarySuccess struct {
	Message string                   `json:"message"`
	Data    DashboardSummaryResponse `json:"data"`
}

// ReceiptScanSuccess representa a resposta do processamento de recibos com OCR.
type ReceiptScanSuccess struct {
	Message string              `json:"message"`
	Data    ReceiptScanResponse `json:"data"`
}

// TipsListSuccess representa a listagem de dicas financeiras.
type TipsListSuccess struct {
	Message string        `json:"message"`
	Data    []TipResponse `json:"data"`
}

// SyncJobSuccess representa o retorno da criação de um job de sincronização.
type SyncJobSuccess struct {
	Message string          `json:"message"`
	Data    SyncJobResponse `json:"data"`
}

// TokenUsageListSuccess representa a listagem de consumo de tokens com totais agregados.
type TokenUsageListSuccess struct {
	Message string                 `json:"message"`
	Data    TokenUsageListResponse `json:"data"`
}
