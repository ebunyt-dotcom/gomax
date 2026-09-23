package types

// StartAuthResponse describes an SMS authentication challenge.
type StartAuthResponse struct {
	Token              string `json:"token" msgpack:"token"`
	CodeLength         int    `json:"codeLength" msgpack:"codeLength"`
	RequestMaxDuration int64  `json:"requestMaxDuration" msgpack:"requestMaxDuration"`
	RequestCountLeft   int    `json:"requestCountLeft" msgpack:"requestCountLeft"`
	AltActionDuration  int64  `json:"altActionDuration" msgpack:"altActionDuration"`
}

type AuthToken struct {
	Token string `json:"token" msgpack:"token"`
}

type TokenAttrs struct {
	Login    *AuthToken `json:"LOGIN,omitempty" msgpack:"LOGIN,omitempty"`
	Register *AuthToken `json:"REGISTER,omitempty" msgpack:"REGISTER,omitempty"`
}

type PasswordChallenge struct {
	TrackID string `json:"trackId" msgpack:"trackId"`
	Hint    string `json:"hint,omitempty" msgpack:"hint,omitempty"`
}

type CheckCodeResponse struct {
	TokenAttrs        TokenAttrs         `json:"tokenAttrs" msgpack:"tokenAttrs"`
	PasswordChallenge *PasswordChallenge `json:"passwordChallenge,omitempty" msgpack:"passwordChallenge,omitempty"`
}

func (response CheckCodeResponse) LoginToken() string {
	if response.TokenAttrs.Login != nil {
		return response.TokenAttrs.Login.Token
	}
	return ""
}

func (response CheckCodeResponse) RegisterToken() string {
	if response.TokenAttrs.Register != nil {
		return response.TokenAttrs.Register.Token
	}
	return ""
}

type CheckPasswordResponse struct {
	TokenAttrs TokenAttrs `json:"tokenAttrs" msgpack:"tokenAttrs"`
	Error      string     `json:"error,omitempty" msgpack:"error,omitempty"`
}

func (response CheckPasswordResponse) LoginToken() string {
	if response.TokenAttrs.Login != nil {
		return response.TokenAttrs.Login.Token
	}
	return ""
}

type RequestQRResponse struct {
	ExpiresAt       int64  `json:"expiresAt" msgpack:"expiresAt"`
	PollingInterval int64  `json:"pollingInterval" msgpack:"pollingInterval"`
	QRLink          string `json:"qrLink" msgpack:"qrLink"`
	TrackID         string `json:"trackId" msgpack:"trackId"`
	TTL             int64  `json:"ttl" msgpack:"ttl"`
}

type QRStatus struct {
	ExpiresAt      int64 `json:"expiresAt" msgpack:"expiresAt"`
	LoginAvailable bool  `json:"loginAvailable,omitempty" msgpack:"loginAvailable,omitempty"`
}

type CheckQRResponse struct {
	Status QRStatus `json:"status" msgpack:"status"`
}

type Profile struct {
	Contact        User  `json:"contact" msgpack:"contact"`
	ProfileOptions []int `json:"profileOptions,omitempty" msgpack:"profileOptions,omitempty"`
}

type ConfirmRegistrationResponse struct {
	UserToken int64   `json:"userToken" msgpack:"userToken"`
	Profile   Profile `json:"profile" msgpack:"profile"`
	TokenType string  `json:"tokenType" msgpack:"tokenType"`
	Token     string  `json:"token" msgpack:"token"`
}

type LoginConfig struct {
	Hash string `json:"hash,omitempty" msgpack:"hash,omitempty"`
}

type Login2Flags struct {
	ConfigEnabled  bool `json:"configEnabled" msgpack:"configEnabled"`
	ContactEnabled bool `json:"contactEnabled" msgpack:"contactEnabled"`
	ProfileEnabled bool `json:"profileEnabled" msgpack:"profileEnabled"`
}

func (flags Login2Flags) Enabled() bool {
	return flags.ConfigEnabled || flags.ContactEnabled || flags.ProfileEnabled
}

type LoginResponse struct {
	Chats       []Chat              `json:"chats" msgpack:"chats"`
	Profile     *Profile            `json:"profile,omitempty" msgpack:"profile,omitempty"`
	Messages    map[int64][]Message `json:"messages" msgpack:"messages"`
	Contacts    []User              `json:"contacts" msgpack:"contacts"`
	Token       string              `json:"token,omitempty" msgpack:"token,omitempty"`
	Time        int64               `json:"time,omitempty" msgpack:"time,omitempty"`
	Config      *LoginConfig        `json:"config,omitempty" msgpack:"config,omitempty"`
	Updates     int64               `json:"updates,omitempty" msgpack:"updates,omitempty"`
	Login2Flags *Login2Flags        `json:"login2Flags,omitempty" msgpack:"login2Flags,omitempty"`
}

type Login2Response struct {
	Profile  *Profile     `json:"profile,omitempty" msgpack:"profile,omitempty"`
	Contacts []User       `json:"contactInfos" msgpack:"contactInfos"`
	Config   *LoginConfig `json:"config,omitempty" msgpack:"config,omitempty"`
}

func ParseStartAuthResponse(src map[string]any) StartAuthResponse {
	result := StartAuthResponse{Token: StringValue(src["token"])}
	if value, ok := Int64Value(src["codeLength"]); ok {
		result.CodeLength = int(value)
	}
	result.RequestMaxDuration, _ = Int64Value(src["requestMaxDuration"])
	if value, ok := Int64Value(src["requestCountLeft"]); ok {
		result.RequestCountLeft = int(value)
	}
	result.AltActionDuration, _ = Int64Value(src["altActionDuration"])
	return result
}

