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

export enum Role {
    Admin = "ADMIN",
    User = "USER"
}

export class AuthService implements IAuth {
    async login(dto: UserDTO): Promise<string> {
        return generateToken(dto.id);
    }
}

export const handleRequest = async (req: Request): Promise<void> => {
    AuthService.login(req.body);
};

export function createServer(): void {
    initApp();
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
	hasArrowFunc := false
	hasMethod := false
	hasEnum := false

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
		if s.Name == "handleRequest" && s.Kind == model.KindFunction {
			hasArrowFunc = true
		}
		if s.Name == "login" && s.Kind == model.KindMethod && s.QualifiedName == "AuthService.login" {
			hasMethod = true
		}
		if s.Name == "Role" && s.Kind == model.KindType {
			hasEnum = true
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
	if !hasArrowFunc {
		t.Errorf("expected handleRequest arrow function")
	}
	if !hasMethod {
		t.Errorf("expected AuthService.login method")
	}
	if !hasEnum {
		t.Errorf("expected Role enum")
	}
}
