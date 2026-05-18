package auth

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

const (
	DefaultAllowedEmailDomains = "wusd.org,it.wusd.org,staff.wusd.org"
	DefaultDeniedEmailDomains  = "stu.wusd.org"

	RoleITAdmin        = "it_admin"
	RoleHumanResources = "human_resources"
	RoleSiteAdmin      = "site_admin"
	RoleSiteSecretary  = "site_secretary"
	RoleDeviceWrangler = "device_wrangler"
	RoleFacultyStaff   = "faculty_staff"
)

type SAMLConfig struct {
	EntityID       string
	ACSURL         string
	IDPMetadataURL string
	IDPSSOURL      string
	IDPCertFile    string
}

type GroupRoleMapping struct {
	Group string   `json:"group"`
	Roles []string `json:"roles"`
}

type AttributeRoleMapping struct {
	Attribute string   `json:"attribute"`
	Values    []string `json:"values"`
	Roles     []string `json:"roles"`
}

type SiteScopeMapping struct {
	SourceType string   `json:"source_type"`
	Source     string   `json:"source"`
	Values     []string `json:"values,omitempty"`
	Sites      []string `json:"sites"`
}

type Policy struct {
	AllowedEmailDomains   []string
	DeniedEmailDomains    []string
	GroupRoleMappings     []GroupRoleMapping
	AttributeRoleMappings []AttributeRoleMapping
	SiteScopeMappings     []SiteScopeMapping
	SAML                  SAMLConfig
}

type GoogleIdentity struct {
	Email           string
	Groups          []string
	Attributes      map[string][]string
	BreakglassLocal bool
}

type Decision struct {
	Authorized bool
	Email      string
	Roles      []string
	SiteScopes []string
	Reason     string
}

// DefaultPolicy returns the production authorization boundary documented for
// Google SAML. Callers may layer group, attribute, and site mappings over these
// defaults, but the staff-domain allowlist and explicit student denial should
// stay active for normal SAML users.
func DefaultPolicy() Policy {
	return Policy{
		AllowedEmailDomains: parseDomainList(DefaultAllowedEmailDomains),
		DeniedEmailDomains:  parseDomainList(DefaultDeniedEmailDomains),
	}
}

// EvaluateGoogleIdentity converts verified Google SAML identity data into the
// application authorization decision. The domain gate runs before role mapping
// for normal SAML users so student or unknown domains cannot receive access via
// a broad group match. Local breakglass identities are evaluated by the future
// breakglass implementation and therefore bypass only the domain gate here.
func EvaluateGoogleIdentity(policy Policy, identity GoogleIdentity) Decision {
	email := canonicalEmail(identity.Email)
	if email == "" {
		return Decision{Email: email, Reason: "missing_email"}
	}

	if !identity.BreakglassLocal {
		if emailDomainDenied(email, policy.DeniedEmailDomains) {
			return Decision{Email: email, Reason: "denied_domain"}
		}
		if !emailDomainAllowed(email, policy.AllowedEmailDomains) {
			return Decision{Email: email, Reason: "domain_not_allowed"}
		}
	}

	roles := resolveRoles(policy, identity)
	if len(roles) == 0 {
		return Decision{Email: email, Reason: "no_role_mapping"}
	}

	return Decision{
		Authorized: true,
		Email:      email,
		Roles:      roles,
		SiteScopes: resolveSiteScopes(policy, identity),
	}
}

// ParseGroupRoleMappings reads the checked-in JSON contract used by
// GOOGLE_AUTH_GROUP_ROLE_MAPPINGS_JSON. It fails closed when a mapping has no
// group or no roles so production startup cannot silently accept an incomplete
// authorization rule.
func ParseGroupRoleMappings(raw string) ([]GroupRoleMapping, error) {
	var mappings []GroupRoleMapping
	if strings.TrimSpace(raw) == "" {
		return mappings, nil
	}
	if err := json.Unmarshal([]byte(raw), &mappings); err != nil {
		return nil, fmt.Errorf("parse group role mappings: %w", err)
	}
	for index, mapping := range mappings {
		mappings[index].Group = canonicalMappingValue(mapping.Group)
		mappings[index].Roles = canonicalList(mapping.Roles)
		if mappings[index].Group == "" || len(mappings[index].Roles) == 0 {
			return nil, fmt.Errorf("group role mapping %d must include group and roles", index)
		}
	}
	return mappings, nil
}

// ParseAttributeRoleMappings reads GOOGLE_AUTH_ATTRIBUTE_ROLE_MAPPINGS_JSON.
// Values are case-insensitive because Google SAML claim formatting can vary by
// admin-entered attribute data, while role ids remain stable application ids.
func ParseAttributeRoleMappings(raw string) ([]AttributeRoleMapping, error) {
	var mappings []AttributeRoleMapping
	if strings.TrimSpace(raw) == "" {
		return mappings, nil
	}
	if err := json.Unmarshal([]byte(raw), &mappings); err != nil {
		return nil, fmt.Errorf("parse attribute role mappings: %w", err)
	}
	for index, mapping := range mappings {
		mappings[index].Attribute = canonicalMappingValue(mapping.Attribute)
		mappings[index].Values = canonicalList(mapping.Values)
		mappings[index].Roles = canonicalList(mapping.Roles)
		if mappings[index].Attribute == "" || len(mappings[index].Values) == 0 || len(mappings[index].Roles) == 0 {
			return nil, fmt.Errorf("attribute role mapping %d must include attribute, values, and roles", index)
		}
	}
	return mappings, nil
}

