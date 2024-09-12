package main

type Boards []Board

type Board struct {
	ID                string        `json:"id"`
	NodeID            string        `json:"nodeId"`
	Name              string        `json:"name"`
	Desc              string        `json:"desc"`
	DescData          interface{}   `json:"descData"`
	Closed            bool          `json:"closed"`
	DateClosed        interface{}   `json:"dateClosed"`
	IDOrganization    string        `json:"idOrganization"`
	IDEnterprise      interface{}   `json:"idEnterprise"`
	Limits            Limits        `json:"limits"`
	Pinned            bool          `json:"pinned"`
	Starred           bool          `json:"starred"`
	URL               string        `json:"url"`
	Prefs             Prefs         `json:"prefs"`
	ShortLink         string        `json:"shortLink"`
	Subscribed        bool          `json:"subscribed"`
	LabelNames        LabelNames    `json:"labelNames"`
	PowerUPS          []interface{} `json:"powerUps"`
	DateLastActivity  *string       `json:"dateLastActivity"`
	DateLastView      string        `json:"dateLastView"`
	ShortURL          string        `json:"shortUrl"`
	IDTags            []interface{} `json:"idTags"`
	DatePluginDisable interface{}   `json:"datePluginDisable"`
	CreationMethod    *string       `json:"creationMethod"`
	IxUpdate          string        `json:"ixUpdate"`
	TemplateGallery   interface{}   `json:"templateGallery"`
	EnterpriseOwned   bool          `json:"enterpriseOwned"`
	IDBoardSource     interface{}   `json:"idBoardSource"`
	PremiumFeatures   []string      `json:"premiumFeatures"`
	IDMemberCreator   string        `json:"idMemberCreator"`
	Type              interface{}   `json:"type"`
	Memberships       []Membership  `json:"memberships"`
}

type LabelNames struct {
	Green       string `json:"green"`
	Yellow      string `json:"yellow"`
	Orange      string `json:"orange"`
	Red         string `json:"red"`
	Purple      string `json:"purple"`
	Blue        string `json:"blue"`
	Sky         string `json:"sky"`
	Lime        string `json:"lime"`
	Pink        string `json:"pink"`
	Black       string `json:"black"`
	GreenDark   string `json:"green_dark"`
	YellowDark  string `json:"yellow_dark"`
	OrangeDark  string `json:"orange_dark"`
	RedDark     string `json:"red_dark"`
	PurpleDark  string `json:"purple_dark"`
	BlueDark    string `json:"blue_dark"`
	SkyDark     string `json:"sky_dark"`
	LimeDark    string `json:"lime_dark"`
	PinkDark    string `json:"pink_dark"`
	BlackDark   string `json:"black_dark"`
	GreenLight  string `json:"green_light"`
	YellowLight string `json:"yellow_light"`
	OrangeLight string `json:"orange_light"`
	RedLight    string `json:"red_light"`
	PurpleLight string `json:"purple_light"`
	BlueLight   string `json:"blue_light"`
	SkyLight    string `json:"sky_light"`
	LimeLight   string `json:"lime_light"`
	PinkLight   string `json:"pink_light"`
	BlackLight  string `json:"black_light"`
}

type Limits struct {
	Attachments        Attachments        `json:"attachments"`
	Boards             BoardsClass        `json:"boards"`
	Cards              CardsLimit         `json:"cards"`
	Checklists         Attachments        `json:"checklists"`
	CheckItems         CheckItems         `json:"checkItems"`
	CustomFields       CustomFields       `json:"customFields"`
	CustomFieldOptions CustomFieldOptions `json:"customFieldOptions"`
	Labels             CustomFields       `json:"labels"`
	Lists              ListsLimits        `json:"lists"`
	Stickers           Stickers           `json:"stickers"`
	Reactions          Reactions          `json:"reactions"`
}

type Attachments struct {
	PerBoard PerBoard `json:"perBoard"`
	PerCard  PerBoard `json:"perCard"`
}

type PerBoard struct {
	Status    string `json:"status"`
	DisableAt int64  `json:"disableAt"`
	WarnAt    int64  `json:"warnAt"`
}

type BoardsClass struct {
	TotalMembersPerBoard        PerBoard `json:"totalMembersPerBoard"`
	TotalAccessRequestsPerBoard PerBoard `json:"totalAccessRequestsPerBoard"`
}

type CardsLimit struct {
	OpenPerBoard  PerBoard `json:"openPerBoard"`
	OpenPerList   PerBoard `json:"openPerList"`
	TotalPerBoard PerBoard `json:"totalPerBoard"`
	TotalPerList  PerBoard `json:"totalPerList"`
}

type CheckItems struct {
	PerChecklist PerBoard `json:"perChecklist"`
}

type CustomFieldOptions struct {
	PerField PerBoard `json:"perField"`
}

type CustomFields struct {
	PerBoard PerBoard `json:"perBoard"`
}

type ListsLimits struct {
	OpenPerBoard  PerBoard `json:"openPerBoard"`
	TotalPerBoard PerBoard `json:"totalPerBoard"`
}

type Reactions struct {
	PerAction       PerBoard `json:"perAction"`
	UniquePerAction PerBoard `json:"uniquePerAction"`
}

type Stickers struct {
	PerCard PerBoard `json:"perCard"`
}

type Membership struct {
	ID          string `json:"id"`
	IDMember    string `json:"idMember"`
	MemberType  string `json:"memberType"`
	Unconfirmed bool   `json:"unconfirmed"`
	Deactivated bool   `json:"deactivated"`
}

