package models

const (
	// ErrorInvalidJSONFormat ...
	ErrorInvalidJSONFormat = 1
	// ErrorReadDataBase ...
	ErrorReadDataBase = 2
	// ErrorUserAlreadyExists ...
	ErrorUserAlreadyExists = 3
	// ErrorCreateUser ...
	ErrorCreateUser = 4
	// ErrorDeleteUser ...
	ErrorDeleteUser = 5
	// ErrorInvalidUsernameOrPassword ...
	ErrorInvalidUsernameOrPassword = 6
	// ErrorGenToken ...
	ErrorGenToken = 7
	// ErrorLogout ...
	ErrorLogout = 8
	// ErrorUnauthorized ...
	ErrorUnauthorized = 9
	// ErrorTokenExpired ...
	ErrorTokenExpired = 10
	// ErrorInvalidToken ...
	ErrorInvalidToken = 11
	// ErrorTorrentAlreadyExists ...
	ErrorTorrentAlreadyExists = 12
	// ErrorAddTorrent ...
	ErrorAddTorrent = 13
	// ErrorDeleteTorrent ...
	ErrorDeleteTorrent = 14
)

// AppKey ...
type AppKey struct {
	PrivateKeyPath string `json:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path"`
	JwtExpDelta    int    `json:"jwt_exp_delta"`
}

// DbAuth ...
type DbAuth struct {
	Hosts    []string `json:"host"`
	Uname    string   `json:"username"`
	Pswd     string   `json:"password"`
	Database string   `json:"database"`
}

// Configuration ...
type Configuration struct {
	AppKeyCfg AppKey `json:"app_key"`
	DbAuthCfg DbAuth `json:"db_auth"`
}

// Authorization ...
type Authorization struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

// CreatedID ...
type CreatedID struct {
	Title string `json:"title"`
	ID    string `json:"id"`
}

// AuthToken ...
type AuthToken struct {
	ClientID     string `json:"client_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthRefresh ...
type AuthRefresh struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// Error ...
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Torrent ...
type Torrent struct {
	InfoHash string `bson:"info_hash" json:"info_hash"`
}

// User ...
type User struct {
	Username string `bson:"username" json:"username"`
	Password string `bson:"password" json:"password"`
}
