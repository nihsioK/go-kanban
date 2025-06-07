package user

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}
