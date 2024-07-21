package main

type FormInfo struct {
	ID              string           `json:"id"`
	Type            string           `json:"type"`
	Title           string           `json:"title"`
	Workspace       Theme            `json:"workspace"`
	Theme           Theme            `json:"theme"`
	Settings        Settings         `json:"settings"`
	ThankyouScreens []ThankyouScreen `json:"thankyou_screens"`
	WelcomeScreens  []WelcomeScreen  `json:"welcome_screens"`
	Fields          []FormInfoField  `json:"fields"`
	CreatedAt       string           `json:"created_at"`
	LastUpdatedAt   string           `json:"last_updated_at"`
	PublishedAt     string           `json:"published_at"`
	Links           Links            `json:"_links"`
}

type FormInfoField struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Ref         string             `json:"ref"`
	Properties  Properties         `json:"properties"`
	Type        string             `json:"type"`
	Validations *FluffyValidations `json:"validations,omitempty"`
}

type Properties struct {
	Fields                 []PropertiesField `json:"fields,omitempty"`
	DefaultCountryCode     *string           `json:"default_country_code,omitempty"`
	ButtonText             *string           `json:"button_text,omitempty"`
	HideMarks              *bool             `json:"hide_marks,omitempty"`
	Randomize              *bool             `json:"randomize,omitempty"`
	AllowMultipleSelection *bool             `json:"allow_multiple_selection,omitempty"`
	AllowOtherChoice       *bool             `json:"allow_other_choice,omitempty"`
	VerticalAlignment      *bool             `json:"vertical_alignment,omitempty"`
	Choices                []Choice          `json:"choices,omitempty"`
	AlphabeticalOrder      *bool             `json:"alphabetical_order,omitempty"`
	Supersized             *bool             `json:"supersized,omitempty"`
	ShowLabels             *bool             `json:"show_labels,omitempty"`
	Separator              *string           `json:"separator,omitempty"`
	Structure              *string           `json:"structure,omitempty"`
	Steps                  *int64            `json:"steps,omitempty"`
	StartAtOne             *bool             `json:"start_at_one,omitempty"`
	Shape                  *string           `json:"shape,omitempty"`
}

type Choice struct {
	ID         string      `json:"id"`
	Ref        string      `json:"ref"`
	Label      string      `json:"label"`
	Attachment *Attachment `json:"attachment,omitempty"`
}

type Attachment struct {
	Type string `json:"type"`
	Href string `json:"href"`
}

type PropertiesField struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Ref         string            `json:"ref"`
	SubfieldKey *string           `json:"subfield_key,omitempty"`
	Properties  FluffyProperties  `json:"properties"`
	Validations PurpleValidations `json:"validations"`
	Type        string            `json:"type"`
}

type FluffyProperties struct {
	DefaultCountryCode     *string  `json:"default_country_code,omitempty"`
	Randomize              *bool    `json:"randomize,omitempty"`
	AllowMultipleSelection *bool    `json:"allow_multiple_selection,omitempty"`
	AllowOtherChoice       *bool    `json:"allow_other_choice,omitempty"`
	VerticalAlignment      *bool    `json:"vertical_alignment,omitempty"`
	Choices                []Choice `json:"choices,omitempty"`
}

type PurpleValidations struct {
	Required  bool   `json:"required"`
	MaxLength *int64 `json:"max_length,omitempty"`
}

type FluffyValidations struct {
	Required bool `json:"required"`
}

type Links struct {
	Display   string `json:"display"`
	Responses string `json:"responses"`
}

