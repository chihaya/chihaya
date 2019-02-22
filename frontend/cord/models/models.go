package models

const (
	ErrorInvalidJSONFormat         = 1
	ErrorReadDataBase              = 2
	ErrorUserAlreadyExists         = 3
	ErrorCreateUser                = 4
	ErrorDeleteUser                = 5
	ErrorInvalidUsernameOrPassword = 6
	ErrorGenToken                  = 7
	ErrorLogout                    = 8
	ErrorUnauthorized              = 9
	ErrorTokenExpired              = 10
	ErrorInvalidToken              = 11
	ErrorTorrentAlreadyExists      = 12
	ErrorAddTorrent                = 13
	ErrorDeleteTorrent             = 14
)

type AppKey struct {
	PrivateKeyPath string `json:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path"`
	JwtExpDelta    int    `json:"jwt_exp_delta"`
}

type DbAuth struct {
	Hosts    []string `json:"host"`
	Uname    string   `json:"username"`
	Pswd     string   `json:"password"`
	Database string   `json:"database"`
}

type Configuration struct {
	AppKeyCfg AppKey `json:"app_key"`
	DbAuthCfg DbAuth `json:"db_auth"`
}

type Authorization struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

type CreatedId struct {
	Title string `json:"title"`
	Id    string `json:"id"`
}

type AuthToken struct {
	ClientId     string `json:"client_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthRefresh struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Torrent struct {
	InfoHash string `bson:"info_hash" json:"info_hash"`
}

type User struct {
	Username string `bson:"username" json:"username"`
	Password string `bson:"password" json:"password"`
}
