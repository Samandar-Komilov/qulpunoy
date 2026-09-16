package routers

type userRegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userRegisterResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	JoinedAt string `json:"joinedAt"`
}
