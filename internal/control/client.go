package control

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"time"
)

func Call(socket, credentialPath string, q Request) (Response, error) {
	b, err := ReadCredential(credentialPath)
	if err != nil {
		return Response{}, err
	}
	q.Credential = base64.RawStdEncoding.EncodeToString(b)
	c, err := net.Dial("unix", socket)
	if err != nil {
		return Response{}, err
	}
	defer c.Close()
	if err := writeRequest(c, q); err != nil {
		return Response{}, err
	}
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		if err := cw.CloseWrite(); err != nil {
			return Response{}, err
		}
	}
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	rr, err := ReadResponse(c)
	return rr, err
}
func writeRequest(c net.Conn, q Request) error {
	b, err := jsonMarshal(q)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if len(b) > MaxFrameSize {
		return ErrFrameTooLarge
	}
	_, err = c.Write(b)
	return err
}
func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
func ReadResponse(r net.Conn) (Response, error) {
	var v Response
	b, err := readFrame(r)
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(b, &v)
	return v, err
}
