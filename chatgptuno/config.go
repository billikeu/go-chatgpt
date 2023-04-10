package chatgptuno

type ChatGPTUnoConfig struct {
	EmailAddr    string
	Passwd       string
	AccessToken  string // https://chat.openai.com/api/auth/session
	SessionToken string
	Proxy        string
	Model        string // model: text-davinci-002-render-paid text-davinci-002-render-sha
	BaseUrl      string
}