// ParseSiteScopeMappings reads GOOGLE_AUTH_SITE_SCOPE_MAPPINGS_JSON. These
// mappings are the checked-in boundary for the manual site-scope bridge until
// Google group authorization fully represents every site assignment.
func ParseSiteScopeMappings(raw string) ([]SiteScopeMapping, error) {
	var mappings []SiteScopeMapping
	if strings.TrimSpace(raw) == "" {
		return mappings, nil
	}
	if err := json.Unmarshal([]byte(raw), &mappings); err != nil {
		return nil, fmt.Errorf("parse site scope mappings: %w", err)
	}
	for index, mapping := range mappings {
		mappings[index].SourceType = canonicalMappingValue(mapping.SourceType)
		mappings[index].Source = canonicalMappingValue(mapping.Source)
		mappings[index].Values = canonicalList(mapping.Values)
		mappings[index].Sites = canonicalList(mapping.Sites)
		if mappings[index].SourceType != "group" && mappings[index].SourceType != "attribute" {
			return nil, fmt.Errorf("site scope mapping %d must use source_type group or attribute", index)
		}
		if mappings[index].Source == "" || len(mappings[index].Sites) == 0 {
			return nil, fmt.Errorf("site scope mapping %d must include source and sites", index)
		}
		if mappings[index].SourceType == "attribute" && len(mappings[index].Values) == 0 {
			return nil, fmt.Errorf("site scope attribute mapping %d must include values", index)
		}
	}
	return mappings, nil
}

// ParseDomainList normalizes comma-separated domain config into lowercase
// domain names without leading at-signs. Empty config returns an empty list so
// callers can deliberately fail closed or supply repo defaults.
func ParseDomainList(raw string) []string {
	return parseDomainList(raw)
}

func resolveRoles(policy Policy, identity GoogleIdentity) []string {
	roles := map[string]struct{}{}
	groups := canonicalSet(identity.Groups)
	attributes := canonicalAttributeSet(identity.Attributes)

	for _, mapping := range policy.GroupRoleMappings {
		if _, ok := groups[canonicalMappingValue(mapping.Group)]; ok {
			addValues(roles, mapping.Roles)
		}
	}
	for _, mapping := range policy.AttributeRoleMappings {
		values := attributes[canonicalMappingValue(mapping.Attribute)]
		if intersects(values, canonicalSet(mapping.Values)) {
			addValues(roles, mapping.Roles)
		}
	}
	return sortedKeys(roles)
}

func resolveSiteScopes(policy Policy, identity GoogleIdentity) []string {
	scopes := map[string]struct{}{}
	groups := canonicalSet(identity.Groups)
	attributes := canonicalAttributeSet(identity.Attributes)

	for _, mapping := range policy.SiteScopeMappings {
		switch canonicalMappingValue(mapping.SourceType) {
		case "group":
			if _, ok := groups[canonicalMappingValue(mapping.Source)]; ok {
				addValues(scopes, mapping.Sites)
			}
		case "attribute":
			values := attributes[canonicalMappingValue(mapping.Source)]
			if intersects(values, canonicalSet(mapping.Values)) {
				addValues(scopes, mapping.Sites)
			}
		}
	}
	return sortedKeys(scopes)
}

func emailDomainAllowed(email string, domains []string) bool {
	domain := domainFromEmail(email)
	return domain != "" && slices.Contains(canonicalList(domains), domain)
}

func emailDomainDenied(email string, domains []string) bool {
	domain := domainFromEmail(email)
	return domain != "" && slices.Contains(canonicalList(domains), domain)
}

func domainFromEmail(email string) string {
	_, domain, ok := strings.Cut(canonicalEmail(email), "@")
	if !ok {
		return ""
	}
	return domain
}

func canonicalEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func canonicalMappingValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseDomainList(raw string) []string {
	parts := strings.Split(raw, ",")
	domains := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		domain := strings.TrimPrefix(canonicalMappingValue(part), "@")
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}
	slices.Sort(domains)
	return domains
}

func canonicalList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		normalized := canonicalMappingValue(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	slices.Sort(out)
	return out
}

func canonicalSet(values []string) map[string]struct{} {
	set := map[string]struct{}{}
	addValues(set, values)
	return set
}

func canonicalAttributeSet(values map[string][]string) map[string]map[string]struct{} {
	out := map[string]map[string]struct{}{}
	for key, attributeValues := range values {
		out[canonicalMappingValue(key)] = canonicalSet(attributeValues)
	}
	return out
}

func intersects(values map[string]struct{}, expected map[string]struct{}) bool {
	for value := range values {
		if _, ok := expected[value]; ok {
			return true
		}
	}
	return false
}

func addValues(set map[string]struct{}, values []string) {
	for _, value := range values {
		normalized := canonicalMappingValue(value)
		if normalized != "" {
			set[normalized] = struct{}{}
		}
	}
}

func sortedKeys(set map[string]struct{}) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	slices.Sort(values)
	return values
}