func ParseCheckCodeResponse(src map[string]any) CheckCodeResponse {
	result := CheckCodeResponse{TokenAttrs: parseTokenAttrs(src)}
	if raw, ok := src["passwordChallenge"].(map[string]any); ok {
		result.PasswordChallenge = &PasswordChallenge{TrackID: StringValue(raw["trackId"]), Hint: StringValue(raw["hint"])}
	}
	return result
}

func ParseCheckPasswordResponse(src map[string]any) CheckPasswordResponse {
	return CheckPasswordResponse{TokenAttrs: parseTokenAttrs(src), Error: StringValue(src["error"])}
}

func parseTokenAttrs(src map[string]any) TokenAttrs {
	raw, _ := src["tokenAttrs"].(map[string]any)
	if raw == nil {
		raw, _ = src["token_attrs"].(map[string]any)
	}
	result := TokenAttrs{}
	if value := tokenValue(raw["LOGIN"]); value != "" {
		result.Login = &AuthToken{Token: value}
	}
	if value := tokenValue(raw["REGISTER"]); value != "" {
		result.Register = &AuthToken{Token: value}
	}
	return result
}

func tokenValue(value any) string {
	if raw, ok := value.(map[string]any); ok {
		return StringValue(raw["token"])
	}
	return StringValue(value)
}

func ParseRequestQRResponse(src map[string]any) RequestQRResponse {
	result := RequestQRResponse{QRLink: StringValue(src["qrLink"]), TrackID: StringValue(src["trackId"])}
	result.ExpiresAt, _ = Int64Value(src["expiresAt"])
	result.PollingInterval, _ = Int64Value(src["pollingInterval"])
	result.TTL, _ = Int64Value(src["ttl"])
	return result
}

func ParseCheckQRResponse(src map[string]any) CheckQRResponse {
	result := CheckQRResponse{}
	if raw, ok := src["status"].(map[string]any); ok {
		result.Status.ExpiresAt, _ = Int64Value(raw["expiresAt"])
		result.Status.LoginAvailable, _ = raw["loginAvailable"].(bool)
	}
	return result
}

func ParseProfilePayload(src map[string]any) Profile {
	result := Profile{}
	if contact, ok := src["contact"].(map[string]any); ok {
		result.Contact = ParseUserPayload(contact)
	}
	if options, ok := src["profileOptions"].([]any); ok {
		for _, raw := range options {
			if value, ok := Int64Value(raw); ok {
				result.ProfileOptions = append(result.ProfileOptions, int(value))
			}
		}
	}
	return result
}

func ParseConfirmRegistrationResponse(src map[string]any) ConfirmRegistrationResponse {
	result := ConfirmRegistrationResponse{Token: StringValue(src["token"]), TokenType: StringValue(src["tokenType"])}
	result.UserToken, _ = Int64Value(src["userToken"])
	if profile, ok := src["profile"].(map[string]any); ok {
		result.Profile = ParseProfilePayload(profile)
	}
	return result
}

func ParseLoginResponse(src map[string]any) LoginResponse {
	result := LoginResponse{Token: StringValue(src["token"]), Messages: make(map[int64][]Message)}
	result.Time, _ = Int64Value(src["time"])
	result.Updates, _ = Int64Value(src["updates"])
	if config, ok := src["config"].(map[string]any); ok {
		result.Config = &LoginConfig{Hash: StringValue(config["hash"])}
	}
	if flags, ok := src["login2Flags"].(map[string]any); ok {
		result.Login2Flags = &Login2Flags{}
		result.Login2Flags.ConfigEnabled, _ = flags["configEnabled"].(bool)
		result.Login2Flags.ContactEnabled, _ = flags["contactEnabled"].(bool)
		result.Login2Flags.ProfileEnabled, _ = flags["profileEnabled"].(bool)
	}
	if profile, ok := src["profile"].(map[string]any); ok {
		value := ParseProfilePayload(profile)
		result.Profile = &value
	}
	if chats, ok := src["chats"].([]any); ok {
		for _, raw := range chats {
			if value, ok := raw.(map[string]any); ok {
				result.Chats = append(result.Chats, ParseChatPayload(value))
			}
		}
	}
	for _, key := range []string{"contacts", "contactInfos"} {
		if contacts, ok := src[key].([]any); ok {
			for _, raw := range contacts {
				if value, ok := raw.(map[string]any); ok {
					result.Contacts = append(result.Contacts, ParseUserPayload(value))
				}
			}
			break
		}
	}
	if messages, ok := src["messages"].(map[string]any); ok {
		for key, rawList := range messages {
			chatID, _ := Int64Value(key)
			if list, ok := rawList.([]any); ok {
				for _, raw := range list {
					if value, ok := raw.(map[string]any); ok {
						message := ParseMessagePayload(value)
						if message.ChatID == 0 {
							message.ChatID = chatID
						}
						result.Messages[chatID] = append(result.Messages[chatID], *message)
					}
				}
			}
		}
	}
	return result
}

func ParseLogin2Response(src map[string]any) Login2Response {
	login := ParseLoginResponse(src)
	return Login2Response{Profile: login.Profile, Contacts: login.Contacts, Config: login.Config}
}