type Prefs struct {
	PermissionLevel          string                  `json:"permissionLevel"`
	HideVotes                bool                    `json:"hideVotes"`
	Voting                   string                  `json:"voting"`
	Comments                 string                  `json:"comments"`
	Invitations              string                  `json:"invitations"`
	SelfJoin                 bool                    `json:"selfJoin"`
	CardCovers               bool                    `json:"cardCovers"`
	CardCounts               bool                    `json:"cardCounts"`
	IsTemplate               bool                    `json:"isTemplate"`
	CardAging                string                  `json:"cardAging"`
	CalendarFeedEnabled      bool                    `json:"calendarFeedEnabled"`
	HiddenPluginBoardButtons []interface{}           `json:"hiddenPluginBoardButtons"`
	SwitcherViews            []SwitcherView          `json:"switcherViews"`
	Background               string                  `json:"background"`
	BackgroundColor          interface{}             `json:"backgroundColor"`
	BackgroundImage          string                  `json:"backgroundImage"`
	BackgroundTile           bool                    `json:"backgroundTile"`
	BackgroundBrightness     string                  `json:"backgroundBrightness"`
	SharedSourceURL          string                  `json:"sharedSourceUrl"`
	BackgroundImageScaled    []BackgroundImageScaled `json:"backgroundImageScaled"`
	BackgroundBottomColor    string                  `json:"backgroundBottomColor"`
	BackgroundTopColor       string                  `json:"backgroundTopColor"`
	CanBePublic              bool                    `json:"canBePublic"`
	CanBeEnterprise          bool                    `json:"canBeEnterprise"`
	CanBeOrg                 bool                    `json:"canBeOrg"`
	CanBePrivate             bool                    `json:"canBePrivate"`
	CanInvite                bool                    `json:"canInvite"`
}

type BackgroundImageScaled struct {
	Width  int64  `json:"width"`
	Height int64  `json:"height"`
	URL    string `json:"url"`
}

type SwitcherView struct {
	ViewType string `json:"viewType"`
	Enabled  bool   `json:"enabled"`
}

// Cards

type Cards []Card

type Card struct {
	ID                    string        `json:"id"`
	Badges                Badges        `json:"badges"`
	CheckItemStates       []interface{} `json:"checkItemStates"`
	Closed                bool          `json:"closed"`
	DueComplete           bool          `json:"dueComplete"`
	DateLastActivity      string        `json:"dateLastActivity"`
	Desc                  string        `json:"desc"`
	DescData              DescData      `json:"descData"`
	Due                   *string       `json:"due"`
	DueReminder           *int64        `json:"dueReminder"`
	Email                 interface{}   `json:"email"`
	IDBoard               string        `json:"idBoard"`
	IDChecklists          []string      `json:"idChecklists"`
	IDList                string        `json:"idList"`
	IDMembers             []string      `json:"idMembers"`
	IDMembersVoted        []interface{} `json:"idMembersVoted"`
	IDShort               int64         `json:"idShort"`
	IDAttachmentCover     interface{}   `json:"idAttachmentCover"`
	Labels                []Label       `json:"labels"`
	IDLabels              []string      `json:"idLabels"`
	ManualCoverAttachment bool          `json:"manualCoverAttachment"`
	Name                  string        `json:"name"`
	Pos                   float64       `json:"pos"`
	ShortLink             string        `json:"shortLink"`
	ShortURL              string        `json:"shortUrl"`
	Start                 *string       `json:"start"`
	Subscribed            bool          `json:"subscribed"`
	URL                   string        `json:"url"`
	Cover                 Cover         `json:"cover"`
	IsTemplate            bool          `json:"isTemplate"`
	CardRole              interface{}   `json:"cardRole"`
}

type Badges struct {
	AttachmentsByType     AttachmentsByType `json:"attachmentsByType"`
	ExternalSource        interface{}       `json:"externalSource"`
	Location              bool              `json:"location"`
	Votes                 int64             `json:"votes"`
	ViewingMemberVoted    bool              `json:"viewingMemberVoted"`
	Subscribed            bool              `json:"subscribed"`
	Fogbugz               string            `json:"fogbugz"`
	CheckItems            int64             `json:"checkItems"`
	CheckItemsChecked     int64             `json:"checkItemsChecked"`
	CheckItemsEarliestDue interface{}       `json:"checkItemsEarliestDue"`
	Comments              int64             `json:"comments"`
	Attachments           int64             `json:"attachments"`
	Description           bool              `json:"description"`
	Due                   *string           `json:"due"`
	DueComplete           bool              `json:"dueComplete"`
	Start                 *string           `json:"start"`
	LastUpdatedByAI       bool              `json:"lastUpdatedByAi"`
}

type AttachmentsByType struct {
	Trello Trello `json:"trello"`
}

type Trello struct {
	Board int64 `json:"board"`
	Card  int64 `json:"card"`
}

type Cover struct {
	IDAttachment         interface{} `json:"idAttachment"`
	Color                interface{} `json:"color"`
	IDUploadedBackground interface{} `json:"idUploadedBackground"`
	Size                 string      `json:"size"`
	Brightness           string      `json:"brightness"`
	IDPlugin             interface{} `json:"idPlugin"`
}

type DescData struct {
	Emoji Emoji `json:"emoji"`
}

type Emoji struct {
}

type Label struct {
	ID      string `json:"id"`
	IDBoard string `json:"idBoard"`
	Name    string `json:"name"`
	Color   string `json:"color"`
	Uses    int64  `json:"uses"`
}

type Lists []List

type List struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Closed     bool        `json:"closed"`
	Color      interface{} `json:"color"`
	IDBoard    string      `json:"idBoard"`
	Pos        float64     `json:"pos"`
	Subscribed bool        `json:"subscribed"`
	SoftLimit  interface{} `json:"softLimit"`
	Type       interface{} `json:"type"`
	Datasource Datasource  `json:"datasource"`
}

type Datasource struct {
	Filter bool `json:"filter"`
}