type Settings struct {
	Language                string       `json:"language"`
	ProgressBar             string       `json:"progress_bar"`
	Meta                    Meta         `json:"meta"`
	HideNavigation          bool         `json:"hide_navigation"`
	IsPublic                bool         `json:"is_public"`
	IsTrial                 bool         `json:"is_trial"`
	ShowProgressBar         bool         `json:"show_progress_bar"`
	ShowTypeformBranding    bool         `json:"show_typeform_branding"`
	AreUploadsPublic        bool         `json:"are_uploads_public"`
	ShowTimeToComplete      bool         `json:"show_time_to_complete"`
	ShowNumberOfSubmissions bool         `json:"show_number_of_submissions"`
	ShowCookieConsent       bool         `json:"show_cookie_consent"`
	ShowQuestionNumber      bool         `json:"show_question_number"`
	ShowKeyHintOnChoices    bool         `json:"show_key_hint_on_choices"`
	AutosaveProgress        bool         `json:"autosave_progress"`
	FreeFormNavigation      bool         `json:"free_form_navigation"`
	UseLeadQualification    bool         `json:"use_lead_qualification"`
	ProSubdomainEnabled     bool         `json:"pro_subdomain_enabled"`
	Capabilities            Capabilities `json:"capabilities"`
}

type Capabilities struct {
	E2EEncryption E2EEncryption `json:"e2e_encryption"`
}

type E2EEncryption struct {
	Enabled    bool `json:"enabled"`
	Modifiable bool `json:"modifiable"`
}

type Meta struct {
	AllowIndexing bool `json:"allow_indexing"`
}

type ThankyouScreen struct {
	ID         string                   `json:"id"`
	Ref        string                   `json:"ref"`
	Title      string                   `json:"title"`
	Type       string                   `json:"type"`
	Properties ThankyouScreenProperties `json:"properties"`
	Attachment *Attachment              `json:"attachment,omitempty"`
}

type ThankyouScreenProperties struct {
	ShowButton  bool    `json:"show_button"`
	ShareIcons  bool    `json:"share_icons"`
	ButtonMode  string  `json:"button_mode"`
	ButtonText  string  `json:"button_text"`
	Description *string `json:"description,omitempty"`
}

type Theme struct {
	Href string `json:"href"`
}

type WelcomeScreen struct {
	ID         string                  `json:"id"`
	Ref        string                  `json:"ref"`
	Title      string                  `json:"title"`
	Properties WelcomeScreenProperties `json:"properties"`
}

type WelcomeScreenProperties struct {
	ShowButton bool   `json:"show_button"`
	ButtonText string `json:"button_text"`
}

type Responses struct {
	Items      []Item `json:"items"`
	TotalItems int64  `json:"total_items"`
	PageCount  int64  `json:"page_count"`
}

type Item struct {
	LandingID    string     `json:"landing_id"`
	Token        string     `json:"token"`
	ResponseID   string     `json:"response_id"`
	ResponseType string     `json:"response_type"`
	LandedAt     string     `json:"landed_at"`
	SubmittedAt  string     `json:"submitted_at"`
	Metadata     Metadata   `json:"metadata"`
	Hidden       Hidden     `json:"hidden"`
	Calculated   Calculated `json:"calculated"`
	Answers      []Answer   `json:"answers"`
}

type Answer struct {
	Field       Field      `json:"field"`
	Type        string     `json:"type"`
	Text        *string    `json:"text,omitempty"`
	Email       *string    `json:"email,omitempty"`
	PhoneNumber *string    `json:"phone_number,omitempty"`
	URL         *string    `json:"url,omitempty"`
	Choice      *ChoiceRes `json:"choice,omitempty"`
	Boolean     *bool      `json:"boolean,omitempty"`
	Number      *int64     `json:"number,omitempty"`
	Date        *string    `json:"date,omitempty"`
	Choices     *Choices   `json:"choices,omitempty"`
}

type ChoiceRes struct {
	ID    string `json:"id"`
	Ref   string `json:"ref"`
	Label string `json:"label"`
}

type Choices struct {
	IDS    []string `json:"ids"`
	Refs   []string `json:"refs"`
	Labels []string `json:"labels"`
}

type Field struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Ref  string `json:"ref"`
}

type Calculated struct {
	Score int64 `json:"score"`
}

type Hidden struct {
}

type Metadata struct {
	UserAgent string `json:"user_agent"`
	Platform  string `json:"platform"`
	Referer   string `json:"referer"`
	NetworkID string `json:"network_id"`
	Browser   string `json:"browser"`
}
