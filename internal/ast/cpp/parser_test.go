package cpp

import (
	"context"
	"strings"
	"testing"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestCppParser(t *testing.T) {
	code := `
#include <iostream>
#include <memory>
#include "connection.h"

namespace net::core {

enum class ConnectionState {
    Disconnected,
    Connecting,
    Connected
};

struct PacketHeader {
    uint32_t magic;
    uint32_t length;
};

class SocketConnection : public IConnection {
public:
    SocketConnection(int socketFd);
    virtual ~SocketConnection();

    virtual bool sendPacket(const PacketHeader& header) override {
        return writeRaw(header.magic);
    }
};

}
`

	parser := NewParser()
	parsed, err := parser.Parse(context.Background(), "socket.cpp", "socket.cpp", []byte(code))
	if err != nil {
		t.Fatalf("failed to parse C++ code: %v", err)
	}

	if len(parsed.Imports) < 3 {
		t.Errorf("expected at least 3 includes, got %v", parsed.Imports)
	}

	hasEnum := false
	hasStruct := false
	hasClass := false
	hasMethod := false

	for _, s := range parsed.Symbols {
		if s.Name == "ConnectionState" && s.Kind == model.KindEnum {
			hasEnum = true
		}
		if s.Name == "PacketHeader" && s.Kind == model.KindStruct {
			hasStruct = true
		}
		if s.Name == "SocketConnection" && s.Kind == model.KindClass {
			hasClass = true
			if len(s.Implements) == 0 || s.Implements[0] != "IConnection" {
				t.Errorf("expected IConnection base class, got: %v", s.Implements)
			}
		}
		if s.Name == "sendPacket" && s.Kind == model.KindMethod {
			hasMethod = true
			if !strings.Contains(s.Signature, "bool sendPacket") {
				t.Errorf("expected bool sendPacket in signature, got: %s", s.Signature)
			}
		}
	}

	if !hasEnum {
		t.Errorf("expected ConnectionState enum")
	}
	if !hasStruct {
		t.Errorf("expected PacketHeader struct")
	}
	if !hasClass {
		t.Errorf("expected SocketConnection class")
	}
	if !hasMethod {
		t.Errorf("expected sendPacket method")
	}
}
