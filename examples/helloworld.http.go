package example

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

type GreeterHandler struct {
	srv GreeterServer
}

func NewGreeterHandler(srv GreeterServer) *GreeterHandler {
	return &GreeterHandler{
		srv: srv,
	}
}

func (h *GreeterHandler) SayHello(cb func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error)) http.HandlerFunc {
	if cb == nil {
		cb = func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error) {
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "%v: arg = %v: ret = %v", err, arg, ret)
			}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx = r.Context()
			arg *HelloRequest
			ret *HelloReply
			err error
		)
		defer func() {
			cb(ctx, w, r, arg, ret, err)
		}()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return
		}

		tmp := &HelloRequest{}
		contentType := r.Header.Get("Content-Type")
		switch contentType {
		case "application/protobuf", "application/x-protobuf":
			if err = proto.Unmarshal(body, tmp); err != nil {
				return
			}
		case "application/json":
			if err = jsonpb.Unmarshal(bytes.NewBuffer(body), tmp); err != nil {
				return
			}
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			_, err = fmt.Fprintf(w, "Unsupported Content-Type: %s", contentType)
			return
		}
		arg = tmp

		ret, err = h.srv.SayHello(ctx, arg)
		if err != nil {
			return
		}

		switch contentType {
		case "application/protobuf", "application/x-protobuf":
			buf, err := proto.Marshal(ret)
			if err != nil {
				return
			}
			if _, err = io.Copy(w, bytes.NewBuffer(buf)); err != nil {
				return
			}
		case "application/json":
			if err = json.NewEncoder(w).Encode(ret); err != nil {
				return
			}
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			_, err = fmt.Fprintf(w, "Unsupported Content-Type: %s", contentType)
			return
		}
	})
}