package csharp

import (
	"context"
	"strings"
	"testing"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestCSharpParser(t *testing.T) {
	code := `
namespace MyApp.Controllers;

using Microsoft.AspNetCore.Mvc;
using System.Threading.Tasks;
using MyApp.Services;

public interface IUserService
{
    Task<UserDto> GetUserAsync(int id);
}

public enum UserStatus
{
    Active,
    Suspended
}

public record UserDto(int Id, string Name, string Email);

[ApiController]
[Route("api/[controller]")]
public class UsersController : ControllerBase, IDisposable
{
    private readonly IUserService _userService;

    public UsersController(IUserService userService)
    {
        _userService = userService;
    }

    [HttpGet("{id}")]
    public async Task<IActionResult> GetUser(int id)
    {
        var user = await _userService.GetUserAsync(id);
        return Ok(user);
    }
}
`

	parser := NewParser()
	parsed, err := parser.Parse(context.Background(), "UsersController.cs", "UsersController.cs", []byte(code))
	if err != nil {
		t.Fatalf("failed to parse C# code: %v", err)
	}

	if len(parsed.Imports) < 3 {
		t.Errorf("expected at least 3 usings, got %v", parsed.Imports)
	}

	hasInterface := false
	hasEnum := false
	hasRecord := false
	hasClass := false
	hasCtor := false
	hasMethod := false

	for _, s := range parsed.Symbols {
		if s.Name == "IUserService" && s.Kind == model.KindInterface {
			hasInterface = true
		}
		if s.Name == "UserStatus" && s.Kind == model.KindEnum {
			hasEnum = true
		}
		if s.Name == "UserDto" && s.Kind == model.KindRecord {
			hasRecord = true
		}
		if s.Name == "UsersController" && s.Kind == model.KindClass {
			hasClass = true
			if !strings.Contains(s.Signature, "[ApiController]") || !strings.Contains(s.Signature, "[Route(") {
				t.Errorf("expected [ApiController] & [Route] in UsersController signature, got: %s", s.Signature)
			}
			if len(s.Implements) < 2 {
				t.Errorf("expected ControllerBase & IDisposable in implements, got: %v", s.Implements)
			}
		}
		if s.Name == "UsersController" && s.Kind == model.KindMethod && s.QualifiedName == "UsersController.UsersController" {
			hasCtor = true
		}
		if s.Name == "GetUser" && s.Kind == model.KindMethod {
			hasMethod = true
			if !strings.Contains(s.Signature, "[HttpGet(") {
				t.Errorf("expected [HttpGet] attribute in GetUser signature, got: %s", s.Signature)
			}
		}
	}

	if !hasInterface {
		t.Errorf("expected IUserService interface")
	}
	if !hasEnum {
		t.Errorf("expected UserStatus enum")
	}
	if !hasRecord {
		t.Errorf("expected UserDto record")
	}
	if !hasClass {
		t.Errorf("expected UsersController class")
	}
	if !hasCtor {
		t.Errorf("expected UsersController constructor")
	}
	if !hasMethod {
		t.Errorf("expected GetUser method")
	}
}
