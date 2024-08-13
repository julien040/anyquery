package main

type GraphqlQuery struct {
	Query string `json:"query"`
}

type APIResponse struct {
	Extensions Extensions  `json:"extensions"`
	Errors     interface{} `json:"errors"`
}

type OrderResponse struct {
	Data OrderData `json:"data"`
	APIResponse
}

type ProductsResponse struct {
	Data ProductsData `json:"data"`
	APIResponse
}

type ProductsVariantResponse struct {
	Data ProductsVariantData `json:"data"`
	APIResponse
}

type ProductsVariantData struct {
	ProductVariant ProductVariantClass `json:"productVariants"`
}

type ProductVariantClass struct {
	PageInfo PageInfo         `json:"pageInfo"`
	Nodes    []ProductVariant `json:"nodes"`
}

type ProductsData struct {
	Products ProductsClass `json:"products"`
}

type ProductsClass struct {
	PageInfo PageInfo  `json:"pageInfo"`
	Nodes    []Product `json:"nodes"`
}

type OrderData struct {
	Orders OrdersClass `json:"orders"`
}

type OrdersClass struct {
	PageInfo PageInfo    `json:"pageInfo"`
	Nodes    []OrderNode `json:"nodes"`
}

type CustomerResponse struct {
	Data CustomerData `json:"data"`
	APIResponse
}

type CustomerData struct {
	Customers CustomersClass `json:"customers"`
}

type CustomersClass struct {
	PageInfo PageInfo   `json:"pageInfo"`
	Nodes    []Customer `json:"nodes"`
}

type OrderNode struct {
	ID                       string      `json:"id"`
	Unpaid                   bool        `json:"unpaid"`
	Confirmed                bool        `json:"confirmed"`
	DisplayFinancialStatus   string      `json:"displayFinancialStatus"`
	DisplayFulfillmentStatus string      `json:"displayFulfillmentStatus"`
	Email                    *string     `json:"email"`
	Fulfillable              bool        `json:"fulfillable"`
	FullyPaid                bool        `json:"fullyPaid"`
	Note                     interface{} `json:"note"`
	RequiresShipping         bool        `json:"requiresShipping"`
	TotalWeight              string      `json:"totalWeight"`
	TotalPriceSet            Set         `json:"totalPriceSet"`
	CurrentTotalPriceSet     Set         `json:"currentTotalPriceSet"`
	TotalDiscountsSet        Set         `json:"totalDiscountsSet"`
	ReturnStatus             string      `json:"returnStatus"`
	Name                     string      `json:"name"`
	ProcessedAt              string      `json:"processedAt"`
	CreatedAt                string      `json:"createdAt"`
	UpdatedAt                string      `json:"updatedAt"`
}

type Set struct {
	PresentmentMoney PresentmentMoney `json:"presentmentMoney"`
}

type PresentmentMoney struct {
	Amount string `json:"amount"`
}

type Risk struct {
	Assessments []Assessment `json:"assessments"`
}

type Assessment struct {
	RiskLevel string `json:"riskLevel"`
}

type PageInfo struct {
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	StartCursor     string `json:"startCursor"`
}

type Extensions struct {
	Cost Cost `json:"cost"`
}

type Cost struct {
	RequestedQueryCost int64          `json:"requestedQueryCost"`
	ActualQueryCost    int64          `json:"actualQueryCost"`
	ThrottleStatus     ThrottleStatus `json:"throttleStatus"`
}

type ThrottleStatus struct {
	MaximumAvailable   float64 `json:"maximumAvailable"`
	CurrentlyAvailable float64 `json:"currentlyAvailable"`
	RestoreRate        float64 `json:"restoreRate"`
}

type Product struct {
	ID                    string      `json:"id"`
	Title                 string      `json:"title"`
	Vendor                string      `json:"vendor"`
	ProductType           string      `json:"productType"`
	CreatedAt             string      `json:"createdAt"`
	UpdatedAt             string      `json:"updatedAt"`
	Status                string      `json:"status"`
	Description           string      `json:"description"`
	DescriptionHTML       string      `json:"descriptionHtml"`
	OnlineStoreURL        interface{} `json:"onlineStoreUrl"`
	OnlineStorePreviewURL string      `json:"onlineStorePreviewUrl"`
	TotalInventory        int64       `json:"totalInventory"`
}

type ProductVariant struct {
	AvailableForSale       bool        `json:"availableForSale"`
	Barcode                interface{} `json:"barcode"`
	CreatedAt              string      `json:"createdAt"`
	UpdatedAt              string      `json:"updatedAt"`
	DisplayName            string      `json:"displayName"`
	ID                     string      `json:"id"`
	InventoryQuantity      int64       `json:"inventoryQuantity"`
	Position               int64       `json:"position"`
	Price                  string      `json:"price"`
	Product                Product     `json:"product"`
	SellableOnlineQuantity int64       `json:"sellableOnlineQuantity"`
	Sku                    string      `json:"sku"`
	Title                  string      `json:"title"`
}

type Customer struct {
	AmountSpent             AmountSpent `json:"amountSpent"`
	CreatedAt               string      `json:"createdAt"`
	UpdatedAt               string      `json:"updatedAt"`
	DataSaleOptOut          bool        `json:"dataSaleOptOut"`
	DisplayName             string      `json:"displayName"`
	Email                   string      `json:"email"`
	FirstName               string      `json:"firstName"`
	LastName                string      `json:"lastName"`
	ID                      string      `json:"id"`
	Locale                  string      `json:"locale"`
	Note                    *string     `json:"note"`
	NumberOfOrders          string      `json:"numberOfOrders"`
	Phone                   string      `json:"phone"`
	ProductSubscriberStatus string      `json:"productSubscriberStatus"`
	State                   string      `json:"state"`
	ValidEmailAddress       bool        `json:"validEmailAddress"`
	VerifiedEmail           bool        `json:"verifiedEmail"`
	TaxExempt               bool        `json:"taxExempt"`
	Tags                    []string    `json:"tags"`
}

type AmountSpent struct {
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
}
