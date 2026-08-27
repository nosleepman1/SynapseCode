package typescript

import (
	"context"
	"testing"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestTypeScriptParser(t *testing.T) {
	code := `
import { Request, Response } from 'express';

export interface UserDTO {
    id: string;
    email: string;
}

export class AuthService {
    async login(dto: UserDTO): Promise<string> {
        return "token";
    }
}

export function createServer(): void {
}
`

	parser := NewParser()
	parsed, err := parser.Parse(context.Background(), "auth.ts", "auth.ts", []byte(code))
	if err != nil {
		t.Fatalf("failed to parse TS code: %v", err)
	}

	if len(parsed.Imports) != 1 || parsed.Imports[0] != "express" {
		t.Errorf("expected 1 import from express, got %v", parsed.Imports)
	}

	hasInterface := false
	hasClass := false
	hasFunction := false

	for _, s := range parsed.Symbols {
		if s.Name == "UserDTO" && s.Kind == model.KindInterface {
			hasInterface = true
		}
		if s.Name == "AuthService" && s.Kind == model.KindClass {
			hasClass = true
		}
		if s.Name == "createServer" && s.Kind == model.KindFunction {
			hasFunction = true
		}
	}

	if !hasInterface {
		t.Errorf("expected UserDTO interface")
	}
	if !hasClass {
		t.Errorf("expected AuthService class")
	}
	if !hasFunction {
		t.Errorf("expected createServer function")
	}
}
