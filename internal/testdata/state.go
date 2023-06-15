package testdata

type Record struct {
	Id   int
	Name string
}

type Authentication struct {
	UserID int
}
type State struct {

	/*
		SELECT * FROM MY_TABLE WHERE USER_ID = $Jwt.UserID
	*/
	Records []*Record `xdatly:"kind:data_view"`

	Auth *Authentication `xdatly:"kind:header,name=Authorization,codec=JwtClaim,statusCode:401"`
}
