package cmdb

import "time"

type ApplicationListQuery struct {
	Environment string
	Status      string
	Search      string
	Limit       int
	Offset      int
}

type InstanceListQuery struct {
	AppID       string
	Environment string
	Limit       int
	Offset      int
}

type ApplicationNaming struct {
	Organization     string `json:"organization"`
	BusinessDomain   string `json:"business_domain"`
	CapabilityDomain string `json:"capability_domain"`
	Application      string `json:"application"`
	Role             string `json:"role"`
}

type CreateApplicationRequest struct {
	AppID         string            `json:"app_id"`
	AppCode       string            `json:"app_code,omitempty"`
	AppName       string            `json:"app_name"`
	Environment   string            `json:"environment"`
	Status        string            `json:"status"`
	Lifecycle     string            `json:"lifecycle"`
	OwnerTeamCode string            `json:"owner_team_code,omitempty"`
	OwnerTeamName string            `json:"owner_team_name,omitempty"`
	Language      string            `json:"language,omitempty"`
	RepositoryURL string            `json:"repository_url,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

type UpdateApplicationRequest struct {
	AppID         string            `json:"app_id"`
	AppName       string            `json:"app_name"`
	Environment   string            `json:"environment"`
	Status        string            `json:"status"`
	Lifecycle     string            `json:"lifecycle"`
	OwnerTeamCode string            `json:"owner_team_code,omitempty"`
	OwnerTeamName string            `json:"owner_team_name,omitempty"`
	Language      string            `json:"language,omitempty"`
	RepositoryURL string            `json:"repository_url,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

type ApplicationSummary struct {
	AppID               string    `json:"app_id"`
	AppCode             string    `json:"app_code"`
	AppName             string    `json:"app_name"`
	Environment         string    `json:"environment"`
	Status              string    `json:"status"`
	Lifecycle           string    `json:"lifecycle"`
	OwnerTeamCode       string    `json:"owner_team_code,omitempty"`
	OwnerTeamName       string    `json:"owner_team_name,omitempty"`
	Language            string    `json:"language,omitempty"`
	RepositoryURL       string    `json:"repository_url,omitempty"`
	InstanceCount       int       `json:"instance_count"`
	ActiveInstanceCount int       `json:"active_instance_count"`
	UpdatedAt           time.Time `json:"updated_at"`
	CacheVersion        int64     `json:"cache_version"`
}

type ApplicationDetail struct {
	CIID                string            `json:"ci_id"`
	AppID               string            `json:"app_id"`
	AppCode             string            `json:"app_code"`
	AppName             string            `json:"app_name"`
	Environment         string            `json:"environment"`
	Status              string            `json:"status"`
	Lifecycle           string            `json:"lifecycle"`
	OwnerTeamCode       string            `json:"owner_team_code,omitempty"`
	OwnerTeamName       string            `json:"owner_team_name,omitempty"`
	Language            string            `json:"language,omitempty"`
	RepositoryURL       string            `json:"repository_url,omitempty"`
	InstanceCount       int               `json:"instance_count"`
	ActiveInstanceCount int               `json:"active_instance_count"`
	Labels              map[string]string `json:"labels,omitempty"`
	Naming              ApplicationNaming `json:"naming"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	CacheVersion        int64             `json:"cache_version"`
}

type PersonRelation struct {
	AppID          string     `json:"app_id"`
	PersonID       string     `json:"person_id"`
	PersonCode     string     `json:"person_code"`
	PersonName     string     `json:"person_name"`
	Email          string     `json:"email,omitempty"`
	Role           string     `json:"role"`
	RelationSource string     `json:"relation_source"`
	ValidFrom      time.Time  `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to,omitempty"`
	ObservedAt     time.Time  `json:"observed_at"`
}

type InstanceSummary struct {
	AppID              string    `json:"app_id"`
	InstanceCIID       string    `json:"instance_ci_id,omitempty"`
	InstanceExternalID string    `json:"instance_external_id"`
	Environment        string    `json:"environment"`
	Region             string    `json:"region,omitempty"`
	Zone               string    `json:"zone,omitempty"`
	PrivateIP          string    `json:"private_ip,omitempty"`
	PublicIP           string    `json:"public_ip,omitempty"`
	Port               int       `json:"port,omitempty"`
	Version            string    `json:"version,omitempty"`
	RuntimeStatus      string    `json:"runtime_status"`
	ResourceVersion    string    `json:"resource_version,omitempty"`
	ObservedAt         time.Time `json:"observed_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ServiceStatus struct {
	PostgreSQLConfigured bool     `json:"postgresql_configured"`
	RedisConfigured      bool     `json:"redis_configured"`
	HighFrequencyReads   []string `json:"high_frequency_reads"`
}
