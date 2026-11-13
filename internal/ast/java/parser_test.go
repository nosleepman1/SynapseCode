package java

import (
	"context"
	"strings"
	"testing"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestJavaParser(t *testing.T) {
	code := `
package com.example.demo.controller;

import org.springframework.web.bind.annotation.*;
import org.springframework.beans.factory.annotation.Autowired;
import java.util.List;

public interface UserService {
    List<String> getAllUsers();
}

public enum UserRole {
    ADMIN, USER
}

public record UserDto(String id, String email) {}

@RestController
@RequestMapping("/api/users")
public class UserController extends BaseController implements ApiHandler {

    private final UserService userService;

    @Autowired
    public UserController(UserService userService) {
        this.userService = userService;
    }

    @GetMapping
    public List<String> listUsers() {
        return userService.getAllUsers();
    }
}
`

	parser := NewParser()
	parsed, err := parser.Parse(context.Background(), "UserController.java", "UserController.java", []byte(code))
	if err != nil {
		t.Fatalf("failed to parse Java code: %v", err)
	}

	if len(parsed.Imports) < 3 {
		t.Errorf("expected at least 3 imports, got %v", parsed.Imports)
	}

	hasInterface := false
	hasEnum := false
	hasRecord := false
	hasClass := false
	hasCtor := false
	hasMethod := false

	for _, s := range parsed.Symbols {
		if s.Name == "UserService" && s.Kind == model.KindInterface {
			hasInterface = true
		}
		if s.Name == "UserRole" && s.Kind == model.KindEnum {
			hasEnum = true
		}
		if s.Name == "UserDto" && s.Kind == model.KindRecord {
			hasRecord = true
		}
		if s.Name == "UserController" && s.Kind == model.KindClass {
			hasClass = true
			if !strings.Contains(s.Signature, "@RestController") {
				t.Errorf("expected @RestController in UserController signature, got: %s", s.Signature)
			}
			if len(s.Implements) < 2 {
				t.Errorf("expected extends BaseController & implements ApiHandler, got: %v", s.Implements)
			}
		}
		if s.Name == "UserController" && s.Kind == model.KindMethod && s.QualifiedName == "UserController.UserController" {
			hasCtor = true
			if !strings.Contains(s.Signature, "@Autowired") {
				t.Errorf("expected @Autowired in constructor signature, got: %s", s.Signature)
			}
		}
		if s.Name == "listUsers" && s.Kind == model.KindMethod {
			hasMethod = true
			if !strings.Contains(s.Signature, "@GetMapping") {
				t.Errorf("expected @GetMapping in listUsers signature, got: %s", s.Signature)
			}
		}
	}

	if !hasInterface {
		t.Errorf("expected UserService interface")
	}
	if !hasEnum {
		t.Errorf("expected UserRole enum")
	}
	if !hasRecord {
		t.Errorf("expected UserDto record")
	}
	if !hasClass {
		t.Errorf("expected UserController class")
	}
	if !hasCtor {
		t.Errorf("expected UserController constructor")
	}
	if !hasMethod {
		t.Errorf("expected listUsers method")
	}
}
