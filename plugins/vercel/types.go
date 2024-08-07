package main

type Projects struct {
	Projects   []Project  `json:"projects"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Count int64       `json:"count"`
	Next  interface{} `json:"next"`
	Prev  int64       `json:"prev"`
}

type Project struct {
	AccountID                        string             `json:"accountId"`
	SpeedInsights                    *Analytics         `json:"speedInsights,omitempty"`
	AutoExposeSystemEnvs             *bool              `json:"autoExposeSystemEnvs,omitempty"`
	AutoAssignCustomDomains          *bool              `json:"autoAssignCustomDomains,omitempty"`
	AutoAssignCustomDomainsUpdatedBy *string            `json:"autoAssignCustomDomainsUpdatedBy,omitempty"`
	BuildCommand                     *string            `json:"buildCommand"`
	CommandForIgnoringBuildStep      interface{}        `json:"commandForIgnoringBuildStep"`
	CreatedAt                        int64              `json:"createdAt"`
	Crons                            *Crons             `json:"crons,omitempty"`
	DevCommand                       interface{}        `json:"devCommand"`
	DirectoryListing                 bool               `json:"directoryListing"`
	GitForkProtection                *bool              `json:"gitForkProtection,omitempty"`
	GitLFS                           *bool              `json:"gitLFS,omitempty"`
	ID                               string             `json:"id"`
	InstallCommand                   interface{}        `json:"installCommand"`
	LastRollbackTarget               interface{}        `json:"lastRollbackTarget"`
	LastAliasRequest                 interface{}        `json:"lastAliasRequest"`
	Name                             string             `json:"name"`
	NodeVersion                      string             `json:"nodeVersion"`
	OutputDirectory                  *string            `json:"outputDirectory"`
	PublicSource                     *bool              `json:"publicSource"`
	RootDirectory                    *string            `json:"rootDirectory"`
	ServerlessFunctionRegion         *string            `json:"serverlessFunctionRegion"`
	SourceFilesOutsideRootDirectory  *bool              `json:"sourceFilesOutsideRootDirectory,omitempty"`
	SsoProtection                    *SsoProtection     `json:"ssoProtection,omitempty"`
	UpdatedAt                        int64              `json:"updatedAt"`
	GitComments                      *GitComments       `json:"gitComments,omitempty"`
	WebAnalytics                     *WebAnalytics      `json:"webAnalytics,omitempty"`
	LatestDeployments                []LatestDeployment `json:"latestDeployments"`
	Targets                          Targets            `json:"targets"`
	Framework                        *string            `json:"framework"`
	Link                             *Link              `json:"link,omitempty"`
	Analytics                        *Analytics         `json:"analytics,omitempty"`
	PasswordProtection               interface{}        `json:"passwordProtection"`
	TransferStartedAt                *int64             `json:"transferStartedAt,omitempty"`
	TransferCompletedAt              *int64             `json:"transferCompletedAt,omitempty"`
	TransferredFromAccountID         *string            `json:"transferredFromAccountId,omitempty"`
	Env                              []Env              `json:"env,omitempty"`
	Live                             *bool              `json:"live,omitempty"`
}

type Analytics struct {
	ID         string `json:"id"`
	EnabledAt  *int64 `json:"enabledAt,omitempty"`
	DisabledAt *int64 `json:"disabledAt,omitempty"`
	CanceledAt *int64 `json:"canceledAt"`
	HasData    *bool  `json:"hasData,omitempty"`
}

type Crons struct {
	EnabledAt    int64         `json:"enabledAt"`
	DisabledAt   interface{}   `json:"disabledAt"`
	UpdatedAt    int64         `json:"updatedAt"`
	DeploymentID *string       `json:"deploymentId"`
	Definitions  []interface{} `json:"definitions"`
}

type Env struct {
	Target          []string    `json:"target"`
	ConfigurationID interface{} `json:"configurationId"`
	ID              string      `json:"id"`
	Key             string      `json:"key"`
	CreatedAt       int64       `json:"createdAt"`
	UpdatedAt       int64       `json:"updatedAt"`
	CreatedBy       string      `json:"createdBy"`
	UpdatedBy       *string     `json:"updatedBy"`
	Type            string      `json:"type"`
	Value           string      `json:"value"`
	Comment         *string     `json:"comment,omitempty"`
}

type GitComments struct {
	OnCommit      bool `json:"onCommit"`
	OnPullRequest bool `json:"onPullRequest"`
}

type LatestDeployment struct {
	Alias                  []string    `json:"alias"`
	AliasAssigned          *int64      `json:"aliasAssigned"`
	AliasError             interface{} `json:"aliasError"`
	AutomaticAliases       []string    `json:"automaticAliases,omitempty"`
	Builds                 []Build     `json:"builds"`
	CreatedAt              int64       `json:"createdAt"`
	CreatedIn              string      `json:"createdIn"`
	Creator                Creator     `json:"creator"`
	DeploymentHostname     string      `json:"deploymentHostname"`
	Forced                 bool        `json:"forced"`
	ID                     string      `json:"id"`
	Meta                   Meta        `json:"meta"`
	Name                   string      `json:"name"`
	Plan                   string      `json:"plan"`
	Private                bool        `json:"private"`
	ReadyState             string      `json:"readyState"`
	ReadySubstate          *string     `json:"readySubstate,omitempty"`
	Target                 *string     `json:"target"`
	TeamID                 string      `json:"teamId"`
	Type                   string      `json:"type"`
	URL                    string      `json:"url"`
	UserID                 string      `json:"userId"`
	WithCache              *bool       `json:"withCache,omitempty"`
	BuildingAt             *int64      `json:"buildingAt,omitempty"`
	ReadyAt                int64       `json:"readyAt"`
	PreviewCommentsEnabled *bool       `json:"previewCommentsEnabled,omitempty"`
}

type Build struct {
	Src    string  `json:"src"`
	Use    string  `json:"use"`
	Config *Config `json:"config,omitempty"`
}

type Config struct {
	ZeroConfig      bool    `json:"zeroConfig"`
	Framework       *string `json:"framework,omitempty"`
	BuildCommand    *string `json:"buildCommand,omitempty"`
	OutputDirectory *string `json:"outputDirectory,omitempty"`
}

type Creator struct {
	Uid         string  `json:"uid"`
	Email       string  `json:"email"`
	Username    string  `json:"username"`
	GithubLogin *string `json:"githubLogin,omitempty"`
}

type Meta struct {
	GithubCommitAuthorName  *string `json:"githubCommitAuthorName,omitempty"`
	GithubCommitMessage     *string `json:"githubCommitMessage,omitempty"`
	GithubCommitOrg         *string `json:"githubCommitOrg,omitempty"`
	GithubCommitRef         *string `json:"githubCommitRef,omitempty"`
	GithubCommitRepo        *string `json:"githubCommitRepo,omitempty"`
	GithubCommitSHA         *string `json:"githubCommitSha,omitempty"`
	GithubDeployment        *string `json:"githubDeployment,omitempty"`
	GithubOrg               *string `json:"githubOrg,omitempty"`
	GithubRepo              *string `json:"githubRepo,omitempty"`
	GithubRepoOwnerType     *string `json:"githubRepoOwnerType,omitempty"`
	GithubCommitRepoID      *string `json:"githubCommitRepoId,omitempty"`
	GithubRepoID            *string `json:"githubRepoId,omitempty"`
	GithubRepoVisibility    *string `json:"githubRepoVisibility,omitempty"`
	GithubCommitAuthorLogin *string `json:"githubCommitAuthorLogin,omitempty"`
	BranchAlias             *string `json:"branchAlias,omitempty"`
	GitCommitAuthorName     *string `json:"gitCommitAuthorName,omitempty"`
	GitCommitMessage        *string `json:"gitCommitMessage,omitempty"`
	GitCommitRef            *string `json:"gitCommitRef,omitempty"`
	GitCommitSHA            *string `json:"gitCommitSha,omitempty"`
	GitRemoteURL            *string `json:"gitRemoteUrl,omitempty"`
	Action                  *string `json:"action,omitempty"`
	OriginalDeploymentID    *string `json:"originalDeploymentId,omitempty"`
	GitDirty                *string `json:"gitDirty,omitempty"`
}

type Link struct {
	Type             string        `json:"type"`
	Repo             string        `json:"repo"`
	RepoID           int64         `json:"repoId"`
	Org              string        `json:"org"`
	GitCredentialID  string        `json:"gitCredentialId"`
	ProductionBranch string        `json:"productionBranch"`
	CreatedAt        int64         `json:"createdAt"`
	UpdatedAt        int64         `json:"updatedAt"`
	DeployHooks      []interface{} `json:"deployHooks"`
	Sourceless       *bool         `json:"sourceless,omitempty"`
}

type SsoProtection struct {
	DeploymentType string `json:"deploymentType"`
}

type Targets struct {
	Production LatestDeployment `json:"production"`
}

type WebAnalytics struct {
	ID string `json:"id"`
}

type Deployments struct {
	Deployments []Deployment `json:"deployments"`
	Pagination  Pagination   `json:"pagination"`
}

type Deployment struct {
	Uid                 string          `json:"uid"`
	Name                string          `json:"name"`
	URL                 string          `json:"url"`
	Created             int64           `json:"created"`
	Source              *string         `json:"source,omitempty"`
	State               string          `json:"state"`
	ReadyState          string          `json:"readyState"`
	ReadySubstate       *string         `json:"readySubstate,omitempty"`
	Type                string          `json:"type"`
	Creator             Creator         `json:"creator"`
	InspectorURL        string          `json:"inspectorUrl"`
	Meta                MetaD           `json:"meta"`
	Target              *string         `json:"target"`
	AliasError          interface{}     `json:"aliasError"`
	AliasAssigned       *int64          `json:"aliasAssigned"`
	IsRollbackCandidate bool            `json:"isRollbackCandidate"`
	CreatedAt           int64           `json:"createdAt"`
	BuildingAt          int64           `json:"buildingAt"`
	Ready               int64           `json:"ready"`
	ProjectSettings     ProjectSettings `json:"projectSettings"`
}

type MetaD struct {
	GithubCommitAuthorName  *string `json:"githubCommitAuthorName,omitempty"`
	GithubCommitMessage     *string `json:"githubCommitMessage,omitempty"`
	GithubCommitOrg         *string `json:"githubCommitOrg,omitempty"`
	GithubCommitRef         *string `json:"githubCommitRef,omitempty"`
	GithubCommitRepo        *string `json:"githubCommitRepo,omitempty"`
	GithubCommitSHA         *string `json:"githubCommitSha,omitempty"`
	GithubDeployment        *string `json:"githubDeployment,omitempty"`
	GithubOrg               *string `json:"githubOrg,omitempty"`
	GithubRepo              *string `json:"githubRepo,omitempty"`
	GithubRepoOwnerType     *string `json:"githubRepoOwnerType,omitempty"`
	GithubCommitRepoID      *string `json:"githubCommitRepoId,omitempty"`
	GithubRepoID            *string `json:"githubRepoId,omitempty"`
	GithubRepoVisibility    *string `json:"githubRepoVisibility,omitempty"`
	GithubCommitAuthorLogin *string `json:"githubCommitAuthorLogin,omitempty"`
	BranchAlias             *string `json:"branchAlias,omitempty"`
	GitDirty                *string `json:"gitDirty,omitempty"`
	Action                  *string `json:"action,omitempty"`
	OriginalDeploymentID    *string `json:"originalDeploymentId,omitempty"`
	GitCommitAuthorName     *string `json:"gitCommitAuthorName,omitempty"`
	GitCommitMessage        *string `json:"gitCommitMessage,omitempty"`
	GitCommitRef            *string `json:"gitCommitRef,omitempty"`
	GitCommitSHA            *string `json:"gitCommitSha,omitempty"`
	GitRemoteURL            *string `json:"gitRemoteUrl,omitempty"`
}

type ProjectSettings struct {
	CommandForIgnoringBuildStep interface{} `json:"commandForIgnoringBuildStep"`
}
