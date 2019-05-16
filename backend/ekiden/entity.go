package ekiden

type Request struct {
	Args   interface{} `cbor:"args" codec:"args"`
	Method string      `cbor:"method" codec:"method"`
}

type ArgTransaction struct {
	Transaction []byte
}

type Response struct {
	Error   string      `cbor:"Error"`
	Success interface{} `cbor:"Success"`
}
