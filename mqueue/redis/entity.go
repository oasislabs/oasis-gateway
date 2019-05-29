package redis

type redisElement struct {
	Set    bool   `json:"set"`
	Offset string `json:"offset"`
	Type   string `json:"value_type"`
	Value  string `json:"value"`
}
