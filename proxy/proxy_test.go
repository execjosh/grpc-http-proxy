package proxy

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	_ "google.golang.org/grpc/test/grpc_testing"

	"github.com/mercari/grpc-http-proxy/metadata"
	"github.com/mercari/grpc-http-proxy/proxy/reflection"
	pstub "github.com/mercari/grpc-http-proxy/proxy/stub"
)

const (
	testService     = "grpc.testing.TestService"
	notFoundService = "not.found.NoService"
	emptyCall       = "EmptyCall"
	unaryCall       = "UnaryCall"
	file            = "grpc_testing/test.proto"
)

var testError = errors.Errorf("an error")

type fakeGrpcreflectClient struct {
	*desc.ServiceDescriptor
}

func (m *fakeGrpcreflectClient) ResolveService(serviceName string) (*desc.ServiceDescriptor, error) {
	if serviceName != testService {
		return nil, errors.Errorf("service not found")
	}
	return m.ServiceDescriptor, nil
}

type fakeGrpcdynamicStub struct {
	failMarshal bool
}

func (m *fakeGrpcdynamicStub) InvokeRpc(ctx context.Context, method *desc.MethodDescriptor, request proto.Message, opts ...grpc.CallOption) (proto.Message, error) {
	if method.GetName() == "UnaryCall" {
		return nil, status.Error(codes.Unimplemented, "unary unimplemented")
	}
	output := dynamic.NewMessage(method.GetOutputType())
	return output, nil
}

func TestNewProxy(t *testing.T) {
	p := NewProxy()
	if p == nil {
		t.Fatalf("proxy was nil")
	}
}

func TestProxy_Connect(t *testing.T) {
	p := NewProxy()
	p.Connect(context.Background(), parseURL(t, "localhost:5000"))
}

func TestProxy_Call(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		p := NewProxy()
		ctx := context.Background()
		md := make(metadata.Metadata)

		p.stub = pstub.NewStub(&fakeGrpcdynamicStub{})
		fd := newFileDescriptor(t, file)
		sd := reflection.ServiceDescriptorFromFileDescriptor(fd, testService)
		p.reflector = reflection.NewReflector(&fakeGrpcreflectClient{ServiceDescriptor: sd.ServiceDescriptor})

		_, err := p.Call(ctx, testService, emptyCall, []byte("{}"), &md)
		if err != nil {
			t.Fatalf("err should be nil, got %s", err.Error())
		}
	})

	t.Run("reflector fails", func(t *testing.T) {
		p := NewProxy()
		ctx := context.Background()
		md := make(metadata.Metadata)

		p.stub = pstub.NewStub(&fakeGrpcdynamicStub{})
		p.reflector = reflection.NewReflector(&fakeGrpcreflectClient{})

		_, err := p.Call(ctx, notFoundService, emptyCall, []byte("{}"), &md)
		if err == nil {
			t.Fatalf("err should be not nil")
		}
	})

	t.Run("invoking RPC returns error", func(t *testing.T) {
		p := NewProxy()
		ctx := context.Background()
		md := make(metadata.Metadata)

		p.stub = pstub.NewStub(&fakeGrpcdynamicStub{})
		fd := newFileDescriptor(t, file)
		sd := reflection.ServiceDescriptorFromFileDescriptor(fd, testService)
		p.reflector = reflection.NewReflector(&fakeGrpcreflectClient{ServiceDescriptor: sd.ServiceDescriptor})

		_, err := p.Call(ctx, testService, unaryCall, []byte("{}"), &md)
		if err == nil {
			t.Fatalf("err should be not nil")
		}
	})
}
