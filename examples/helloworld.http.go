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

type GreeterHTTPConverter struct {
	srv GreeterServer
}

func NewGreeterHTTPConverter(srv GreeterServer) *GreeterHTTPConverter {
	return &GreeterHTTPConverter{
		srv: srv,
	}
}

func (h *GreeterHTTPConverter) SayHello(cb func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error)) http.HandlerFunc {
	if cb == nil {
		cb = func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error) {
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "%v: arg = %v: ret = %v", err, arg, ret)
			}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			cb(ctx, w, r, nil, nil, err)
			return
		}

		arg := &HelloRequest{}

		contentType := r.Header.Get("Content-Type")
		switch contentType {
		case "application/protobuf", "application/x-protobuf":
			if err := proto.Unmarshal(body, arg); err != nil {
				cb(ctx, w, r, nil, nil, err)
				return
			}
		case "application/json":
			if err := jsonpb.Unmarshal(bytes.NewBuffer(body), arg); err != nil {
				cb(ctx, w, r, nil, nil, err)
				return
			}
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			_, err := fmt.Fprintf(w, "Unsupported Content-Type: %s", contentType)
			cb(ctx, w, r, nil, nil, err)
			return
		}

		ret, err := h.srv.SayHello(ctx, arg)
		if err != nil {
			cb(ctx, w, r, arg, nil, err)
			return
		}

		switch contentType {
		case "application/protobuf", "application/x-protobuf":
			buf, err := proto.Marshal(ret)
			if err != nil {
				cb(ctx, w, r, arg, ret, err)
				return
			}
			if _, err := io.Copy(w, bytes.NewBuffer(buf)); err != nil {
				cb(ctx, w, r, arg, ret, err)
				return
			}
		case "application/json":
			if err := json.NewEncoder(w).Encode(ret); err != nil {
				cb(ctx, w, r, arg, ret, err)
				return
			}
		default:
			w.WriteHeader(http.StatusUnsupportedMediaType)
			_, err := fmt.Fprintf(w, "Unsupported Content-Type: %s", contentType)
			cb(ctx, w, r, arg, ret, err)
			return
		}
		cb(ctx, w, r, arg, ret, nil)
	})
}

func (h *GreeterHTTPConverter) SayHelloWithPath(cb func(ctx context.Context, w http.ResponseWriter, r *http.Request, arg, ret proto.Message, err error)) (string, http.HandlerFunc) {
	return "greeter/sayhello", h.SayHello(cb)
}
